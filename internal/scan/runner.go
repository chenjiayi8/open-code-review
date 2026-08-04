package scan

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alibaba/open-code-review/internal/model"
	localrunner "github.com/alibaba/open-code-review/internal/runner"
	"github.com/alibaba/open-code-review/internal/stdout"
	"github.com/alibaba/open-code-review/internal/telemetry"
)

// RunExternal executes full-file scan through one deterministic local runner invocation.
func (a *Agent) RunExternal(ctx context.Context, r *localrunner.Runner, background string) ([]model.LlmComment, error) {
	if r == nil {
		return nil, errors.New("run external runner: nil runner")
	}
	if a.args.Resume != nil {
		if err := a.args.Resume.ValidateScanOptions(a.args.Paths); err != nil {
			return nil, fmt.Errorf("validate scan resume: %w", err)
		}
	}

	ctx, scanSpan := telemetry.StartSpan(ctx, "scan.enumerate")
	provider := NewProvider(a.args.RepoDir, a.args.Paths, a.args.GitRunner, a.args.MaxFileSizeBytes)
	items, err := provider.Enumerate(ctx)
	if err != nil {
		scanSpan.End()
		return nil, fmt.Errorf("enumerate files: %w", err)
	}
	telemetry.SetAttr(scanSpan, "files.enumerated", len(items))
	scanSpan.End()

	a.items = items
	totalDiscovered := len(a.items)
	a.items = a.filterScanItems(a.items)
	a.items = a.filterLargeScans(a.items)

	reviewable := len(a.items)
	fmt.Fprintf(stdout.Writer(), "[ocr] full-scan: %d file(s) discovered, reviewing %d in %s\n",
		totalDiscovered, reviewable, a.args.RepoDir)
	telemetry.RecordFilesReviewed(ctx, int64(reviewable))

	if reviewable == 0 {
		fmt.Fprintln(stdout.Writer(), "[ocr] No reviewable files. Skipping scan.")
		telemetry.Event(ctx, "scan.no.files")
		if ferr := a.session.Finalize(); ferr != nil {
			return []model.LlmComment{}, fmt.Errorf("finalize session: %w", ferr)
		}
		return []model.LlmComment{}, nil
	}

	a.initScanFingerprints(a.items)
	a.initResumeInfo(a.items)

	remaining := make([]model.ScanItem, 0, len(a.items))
	for _, it := range a.items {
		fingerprint := a.scanItemFingerprint(it)
		if item, ok := a.resumeItem(fingerprint); ok {
			for _, cm := range item.Comments {
				a.args.CommentCollector.Add(cm)
			}
			a.session.RecordReviewItemReused(it.Path, it.Path, it.Path, fingerprint, a.args.Resume.SessionID, item.Comments)
			continue
		}
		remaining = append(remaining, it)
	}

	if len(remaining) == 0 {
		comments := a.args.CommentCollector.Comments()
		return comments, a.finalizeExternalScanSession(nil)
	}

	req := localrunner.Request{
		Operation:  localrunner.Review,
		Repository: a.args.RepoDir,
		Background: background,
		Files:      make([]localrunner.File, 0, len(remaining)),
	}
	if req.Background == "" {
		req.Background = a.args.Background
	}
	for _, it := range remaining {
		req.Files = append(req.Files, localrunner.File{Path: it.Path, Rule: a.resolveExternalScanRule(it.Path)})
	}

	result, err := r.Run(ctx, req)
	if err != nil {
		reason := classifyExternalScanRunError(err)
		for _, it := range remaining {
			a.session.RecordReviewItemFailed(it.Path, it.Path, it.Path, a.scanItemFingerprint(it), err.Error())
		}
		finalErr := a.finalizeExternalScanSession(nil)
		return nil, errors.Join(fmt.Errorf("run external runner: %w", err), errors.New(reason), finalErr)
	}

	comments, err := a.validateExternalScanResult(result, remaining)
	if err != nil {
		for _, it := range remaining {
			a.session.RecordReviewItemFailed(it.Path, it.Path, it.Path, a.scanItemFingerprint(it), err.Error())
		}
		finalErr := a.finalizeExternalScanSession(nil)
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
	var missing []string
	for _, it := range remaining {
		fingerprint := a.scanItemFingerprint(it)
		if _, ok := reviewed[it.Path]; ok {
			a.session.RecordReviewItemDone(it.Path, it.Path, it.Path, fingerprint, commentsByPath[it.Path])
			continue
		}
		reason := "runner did not report file as reviewed"
		a.session.RecordReviewItemFailed(it.Path, it.Path, it.Path, fingerprint, reason)
		missing = append(missing, it.Path)
	}

	out := a.args.CommentCollector.Comments()
	if len(out) > 0 {
		telemetry.RecordCommentsGenerated(ctx, int64(len(out)))
	}
	if len(missing) > 0 {
		finalErr := a.finalizeExternalScanSession(nil)
		return out, errors.Join(fmt.Errorf("external runner did not report %d scan file(s) as reviewed: %s", len(missing), strings.Join(missing, ", ")), finalErr)
	}
	return out, a.finalizeExternalScanSession(nil)
}

func (a *Agent) resolveExternalScanRule(path string) string {
	if a.args.SystemRule == nil {
		return ""
	}
	return a.args.SystemRule.Resolve(strings.ToLower(path))
}

func classifyExternalScanRunError(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, localrunner.ErrRunnerTimeout):
		return "external runner timed out"
	case errors.Is(err, context.Canceled):
		return "external runner was cancelled"
	case errors.Is(err, localrunner.ErrExecutableNotFound), errors.Is(err, localrunner.ErrUnauthenticated):
		return "external runner is not configured"
	default:
		return "external runner failed"
	}
}

func (a *Agent) validateExternalScanResult(result localrunner.Result, selected []model.ScanItem) ([]model.LlmComment, error) {
	selectedByPath := make(map[string]model.ScanItem, len(selected))
	for _, it := range selected {
		selectedByPath[it.Path] = it
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
			return nil, fmt.Errorf("external runner reviewed file %q outside selected scan", path)
		}
		reviewed[path] = struct{}{}
	}

	comments := make([]model.LlmComment, 0, len(result.Findings))
	for i, finding := range result.Findings {
		it, ok := selectedByPath[finding.Path]
		if !ok {
			return nil, fmt.Errorf("external runner finding %d path %q outside selected scan", i, finding.Path)
		}
		if _, ok := reviewed[finding.Path]; !ok {
			return nil, fmt.Errorf("external runner finding %d path %q was not reported reviewed", i, finding.Path)
		}
		cm := finding.AsComment()
		if err := validateExternalScanCommentLine(cm, it); err != nil {
			return nil, fmt.Errorf("external runner finding %d: %w", i, err)
		}
		comments = append(comments, cm)
	}
	return comments, nil
}

func validateExternalScanCommentLine(cm model.LlmComment, it model.ScanItem) error {
	if cm.StartLine < 1 {
		return errors.New("start_line must be >= 1")
	}
	if cm.EndLine < cm.StartLine {
		return errors.New("end_line must be >= start_line")
	}
	if it.LineCount == 0 {
		return fmt.Errorf("line range %d-%d exceeds %s line count 0", cm.StartLine, cm.EndLine, it.Path)
	}
	if cm.EndLine > it.LineCount {
		return fmt.Errorf("line range %d-%d exceeds %s line count %d", cm.StartLine, cm.EndLine, it.Path, it.LineCount)
	}
	return nil
}

func (a *Agent) finalizeExternalScanSession(err error) error {
	if ferr := a.session.Finalize(); ferr != nil {
		finalizeErr := fmt.Errorf("finalize session: %w", ferr)
		if err != nil {
			return errors.Join(err, finalizeErr)
		}
		return finalizeErr
	}
	return err
}
