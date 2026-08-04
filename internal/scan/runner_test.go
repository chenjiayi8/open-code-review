package scan

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/alibaba/open-code-review/internal/config/rules"
	"github.com/alibaba/open-code-review/internal/model"
	localrunner "github.com/alibaba/open-code-review/internal/runner"
	"github.com/alibaba/open-code-review/internal/session"
	"github.com/alibaba/open-code-review/internal/tool"
)

type scriptedScanRunnerExecutor struct {
	result localrunner.Result
	err    error
	req    localrunner.Request
	calls  int
}

func (s *scriptedScanRunnerExecutor) Run(_ context.Context, req localrunner.Request) (localrunner.Result, error) {
	s.calls++
	s.req = req
	return s.result, s.err
}

func newExternalScanAgent(t *testing.T, repo string, paths []string) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "main", "local-runner", session.SessionOptions{
		ReviewMode: session.ReviewModeFullScan,
		ScanPaths:  paths,
	})
	return NewAgent(Args{
		RepoDir:          repo,
		Paths:            paths,
		Template:         makeTemplateWithFullScan(),
		Model:            "local-runner",
		CommentCollector: tool.NewCommentCollector(),
		Tools:            tool.NewRegistry(),
		Session:          sess,
		SkipPlan:         true,
		SkipDedup:        true,
		SkipSummary:      true,
	})
}

func TestScanRunExternalRejectsUnenumeratedPath(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "selected/a.go", []byte("package selected\n\nfunc A() {}\n"))
	writeFile(t, repo, "other/b.go", []byte("package other\n\nfunc B() {}\n"))
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"selected/a.go"},
		Findings: []localrunner.Finding{{
			Path:      "other/b.go",
			StartLine: 3,
			EndLine:   3,
			Content:   "outside selected scan path",
			Severity:  model.SeverityLow,
			Category:  model.CategoryOther,
		}},
	}}
	a := newExternalScanAgent(t, repo, []string{"selected"})

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "outside selected scan") {
		t.Fatalf("RunExternal error = %v, want outside selected scan", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if got := a.args.CommentCollector.Comments(); len(got) != 0 {
		t.Fatalf("persisted comments = %+v, want none", got)
	}
	if len(exec.req.Files) != 1 || exec.req.Files[0].Path != "selected/a.go" {
		t.Fatalf("runner files = %+v, want only selected/a.go", exec.req.Files)
	}
}

func TestScanRunExternalPersistsCompletedItems(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "a.go", []byte("package main\n\nfunc A() {}\n"))
	writeFile(t, repo, "b.go", []byte("package main\n\nfunc B() {}\n"))
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go", "b.go"},
		Findings:      []localrunner.Finding{},
	}}
	a := newExternalScanAgent(t, repo, nil)

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "scan background")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if exec.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", exec.calls)
	}
	if len(exec.req.Files) != 2 {
		t.Fatalf("runner files = %+v, want two files", exec.req.Files)
	}
	if a.RunManifest() != nil {
		t.Fatalf("RunManifest = %+v, want nil for scan", a.RunManifest())
	}
	if got := a.FilesReviewed(); got != 2 {
		t.Fatalf("FilesReviewed = %d, want 2", got)
	}

	resume, err := session.LoadResumeState(repo, a.SessionID())
	if err != nil {
		t.Fatalf("LoadResumeState: %v", err)
	}
	if got := resume.CompletedCount(); got != 2 {
		t.Fatalf("CompletedCount = %d, want 2", got)
	}
}

func TestScanRunExternalOmitsExcludedAndBinaryFilesFromManifest(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "a.go", []byte("package main\n\nfunc A() {}\n"))
	writeFile(t, repo, "vendor/skip.go", []byte("package vendor\n\nfunc Skip() {}\n"))
	writeFile(t, repo, "image.png", []byte{'P', 'N', 'G', 0, 'x'})
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings:      []localrunner.Finding{},
	}}
	a := newExternalScanAgent(t, repo, nil)
	a.args.FileFilter = &rules.FileFilter{Exclude: []string{"vendor/**"}}

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(exec.req.Files) != 1 || exec.req.Files[0].Path != "a.go" {
		t.Fatalf("runner files = %+v, want only a.go", exec.req.Files)
	}
	if got := a.FilesReviewed(); got != 1 {
		t.Fatalf("FilesReviewed = %d, want 1 reviewable file", got)
	}
}

func TestScanRunExternalReusesResumeAndRunsRemainingFiles(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "a.go", []byte("package main\n\nfunc A() {}\n"))
	writeFile(t, repo, "b.go", []byte("package main\n\nfunc B() {}\n"))
	gitCommit(t, repo, "init")

	seedExec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go", "b.go"},
		Findings: []localrunner.Finding{{
			Path:      "a.go",
			StartLine: 3,
			EndLine:   3,
			Content:   "cached finding",
			Severity:  model.SeverityLow,
			Category:  model.CategoryOther,
		}},
	}}
	seed := newExternalScanAgent(t, repo, nil)
	if _, err := seed.RunExternal(context.Background(), localrunner.NewWithExecutor(seedExec), ""); err != nil {
		t.Fatalf("seed RunExternal: %v", err)
	}
	resume, err := session.LoadResumeState(repo, seed.SessionID())
	if err != nil {
		t.Fatalf("LoadResumeState: %v", err)
	}

	writeFile(t, repo, "b.go", []byte("package main\n\nfunc B() string { return \"changed\" }\n"))
	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"b.go"},
		Findings:      []localrunner.Finding{},
	}}
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "main", "local-runner", session.SessionOptions{
		ReviewMode:  session.ReviewModeFullScan,
		ResumedFrom: resume.SessionID,
	})
	a := NewAgent(Args{
		RepoDir:          repo,
		Template:         makeTemplateWithFullScan(),
		Model:            "local-runner",
		CommentCollector: tool.NewCommentCollector(),
		Tools:            tool.NewRegistry(),
		Session:          sess,
		Resume:           resume,
		SkipPlan:         true,
		SkipDedup:        true,
		SkipSummary:      true,
	})

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 1 || comments[0].Path != "a.go" || comments[0].Content != "cached finding" {
		t.Fatalf("comments = %+v, want cached a.go finding", comments)
	}
	if len(exec.req.Files) != 1 || exec.req.Files[0].Path != "b.go" {
		t.Fatalf("runner files = %+v, want only changed b.go", exec.req.Files)
	}
	info := a.ResumeInfo()
	if info == nil || info.ResumedFrom != resume.SessionID || info.ReusedFiles != 1 || info.RerunFiles != 1 {
		t.Fatalf("ResumeInfo = %+v, want one reused and one rerun", info)
	}
}

func TestScanRunExternalRejectsMissingCoverage(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "a.go", []byte("package main\n\nfunc A() {}\n"))
	writeFile(t, repo, "b.go", []byte("package main\n\nfunc B() {}\n"))
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings:      []localrunner.Finding{},
	}}
	a := newExternalScanAgent(t, repo, nil)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "did not report 1 scan file") {
		t.Fatalf("RunExternal error = %v, want missing coverage", err)
	}
	resume, loadErr := session.LoadResumeState(repo, a.SessionID())
	if loadErr != nil {
		t.Fatalf("LoadResumeState: %v", loadErr)
	}
	if got := resume.CompletedCount(); got != 1 {
		t.Fatalf("CompletedCount = %d, want only the reported file reusable", got)
	}
}

func TestScanRunExternalRejectsMalformedLineNumbers(t *testing.T) {
	repo := initTestRepo(t)
	writeFile(t, repo, "a.go", []byte("package main\n\nfunc A() {}\n"))
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings: []localrunner.Finding{{
			Path:      "a.go",
			StartLine: 3,
			EndLine:   99,
			Content:   "bad line range",
			Severity:  model.SeverityLow,
			Category:  model.CategoryOther,
		}},
	}}
	a := newExternalScanAgent(t, repo, nil)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "line range") {
		t.Fatalf("RunExternal error = %v, want line range rejection", err)
	}
	if got := a.args.CommentCollector.Comments(); len(got) != 0 {
		t.Fatalf("persisted comments = %+v, want none", got)
	}
}

func TestScanRunExternalRunnerFailureLeavesSourceFilesUnmodified(t *testing.T) {
	repo := initTestRepo(t)
	original := []byte("package main\n\nfunc A() {}\n")
	writeFile(t, repo, "a.go", original)
	gitCommit(t, repo, "init")

	exec := &scriptedScanRunnerExecutor{err: errors.New("boom")}
	a := newExternalScanAgent(t, repo, nil)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "run external runner") {
		t.Fatalf("RunExternal error = %v, want runner failure", err)
	}
	got, readErr := os.ReadFile(repo + "/a.go")
	if readErr != nil {
		t.Fatalf("read source file: %v", readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("source file changed:\n%s", got)
	}
	if exec.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", exec.calls)
	}
}
