package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alibaba/open-code-review/internal/config/template"
	"github.com/alibaba/open-code-review/internal/scan"
	"github.com/alibaba/open-code-review/internal/session"
	"github.com/alibaba/open-code-review/internal/telemetry"
	"github.com/spf13/cobra"

	"go.opentelemetry.io/otel/codes"
)

type scanOptions struct {
	rulePath      string
	repoDir       string
	paths         string
	excludes      string
	outputFormat  string
	audience      string
	background    string
	maxGitProcs   int
	preview       bool
	noPlan        bool
	noDedup       bool
	noSummary     bool
	batch         string
	runner        string
	runnerModel   string
	runnerTimeout int
	resume        string
}

var scanOpts scanOptions

var scanCmd = &cobra.Command{
	Use:     "scan [flags]",
	Aliases: []string{"s"},
	Short:   "Scan entire files (no diff required)",
	Long:    "OpenCodeReview - Full-File Scan\n\nScan entire files for code review without requiring a diff.",
	Args:    cobra.NoArgs,
	Example: `  # Scan the entire repository
  ocr scan --runner codex

  # Scan a single directory
  ocr scan --runner codex --path internal/agent

  # Scan multiple files
  ocr scan --runner codex --path internal/agent/agent.go,internal/diff/scan.go

  # Select an authenticated local CLI runner with native auth and optional model for this run
  ocr scan --runner claude --runner-model sonnet --format json

  # Exclude generated files / fixtures
  ocr scan --runner codex --exclude '**/generated/*,**/testdata/*'

  # Preview which files would be scanned without calling a local runner
  ocr scan --preview

  # Skip the per-file PLAN_TASK pre-pass
  ocr scan --runner codex --no-plan

  # Resume a previous full-file scan
  ocr scan --runner codex --resume <session-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateScanOptions(&scanOpts); err != nil {
			return err
		}
		return executeScan(scanOpts)
	},
}

func init() {
	registerScanFlags(scanCmd, &scanOpts)
}

func splitPaths(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func executeScan(opts scanOptions) error {
	cc, err := loadCommonContext(opts.repoDir, opts.rulePath, 0, opts.maxGitProcs, false)
	if err != nil {
		return err
	}
	applyCLIExcludes(cc, splitPaths(opts.excludes))

	// scan owns its own template (scan_template.json) independent from the
	// diff-review template loaded by loadCommonContext above.
	scanTpl, err := template.LoadScanDefault()
	if err != nil {
		return fmt.Errorf("load scan template: %w", err)
	}
	if err := scanTpl.Validate(); err != nil {
		return fmt.Errorf("invalid scan template: %w", err)
	}
	if opts.batch != "" {
		// CLI override of BATCH_STRATEGY; validated downstream by parseBatchStrategy
		// (unknown values silently fall back to "none").
		scanTpl.BatchStrategy = opts.batch
	}
	scanPaths := splitPaths(opts.paths)

	if opts.preview {
		return runScanPreview(cc, scanTpl, scanPaths)
	}

	resumeState, err := loadScanResumeState(cc.RepoDir, opts, scanPaths)
	if err != nil {
		return err
	}

	rt, err := loadRunnerRuntime(cc.Template, opts.runner, opts.runnerModel, opts.runnerTimeout)
	if err != nil {
		return err
	}
	llmIdentity := &jsonLLMIdentity{
		Provider: "local-runner",
		Runner:   rt.RunnerKind,
		Model:    rt.RunnerModel,
	}
	if rt.AppCfg != nil {
		scanTpl.ApplyLanguage(rt.AppCfg.Language)
	}

	ag := scan.NewAgent(scan.Args{
		RepoDir:          cc.RepoDir,
		Paths:            scanPaths,
		Template:         *scanTpl,
		SystemRule:       cc.Resolver,
		FileFilter:       cc.FileFilter,
		CommentCollector: rt.Collector,
		Model:            rt.RunnerModel,
		Background:       opts.background,
		GitRunner:        cc.GitRunner,
		MaxFileSizeBytes: scanTpl.MaxFileSizeBytes,
		SkipPlan:         opts.noPlan,
		SkipDedup:        opts.noDedup,
		SkipSummary:      opts.noSummary,
		Resume:           resumeState,
	})

	q := newQuietHandle(opts.outputFormat, opts.audience)
	defer q.Restore()

	ctx, span := telemetry.StartSpan(telemetry.ContextWithTraceParentFromEnv(context.Background()), "scan.run")
	defer span.End()
	var traceID string
	if telemetry.IsEnabled() {
		traceID = telemetry.TraceIDFromContext(ctx)
		if opts.outputFormat != "json" {
			fmt.Fprintf(os.Stderr, "[ocr] TraceID: %s\n", traceID)
		}
	}
	startTime := time.Now()

	comments, err := ag.RunExternal(ctx, rt.Runner, opts.background)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		if id := ag.SessionID(); id != "" {
			fmt.Fprintf(os.Stderr, "[ocr] Session: %s (retry with: --resume %s)\n", id, id)
		}
		return fmt.Errorf("scan failed: %w", err)
	}

	return emitRunResult(ctx, ag, comments, startTime, opts.outputFormat, opts.audience, q, llmIdentity)
}

func loadScanResumeState(repoDir string, opts scanOptions, scanPaths []string) (*session.ResumeState, error) {
	if opts.resume == "" {
		return nil, nil
	}
	state, err := session.LoadResumeState(repoDir, opts.resume)
	if err != nil {
		return nil, fmt.Errorf("load resume session: %w (run 'ocr session list' to see available sessions)", err)
	}
	if err := state.ValidateScanOptions(scanPaths); err != nil {
		return nil, fmt.Errorf("%w (run 'ocr session list' to see available sessions)", err)
	}
	if state.CompletedCount() == 0 {
		return nil, fmt.Errorf("resume session %q has no completed scan items (run 'ocr session list' to see available sessions)", opts.resume)
	}
	return state, nil
}

func runScanPreview(cc *commonContext, scanTpl *template.ScanTemplate, scanPaths []string) error {
	ag := scan.NewAgent(scan.Args{
		RepoDir:          cc.RepoDir,
		Paths:            scanPaths,
		FileFilter:       cc.FileFilter,
		GitRunner:        cc.GitRunner,
		MaxFileSizeBytes: scanTpl.MaxFileSizeBytes,
		// Template's prompt fields are unused by Preview; pass the same
		// value so MaxFileSizeBytes is consistent.
		Template: *scanTpl,
	})

	preview, err := ag.Preview(context.Background())
	if err != nil {
		return fmt.Errorf("scan preview failed: %w", err)
	}
	outputPreviewText(preview)
	return nil
}
