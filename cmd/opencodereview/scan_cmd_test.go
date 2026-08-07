package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitPaths(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "internal/agent", []string{"internal/agent"}},
		{"multiple", "a.go,b.go,c.go", []string{"a.go", "b.go", "c.go"}},
		{"trims whitespace", "  a.go ,  b.go  ", []string{"a.go", "b.go"}},
		{"drops empty segments", "a.go,,b.go,", []string{"a.go", "b.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitPaths(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitPaths(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseScanFlagsRequiresRunner(t *testing.T) {
	_, err := parseScanFlags([]string{})
	if err == nil || !strings.Contains(err.Error(), "--runner") {
		t.Fatalf("%v", err)
	}
}

func TestParseScanFlagsAcceptsClaudeRunnerModel(t *testing.T) {
	got, err := parseScanFlags([]string{"--runner", "claude", "--runner-model", "sonnet"})
	if err != nil || got.runner != "claude" || got.runnerModel != "sonnet" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestParseScanFlagsPreviewDoesNotRequireRunner(t *testing.T) {
	opts, err := parseScanFlags([]string{"--preview"})
	if err != nil {
		t.Fatalf("preview should not require --runner: %v", err)
	}
	if !opts.preview {
		t.Fatal("expected preview=true")
	}
}

func TestParseScanFlags_RejectsInvalidAudience(t *testing.T) {
	_, err := parseScanFlags([]string{"--runner", "codex", "--audience", "robot"})
	if err == nil {
		t.Fatal("expected error for invalid --audience")
	}
	if !strings.Contains(err.Error(), "invalid --audience") {
		t.Errorf("error message = %q; want invalid --audience", err.Error())
	}
}

func TestParseScanFlags_RejectsNegativeMaxGitProcs(t *testing.T) {
	_, err := parseScanFlags([]string{"--runner", "codex", "--max-git-procs", "-3"})
	if err == nil {
		t.Fatal("expected error for negative --max-git-procs")
	}
	if !strings.Contains(err.Error(), "--max-git-procs") {
		t.Errorf("error message = %q; want it to mention --max-git-procs", err.Error())
	}
}

func TestParseScanFlags_PathNarrowsScope(t *testing.T) {
	opts, err := parseScanFlags([]string{"--runner", "codex", "--path", "internal/agent,internal/diff"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := splitPaths(opts.paths); !reflect.DeepEqual(got, []string{"internal/agent", "internal/diff"}) {
		t.Errorf("splitPaths(opts.paths) = %v", got)
	}
}

func TestParseScanFlags_HelpFlag(t *testing.T) {
	_, err := parseScanFlags([]string{"-h"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseScanFlags_BooleanFlags(t *testing.T) {
	opts, err := parseScanFlags([]string{"--runner", "codex", "--no-plan", "--no-dedup", "--no-summary"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.noPlan {
		t.Error("noPlan should be true")
	}
	if !opts.noDedup {
		t.Error("noDedup should be true")
	}
	if !opts.noSummary {
		t.Error("noSummary should be true")
	}
}

func TestParseScanFlagsRejectsNegativeRunnerTimeout(t *testing.T) {
	_, err := parseScanFlags([]string{"--runner", "claude", "--timeout", "-1"})
	if err == nil || !strings.Contains(err.Error(), "--timeout") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseScanFlagsRejectsInvalidBatch(t *testing.T) {
	_, err := parseScanFlags([]string{"--runner", "codex", "--batch", "sideways"})
	if err == nil {
		t.Fatal("expected error for invalid --batch")
	}
	if !strings.Contains(err.Error(), "invalid --batch") {
		t.Fatalf("error = %q, want invalid --batch", err.Error())
	}
}

func TestParseScanFlagsAcceptsValidBatchValues(t *testing.T) {
	for _, value := range []string{"none", "by-language", "by-directory"} {
		t.Run(value, func(t *testing.T) {
			opts, err := parseScanFlags([]string{"--runner", "codex", "--batch", value})
			if err != nil {
				t.Fatalf("parseScanFlags: %v", err)
			}
			if opts.batch != value {
				t.Fatalf("batch = %q, want %q", opts.batch, value)
			}
		})
	}
}

func TestScanExamplesRequireRunnerForRealExecutionAndAvoidLLMPreviewWording(t *testing.T) {
	for _, forbidden := range []string{"ocr scan\n", "ocr scan --path", "ocr scan --exclude", "ocr scan --no-plan", "ocr scan --resume"} {
		if strings.Contains(scanCmd.Example, forbidden) {
			t.Fatalf("scan example contains runner-less real invocation %q:\n%s", forbidden, scanCmd.Example)
		}
	}
	if strings.Contains(strings.ToLower(scanCmd.Example), "llm") {
		t.Fatalf("scan example still mentions LLM:\n%s", scanCmd.Example)
	}
}
