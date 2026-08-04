package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alibaba/open-code-review/internal/session"
)

const runnerE2ESecret = "sk_live_ocr_runner_e2e_do_not_leak"

func TestRunnerE2E(t *testing.T) {
	if os.Getenv("OCR_RUNNER_E2E") != "1" {
		t.Skip("set OCR_RUNNER_E2E=1 to consume local runner subscription quota")
	}

	ocr := buildRunnerE2EBinary(t)
	for _, kind := range []string{"codex", "claude"} {
		t.Run(kind+"-review", func(t *testing.T) { runRealReview(t, ocr, kind) })
		t.Run(kind+"-scan", func(t *testing.T) { runRealScan(t, ocr, kind) })
	}
}

func buildRunnerE2EBinary(t *testing.T) string {
	t.Helper()
	moduleRoot := moduleRoot(t)
	ocr := filepath.Join(t.TempDir(), "ocr")
	cmd := exec.Command("go", "build", "-o", ocr, "./cmd/opencodereview")
	cmd.Dir = moduleRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build OCR binary: %v\n%s", err, out)
	}
	return ocr
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("go", "env", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("go env GOMOD did not return a module file")
	}
	return filepath.Dir(gomod)
}

func runRealReview(t *testing.T, ocr, kind string) {
	t.Helper()
	fixture := newRunnerE2EFixture(t)
	defer fixture.assertSourceUnchanged(t)
	out := runOCRJSON(t, ocr, "review", "--repo", fixture.repo, "--runner", kind, "--format", "json", "--audience", "agent")
	assertRunnerE2EOutput(t, out, kind, fixture.sourcePath)
}

func runRealScan(t *testing.T, ocr, kind string) {
	t.Helper()
	fixture := newRunnerE2EFixture(t)
	defer fixture.assertSourceUnchanged(t)
	out := runOCRJSON(t, ocr, "scan", "--repo", fixture.repo, "--path", fixture.sourcePath, "--runner", kind, "--format", "json", "--audience", "agent", "--no-plan", "--no-dedup", "--no-summary")
	assertRunnerE2EOutput(t, out, kind, fixture.sourcePath)
}

type runnerE2EFixture struct {
	repo       string
	sourcePath string
	snapshot   []byte
}

func newRunnerE2EFixture(t *testing.T) runnerE2EFixture {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "runner-e2e@example.invalid")
	runGit(t, repo, "config", "user.name", "Runner E2E")

	writeFile(t, filepath.Join(repo, "go.mod"), "module runnerfixture\n\ngo 1.24\n")
	writeFile(t, filepath.Join(repo, "main.go"), `package main

import "fmt"

func main() {
	fmt.Println("safe baseline")
}
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "baseline")

	badSource := fmt.Sprintf(`package main

import "os"

func main() {
	apiKey := %q
	os.WriteFile("token.txt", []byte(apiKey), 0o600)
}
`, runnerE2ESecret)
	writeFile(t, filepath.Join(repo, "main.go"), badSource)
	snapshot := readFile(t, filepath.Join(repo, "main.go"))
	return runnerE2EFixture{repo: repo, sourcePath: "main.go", snapshot: snapshot}
}

func (f runnerE2EFixture) assertSourceUnchanged(t *testing.T) {
	t.Helper()
	got := readFile(t, filepath.Join(f.repo, f.sourcePath))
	if !bytes.Equal(got, f.snapshot) {
		t.Fatalf("%s was mutated by OCR invocation", f.sourcePath)
	}
}

func runOCRJSON(t *testing.T, ocr string, args ...string) jsonOutput {
	t.Helper()
	cmd := exec.Command(ocr, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("ocr %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, redactRunnerE2ESecrets(stdout.String()), redactRunnerE2ESecrets(stderr.String()))
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, runnerE2ESecret) {
		t.Fatalf("ocr %s output leaked the fixture secret", strings.Join(args, " "))
	}
	var out jsonOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("ocr %s emitted invalid JSON: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, redactRunnerE2ESecrets(stdout.String()), redactRunnerE2ESecrets(stderr.String()))
	}
	return out
}

func assertRunnerE2EOutput(t *testing.T, out jsonOutput, kind, wantPath string) {
	t.Helper()
	if out.LLM == nil || out.LLM.Provider != "local-runner" || out.LLM.Runner != kind {
		t.Fatalf("llm identity = %+v, want local-runner/%s", out.LLM, kind)
	}
	if out.Status == "" {
		t.Fatal("status is empty")
	}
	if out.Summary == nil || out.Summary.FilesReviewed < 1 {
		t.Fatalf("summary = %+v, want at least one reviewed file", out.Summary)
	}
	if out.ToolCalls == nil || out.ToolCalls.ByTool == nil {
		t.Fatalf("tool_calls = %+v, want initialized tool-call metadata", out.ToolCalls)
	}
	if out.Manifest == nil {
		t.Fatal("manifest is nil; want coverage/session metadata")
	}
	if out.SessionID == "" {
		t.Fatal("session_id is empty")
	}
	if !manifestMentionsPath(out, wantPath) {
		t.Fatalf("manifest coverage does not mention %q: %+v", wantPath, out.Manifest.Coverage)
	}
	for i, comment := range out.Comments {
		if comment.Path != wantPath {
			t.Fatalf("comments[%d].Path = %q, want %q", i, comment.Path, wantPath)
		}
		if comment.StartLine < 1 || comment.EndLine < comment.StartLine {
			t.Fatalf("comments[%d] has invalid line range %d-%d", i, comment.StartLine, comment.EndLine)
		}
		if comment.Content == "" {
			t.Fatalf("comments[%d].Content is empty", i)
		}
		if !containsString([]string{"critical", "high", "medium", "low"}, comment.Severity) {
			t.Fatalf("comments[%d].Severity = %q", i, comment.Severity)
		}
		if !containsString([]string{"bug", "security", "performance", "maintainability", "test", "style", "documentation", "other"}, comment.Category) {
			t.Fatalf("comments[%d].Category = %q", i, comment.Category)
		}
	}
}

func manifestMentionsPath(out jsonOutput, wantPath string) bool {
	coverage := out.Manifest.Coverage
	groups := [][]string{
		coveragePaths(coverage.Selected),
		coveragePaths(coverage.Completed),
		coveragePaths(coverage.Reused),
		coveragePaths(coverage.Failed),
		coveragePaths(coverage.Waived),
	}
	for _, paths := range groups {
		for _, path := range paths {
			if path == wantPath {
				return true
			}
		}
	}
	return false
}

func coveragePaths(items []session.CoverageItem) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		paths = append(paths, item.Path)
	}
	return paths
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func redactRunnerE2ESecrets(s string) string {
	return strings.ReplaceAll(s, runnerE2ESecret, "[REDACTED_RUNNER_E2E_SECRET]")
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
