package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alibaba/open-code-review/internal/config/template"
	"github.com/alibaba/open-code-review/internal/model"
	localrunner "github.com/alibaba/open-code-review/internal/runner"
	"github.com/alibaba/open-code-review/internal/session"
)

type scriptedRunnerExecutor struct {
	result localrunner.Result
	err    error
	req    localrunner.Request
	calls  int
}

func (s *scriptedRunnerExecutor) Run(_ context.Context, req localrunner.Request) (localrunner.Result, error) {
	s.calls++
	s.req = req
	return s.result, s.err
}

func initExternalRunnerRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	runAgentGit(t, repo, "init", "-q")
	runAgentGit(t, repo, "config", "user.email", "test@example.com")
	runAgentGit(t, repo, "config", "user.name", "Test User")
	runAgentGit(t, repo, "config", "commit.gpgsign", "false")
	for path, content := range files {
		writeAgentFile(t, repo, path, content)
	}
	runAgentGit(t, repo, "add", ".")
	runAgentGit(t, repo, "commit", "-q", "-m", "initial")
	return repo
}

func runAgentGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeAgentFile(t *testing.T, repo, path, content string) {
	t.Helper()
	full := filepath.Join(repo, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newExternalAgent(t *testing.T, repo string) *Agent {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "feature", "local-runner", session.SessionOptions{
		ReviewMode: session.ReviewModeWorkspace,
		Operation:  session.OperationReview,
	})
	return New(Args{RepoDir: repo, Model: "local-runner", Session: sess})
}

func TestRunExternalMarksOnlyReturnedFilesCompleted(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
		"b.go": "package main\n\nfunc b() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	writeAgentFile(t, repo, "b.go", "package main\n\nfunc b() string { return \"new\" }\n")

	exec := &scriptedRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings:      []localrunner.Finding{},
	}}
	r := localrunner.NewWithExecutor(exec)
	a := newExternalAgent(t, repo)

	comments, err := a.RunExternal(context.Background(), r, "background")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	manifest := a.RunManifest()
	if manifest == nil {
		t.Fatal("RunManifest is nil")
	}
	if got := len(manifest.Coverage.Completed); got != 1 {
		t.Fatalf("completed count = %d, want 1; manifest=%+v", got, manifest.Coverage)
	}
	if manifest.Coverage.Completed[0].Path != "a.go" {
		t.Fatalf("completed path = %q, want a.go", manifest.Coverage.Completed[0].Path)
	}
	if got := len(manifest.Coverage.Failed); got != 1 {
		t.Fatalf("failed count = %d, want 1; manifest=%+v", got, manifest.Coverage)
	}
	if manifest.Coverage.Failed[0].Path != "b.go" || !strings.Contains(manifest.Coverage.Failed[0].Reason, "runner did not report file as reviewed") {
		t.Fatalf("failed item = %+v, want b.go runner coverage failure", manifest.Coverage.Failed[0])
	}
	if exec.calls != 1 {
		t.Fatalf("executor calls = %d, want 1", exec.calls)
	}
}

func TestRunExternalRejectsFindingOutsideSelectedDiff(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")

	exec := &scriptedRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings: []localrunner.Finding{{
			Path:      "secret.txt",
			StartLine: 1,
			EndLine:   1,
			Content:   "do not review this",
			Severity:  model.SeverityLow,
			Category:  model.CategoryOther,
		}},
	}}
	r := localrunner.NewWithExecutor(exec)
	a := newExternalAgent(t, repo)

	comments, err := a.RunExternal(context.Background(), r, "")
	if err == nil || !strings.Contains(err.Error(), "outside selected diff") {
		t.Fatalf("RunExternal error = %v, want outside selected diff", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if got := a.args.CommentCollector.Comments(); len(got) != 0 {
		t.Fatalf("persisted comments = %+v, want none", got)
	}
	if manifest := a.RunManifest(); manifest == nil || manifest.RunFailure == nil {
		t.Fatalf("RunManifest = %+v, want persisted failed manifest", manifest)
	}
}

func TestRunExternalRunnerFailurePreservesSessionEndManifest(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	exec := &scriptedRunnerExecutor{err: errors.New("boom")}
	a := newExternalAgent(t, repo)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "run external runner") {
		t.Fatalf("RunExternal error = %v, want runner failure", err)
	}
	manifest := a.RunManifest()
	if manifest == nil || manifest.RunFailure == nil || manifest.RunFailure.Classification != session.RunFailureUnknown {
		t.Fatalf("manifest = %+v, want unknown run failure", manifest)
	}
	if got := len(manifest.Coverage.Failed); got != 1 {
		t.Fatalf("failed coverage = %d, want 1", got)
	}
}

func TestRunExternalNoSelectedFilesSkipsWithoutCallingRunner(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"removed.go": "package main\n",
	})
	if err := os.Remove(filepath.Join(repo, "removed.go")); err != nil {
		t.Fatalf("remove changed file: %v", err)
	}
	exec := &scriptedRunnerExecutor{}
	a := newExternalAgent(t, repo)

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if exec.calls != 0 {
		t.Fatalf("executor calls = %d, want 0", exec.calls)
	}
	manifest := a.RunManifest()
	if manifest == nil || manifest.TerminalState != session.StateSkipped || len(manifest.Coverage.Selected) != 0 {
		t.Fatalf("manifest = %+v, want skipped empty coverage", manifest)
	}
}

func TestRunExternalValidZeroFindingResponseCoversAllFiles(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
		"b.go": "package main\n\nfunc b() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	writeAgentFile(t, repo, "b.go", "package main\n\nfunc b() string { return \"new\" }\n")
	exec := &scriptedRunnerExecutor{result: localrunner.Result{ReviewedFiles: []string{"a.go", "b.go"}, Findings: []localrunner.Finding{}}}
	a := newExternalAgent(t, repo)

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	manifest := a.RunManifest()
	if manifest == nil || manifest.TerminalState != session.StateComplete || len(manifest.Coverage.Completed) != 2 || len(manifest.Coverage.Failed) != 0 {
		t.Fatalf("manifest = %+v, want complete coverage for both files", manifest)
	}
}

func TestRunExternalReusesResumeAndRunsOneRemainingFile(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
		"b.go": "package main\n\nfunc b() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	writeAgentFile(t, repo, "b.go", "package main\n\nfunc b() string { return \"new\" }\n")

	seed := newExternalAgent(t, repo)
	if err := seed.loadDiffs(context.Background()); err != nil {
		t.Fatalf("seed loadDiffs: %v", err)
	}
	seed.diffs = seed.filterDiffs(seed.diffs)
	var aDiff model.Diff
	for _, d := range seed.diffs {
		if d.NewPath == "a.go" {
			aDiff = d
		}
	}
	if aDiff.NewPath == "" {
		t.Fatal("seed diff for a.go not found")
	}
	previous := session.New(repo, "feature", "previous-model", session.SessionOptions{ReviewMode: session.ReviewModeWorkspace})
	cached := []model.LlmComment{{Path: "a.go", Content: "cached", StartLine: 3, EndLine: 3, Severity: model.SeverityLow, Category: model.CategoryOther}}
	previous.RecordReviewItemDone("a.go", "a.go", "a.go", reviewItemFingerprint(session.ReviewModeWorkspace, aDiff), cached)
	if err := previous.Finalize(); err != nil {
		t.Fatalf("finalize previous session: %v", err)
	}
	resume, err := session.LoadResumeState(repo, previous.SessionID)
	if err != nil {
		t.Fatalf("LoadResumeState: %v", err)
	}

	exec := &scriptedRunnerExecutor{result: localrunner.Result{ReviewedFiles: []string{"b.go"}, Findings: []localrunner.Finding{}}}
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "feature", "current-model", session.SessionOptions{
		ReviewMode:  session.ReviewModeWorkspace,
		ResumedFrom: previous.SessionID,
		Operation:   session.OperationReview,
	})
	a := New(Args{RepoDir: repo, Model: "current-model", Session: sess, Resume: resume})

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 1 || comments[0].Content != "cached" {
		t.Fatalf("comments = %+v, want cached resume comment only", comments)
	}
	if len(exec.req.Files) != 1 || exec.req.Files[0].Path != "b.go" {
		t.Fatalf("runner files = %+v, want only b.go", exec.req.Files)
	}
	info := a.ResumeInfo()
	if info == nil || info.ReusedFiles != 1 || info.RerunFiles != 1 {
		t.Fatalf("ResumeInfo = %+v, want 1 reused and 1 rerun", info)
	}
	manifest := a.RunManifest()
	if manifest == nil || len(manifest.Coverage.Reused) != 1 || manifest.Coverage.Reused[0].Path != "a.go" || len(manifest.Coverage.Completed) != 1 || manifest.Coverage.Completed[0].Path != "b.go" {
		t.Fatalf("manifest coverage = %+v, want a.go reused and b.go completed", manifest.Coverage)
	}
}

func TestRunExternalRejectsFindingOutsideChangedRanges(t *testing.T) {
	original := strings.Join([]string{
		"package main",
		"",
		"func header() string {",
		"	return \"same\"",
		"}",
		"",
		"func spacerOne() string {",
		"	return \"same\"",
		"}",
		"",
		"func spacerTwo() string {",
		"	return \"same\"",
		"}",
		"",
		"func target() string {",
		"	return \"old\"",
		"}",
		"",
	}, "\n")
	changed := strings.Replace(original, "return \"old\"", "value := \"new\"\n\treturn value", 1)
	repo := initExternalRunnerRepo(t, map[string]string{"a.go": original})
	writeAgentFile(t, repo, "a.go", changed)
	runAgentGit(t, repo, "add", "a.go")
	runAgentGit(t, repo, "commit", "-q", "-m", "change a")
	head := strings.TrimSpace(agentGitOutput(t, repo, "rev-parse", "HEAD"))
	exec := &scriptedRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings: []localrunner.Finding{{
			Path:      "a.go",
			StartLine: 1,
			EndLine:   1,
			Content:   "outside the changed hunk",
			Severity:  model.SeverityLow,
			Category:  model.CategoryOther,
		}},
	}}
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "feature", "local-runner", session.SessionOptions{ReviewMode: session.ReviewModeCommit, Operation: session.OperationReview})
	a := New(Args{RepoDir: repo, Model: "local-runner", Session: sess, Commit: head, ReviewMode: session.ReviewModeCommit})

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "outside changed ranges") {
		t.Fatalf("RunExternal error = %v, want outside changed ranges", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if got := a.args.CommentCollector.Comments(); len(got) != 0 {
		t.Fatalf("persisted comments = %+v, want none", got)
	}
}

func TestRunExternalRejectsInvalidLineRange(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	exec := &scriptedRunnerExecutor{result: localrunner.Result{
		ReviewedFiles: []string{"a.go"},
		Findings:      []localrunner.Finding{{Path: "a.go", StartLine: 3, EndLine: 99, Content: "bad range", Severity: model.SeverityLow, Category: model.CategoryOther}},
	}}
	a := newExternalAgent(t, repo)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "line range") {
		t.Fatalf("RunExternal error = %v, want invalid line range", err)
	}
	if got := a.args.CommentCollector.Comments(); len(got) != 0 {
		t.Fatalf("persisted comments = %+v, want none", got)
	}
}

func TestRunExternalRejectsDuplicateReviewedFiles(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"a.go": "package main\n\nfunc a() string { return \"old\" }\n",
	})
	writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string { return \"new\" }\n")
	exec := &scriptedRunnerExecutor{result: localrunner.Result{ReviewedFiles: []string{"a.go", "a.go"}, Findings: []localrunner.Finding{}}}
	a := newExternalAgent(t, repo)

	_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err == nil || !strings.Contains(err.Error(), "duplicate reviewed file") {
		t.Fatalf("RunExternal error = %v, want duplicate reviewed file", err)
	}
}

func TestRunExternalSkipsLargeDiffBeforeInvokingRunner(t *testing.T) {
	repo := initExternalRunnerRepo(t, map[string]string{
		"large.go": "package main\n\nfunc old() {}\n",
	})
	writeAgentFile(t, repo, "large.go", "package main\n"+strings.Repeat("// changed line with enough tokens to exceed the limit\n", 200))
	exec := &scriptedRunnerExecutor{result: localrunner.Result{ReviewedFiles: []string{"large.go"}, Findings: []localrunner.Finding{}}}
	t.Setenv("HOME", t.TempDir())
	sess := session.New(repo, "feature", "local-runner", session.SessionOptions{
		ReviewMode: session.ReviewModeWorkspace,
		Operation:  session.OperationReview,
	})
	a := New(Args{RepoDir: repo, Model: "local-runner", Session: sess, Template: template.Template{MaxTokens: 100}})

	comments, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
	if err != nil {
		t.Fatalf("RunExternal: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("comments = %+v, want none", comments)
	}
	if exec.calls != 0 {
		t.Fatalf("executor calls = %d, want 0 for large diff excluded before runner", exec.calls)
	}
	manifest := a.RunManifest()
	if manifest == nil || manifest.TerminalState != session.StateSkipped || len(manifest.Coverage.Selected) != 0 {
		t.Fatalf("manifest = %+v, want skipped empty coverage", manifest)
	}
}

func TestRunExternalBuildsStructuredDiffContextForReviewModes(t *testing.T) {
	cases := []struct {
		name string
		mode string
		args func(t *testing.T, repo string, base, head string, sess *session.SessionHistory) Args
		want func(t *testing.T, req localrunner.Request, base, head string)
	}{
		{
			name: "workspace",
			mode: session.ReviewModeWorkspace,
			args: func(t *testing.T, repo string, base, head string, sess *session.SessionHistory) Args {
				return Args{RepoDir: repo, Model: "local-runner", Session: sess, ReviewMode: session.ReviewModeWorkspace}
			},
			want: func(t *testing.T, req localrunner.Request, base, head string) {
				t.Helper()
				if req.Review.Mode != session.ReviewModeWorkspace || req.Review.RequestedFrom != "" || req.Review.RequestedHead != "" || req.Review.ResolvedBase != base || req.Review.ResolvedHead != "" || req.Review.ExactRange != "" {
					t.Fatalf("workspace review context = %+v, want mode/base only", req.Review)
				}
			},
		},
		{
			name: "range",
			mode: session.ReviewModeRange,
			args: func(t *testing.T, repo string, base, head string, sess *session.SessionHistory) Args {
				return Args{RepoDir: repo, Model: "local-runner", Session: sess, From: base, To: head, ReviewMode: session.ReviewModeRange}
			},
			want: func(t *testing.T, req localrunner.Request, base, head string) {
				t.Helper()
				if req.Review.Mode != session.ReviewModeRange || req.Review.RequestedFrom != base || req.Review.RequestedHead != head || req.Review.ResolvedBase != base || req.Review.ResolvedHead != head || req.Review.ExactRange != base+".."+head {
					t.Fatalf("range review context = %+v, want requested/resolved range", req.Review)
				}
			},
		},
		{
			name: "commit",
			mode: session.ReviewModeCommit,
			args: func(t *testing.T, repo string, base, head string, sess *session.SessionHistory) Args {
				return Args{RepoDir: repo, Model: "local-runner", Session: sess, Commit: head, ReviewMode: session.ReviewModeCommit}
			},
			want: func(t *testing.T, req localrunner.Request, base, head string) {
				t.Helper()
				if req.Review.Mode != session.ReviewModeCommit || req.Review.RequestedFrom != "" || req.Review.RequestedHead != head || req.Review.ResolvedBase != base || req.Review.ResolvedHead != head || req.Review.ExactRange != base+".."+head {
					t.Fatalf("commit review context = %+v, want requested commit and resolved parent range", req.Review)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := initExternalRunnerRepo(t, map[string]string{
				"a.go": "package main\n\nfunc a() string {\n\treturn \"old\"\n}\n",
			})
			base := strings.TrimSpace(agentGitOutput(t, repo, "rev-parse", "HEAD"))
			writeAgentFile(t, repo, "a.go", "package main\n\nfunc a() string {\n\tvalue := \"new\"\n\treturn value\n}\n")
			head := ""
			if tc.mode != session.ReviewModeWorkspace {
				runAgentGit(t, repo, "add", "a.go")
				runAgentGit(t, repo, "commit", "-q", "-m", "change a")
				head = strings.TrimSpace(agentGitOutput(t, repo, "rev-parse", "HEAD"))
			}
			t.Setenv("HOME", t.TempDir())
			sess := session.New(repo, "feature", "local-runner", session.SessionOptions{ReviewMode: tc.mode, Operation: session.OperationReview})
			exec := &scriptedRunnerExecutor{result: localrunner.Result{ReviewedFiles: []string{"a.go"}, Findings: []localrunner.Finding{}}}
			a := New(tc.args(t, repo, base, head, sess))

			_, err := a.RunExternal(context.Background(), localrunner.NewWithExecutor(exec), "")
			if err != nil {
				t.Fatalf("RunExternal: %v", err)
			}
			if exec.calls != 1 {
				t.Fatalf("executor calls = %d, want 1", exec.calls)
			}
			tc.want(t, exec.req, base, head)
			if len(exec.req.Files) != 1 {
				t.Fatalf("runner files = %+v, want one file", exec.req.Files)
			}
			file := exec.req.Files[0]
			if file.Path != "a.go" || file.OldPath != "a.go" || file.NewPath != "a.go" {
				t.Fatalf("runner file identity = %+v, want a.go old/new path", file)
			}
			if len(file.ChangedRanges) == 0 || file.ChangedRanges[0].NewStart == 0 || file.ChangedRanges[0].NewEnd == 0 {
				t.Fatalf("changed ranges = %+v, want non-empty new-line range", file.ChangedRanges)
			}
			for _, want := range []string{"diff --git a/a.go b/a.go", "@@", "+\tvalue := \"new\""} {
				if !strings.Contains(file.UnifiedDiff, want) {
					t.Fatalf("unified diff missing %q:\n%s", want, file.UnifiedDiff)
				}
			}
		})
	}
}

func agentGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
