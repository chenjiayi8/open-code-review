package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alibaba/open-code-review/internal/diff"
	"github.com/alibaba/open-code-review/internal/model"
	localrunner "github.com/alibaba/open-code-review/internal/runner"
	"github.com/alibaba/open-code-review/internal/session"
	"github.com/alibaba/open-code-review/internal/telemetry"
)

// RunExternal executes the diff review through one deterministic local runner invocation.
func (a *Agent) RunExternal(ctx context.Context, r *localrunner.Runner, background string) ([]model.LlmComment, error) {
	if r == nil {
		return nil, errors.New("run external runner: nil runner")
	}
	prep, err := a.prepareReview(ctx)
	if err != nil {
		return nil, a.finalizeReviewSessionWithError(err)
	}
	if prep.skipped {
		return []model.LlmComment{}, a.finalizeReviewSession()
	}

	if err := a.registerCoverage(a.diffs); err != nil {
		a.recordWarning("manifest_error", "", err.Error())
		if b := a.session.Manifest(); b != nil {
			_ = b.SetRunFailure(session.RunFailureInternal, "coverage registration failed")
		}
	}

	toRun := a.applyResume(a.diffs)
	remaining := make([]model.Diff, 0, len(toRun))
	for _, d := range toRun {
		if !d.IsDeleted {
			remaining = append(remaining, d)
		}
	}
	if len(remaining) == 0 {
		comments := a.args.CommentCollector.Comments()
		return comments, a.finalizeReviewSession()
	}

	req := localrunner.Request{
		Operation:  localrunner.Review,
		Repository: a.args.RepoDir,
		Review:     a.runnerReviewContext(),
		Background: background,
		Files:      make([]localrunner.File, 0, len(remaining)),
	}
	if req.Background == "" {
		req.Background = a.args.Background
	}
	for _, d := range remaining {
		path := effectivePath(d)
		req.Files = append(req.Files, localrunner.File{
			Path:          path,
			Rule:          a.resolveSystemRule(strings.ToLower(path)),
			OldPath:       d.OldPath,
			NewPath:       d.NewPath,
			ChangedRanges: changedRangesFromDiff(d),
			UnifiedDiff:   d.Diff,
		})
	}

	result, err := r.Run(ctx, req)
	if err != nil {
		class, itemClass, reason := classifyExternalRunError(err)
		if b := a.session.Manifest(); b != nil {
			if e := b.SetRunFailure(class, reason); e != nil {
				a.recordWarning("manifest_error", "", e.Error())
			}
		}
		for _, d := range remaining {
			a.markFailed(d, itemClass, reason)
			a.session.RecordReviewItemFailed(effectivePath(d), d.OldPath, d.NewPath, reviewItemFingerprint(a.reviewMode(), d), err.Error())
		}
		finalErr := a.finalizeReviewSession()
		return nil, errors.Join(fmt.Errorf("run external runner: %w", err), finalErr)
	}

	comments, err := a.validateExternalResult(result, remaining)
	if err != nil {
		if b := a.session.Manifest(); b != nil {
			if e := b.SetRunFailure(session.RunFailureUnknown, "external runner returned invalid review result"); e != nil {
				a.recordWarning("manifest_error", "", e.Error())
			}
		}
		for _, d := range remaining {
			a.markFailed(d, session.FailureUnknown, "external runner returned invalid review result")
			a.session.RecordReviewItemFailed(effectivePath(d), d.OldPath, d.NewPath, reviewItemFingerprint(a.reviewMode(), d), err.Error())
		}
		finalErr := a.finalizeReviewSession()
		return nil, errors.Join(err, finalErr)
	}

	commentsByPath := make(map[string][]model.LlmComment)
	for _, cm := range comments {
		commentsByPath[cm.Path] = append(commentsByPath[cm.Path], cm)
		a.args.CommentCollector.Add(cm)
	}
	reviewed := make(map[string]struct{}, len(result.ReviewedFiles))
	for _, path := range result.ReviewedFiles {
		reviewed[path] = struct{}{}
	}
	for _, d := range remaining {
		path := effectivePath(d)
		fingerprint := reviewItemFingerprint(a.reviewMode(), d)
		if _, ok := reviewed[path]; ok {
			a.markCompleted(d)
			a.session.RecordReviewItemDone(path, d.OldPath, d.NewPath, fingerprint, commentsByPath[path])
			continue
		}
		reason := "runner did not report file as reviewed"
		a.markFailed(d, session.FailureUnknown, reason)
		a.session.RecordReviewItemFailed(path, d.OldPath, d.NewPath, fingerprint, reason)
	}

	out := a.args.CommentCollector.Comments()
	if len(out) > 0 {
		telemetry.RecordCommentsGenerated(ctx, int64(len(out)))
	}
	return out, a.finalizeReviewSession()
}

func classifyExternalRunError(err error) (session.RunFailureClass, session.FailureClass, string) {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, localrunner.ErrRunnerTimeout):
		return session.RunFailureTimeout, session.FailureTimeout, "external runner timed out"
	case errors.Is(err, context.Canceled):
		return session.RunFailureCancelled, session.FailureCancelled, "external runner was cancelled"
	case errors.Is(err, localrunner.ErrExecutableNotFound), errors.Is(err, localrunner.ErrUnauthenticated):
		return session.RunFailureConfiguration, session.FailureConfiguration, "external runner is not configured"
	default:
		return session.RunFailureUnknown, session.FailureUnknown, "external runner failed"
	}
}

func (a *Agent) validateExternalResult(result localrunner.Result, selected []model.Diff) ([]model.LlmComment, error) {
	selectedByPath := make(map[string]model.Diff, len(selected))
	for _, d := range selected {
		selectedByPath[effectivePath(d)] = d
	}
	reviewed := make(map[string]struct{}, len(result.ReviewedFiles))
	for _, path := range result.ReviewedFiles {
		if path == "" {
			return nil, errors.New("external runner result has empty reviewed file")
		}
		if _, dup := reviewed[path]; dup {
			return nil, fmt.Errorf("external runner result has duplicate reviewed file %q", path)
		}
		if _, ok := selectedByPath[path]; !ok {
			return nil, fmt.Errorf("external runner reviewed file %q outside selected diff", path)
		}
		reviewed[path] = struct{}{}
	}

	comments := make([]model.LlmComment, 0, len(result.Findings))
	for i, finding := range result.Findings {
		if _, ok := selectedByPath[finding.Path]; !ok {
			return nil, fmt.Errorf("external runner finding %d path %q outside selected diff", i, finding.Path)
		}
		if _, ok := reviewed[finding.Path]; !ok {
			return nil, fmt.Errorf("external runner finding %d path %q was not reported reviewed", i, finding.Path)
		}
		cm := finding.AsComment()
		resolved := diff.ResolveLineNumbers([]model.LlmComment{cm}, selected)
		cm = resolved[0]
		d := selectedByPath[finding.Path]
		if err := validateExternalCommentLine(cm, d); err != nil {
			return nil, fmt.Errorf("external runner finding %d: %w", i, err)
		}
		if err := a.validateExternalFindingScope(cm, d); err != nil {
			return nil, fmt.Errorf("external runner finding %d: %w", i, err)
		}
		comments = append(comments, cm)
	}
	return comments, nil
}

func (a *Agent) validateExternalFindingScope(cm model.LlmComment, d model.Diff) error {
	mode := a.reviewMode()
	if mode == session.ReviewModeWorkspace {
		return nil
	}
	ranges := changedRangesFromDiff(d)
	if len(ranges) == 0 {
		return nil
	}
	for _, r := range ranges {
		if r.NewStart < 1 || r.NewEnd < r.NewStart {
			continue
		}
		if cm.StartLine >= r.NewStart && cm.EndLine <= r.NewEnd {
			return nil
		}
	}
	return fmt.Errorf("line range %d-%d is outside changed ranges for %s", cm.StartLine, cm.EndLine, effectivePath(d))
}

func validateExternalCommentLine(cm model.LlmComment, d model.Diff) error {
	if cm.StartLine < 1 {
		return errors.New("start_line must be >= 1")
	}
	if cm.EndLine < cm.StartLine {
		return errors.New("end_line must be >= start_line")
	}
	if d.NewFileContent != "" {
		lineCount := strings.Count(d.NewFileContent, "\n")
		if !strings.HasSuffix(d.NewFileContent, "\n") {
			lineCount++
		}
		if lineCount == 0 {
			lineCount = 1
		}
		if cm.EndLine > lineCount {
			return fmt.Errorf("line range %d-%d exceeds %s line count %d", cm.StartLine, cm.EndLine, effectivePath(d), lineCount)
		}
	}
	return nil
}

func (a *Agent) runnerReviewContext() localrunner.ReviewContext {
	ctx := localrunner.ReviewContext{
		Mode:         a.reviewMode(),
		ResolvedBase: a.inputResolution.ResolvedBase,
		ResolvedHead: a.inputResolution.ResolvedHead,
		ExactRange:   a.inputResolution.ExactRange,
	}
	switch ctx.Mode {
	case session.ReviewModeRange:
		ctx.RequestedFrom = a.args.From
		ctx.RequestedHead = a.args.To
	case session.ReviewModeCommit:
		ctx.RequestedHead = a.args.Commit
	}
	return ctx
}

func changedRangesFromDiff(d model.Diff) []localrunner.ChangedRange {
	hunks := diff.ParseHunks(d.Diff)
	ranges := make([]localrunner.ChangedRange, 0, len(hunks))
	for _, h := range hunks {
		ranges = append(ranges, localrunner.ChangedRange{
			OldStart: h.OldStart,
			OldEnd:   rangeEnd(h.OldStart, h.OldCount),
			NewStart: h.NewStart,
			NewEnd:   rangeEnd(h.NewStart, h.NewCount),
		})
	}
	return ranges
}

func rangeEnd(start, count int) int {
	if count <= 0 {
		return 0
	}
	return start + count - 1
}
