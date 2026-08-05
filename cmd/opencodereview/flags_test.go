package main

import (
	"strings"
	"testing"
)

func TestParseReviewFlagsBackgroundFile(t *testing.T) {
	for _, flag := range []string{"--background-file", "-B"} {
		t.Run(flag, func(t *testing.T) {
			opts, err := parseReviewFlags([]string{flag, "./docs/req.md", "--preview"})
			if err != nil {
				t.Fatalf("parseReviewFlags: %v", err)
			}
			if opts.backgroundFile != "./docs/req.md" {
				t.Errorf("backgroundFile = %q, want %q", opts.backgroundFile, "./docs/req.md")
			}
		})
	}
}

func TestParseReviewFlagsRunnerAndModel(t *testing.T) {
	opts, err := parseReviewFlags([]string{"--runner", "codex", "--runner-model", "gpt-5-codex"})
	if err != nil {
		t.Fatalf("parseReviewFlags: %v", err)
	}
	if opts.runner != "codex" || opts.runnerModel != "gpt-5-codex" {
		t.Fatalf("runner=%q runnerModel=%q", opts.runner, opts.runnerModel)
	}
	if opts.outputFormat != "text" {
		t.Errorf("outputFormat = %q, want %q", opts.outputFormat, "text")
	}
	if opts.audience != "human" {
		t.Errorf("audience = %q, want %q", opts.audience, "human")
	}
}

func TestParseReviewFlagsRejectsInvalidRunner(t *testing.T) {
	_, err := parseReviewFlags([]string{"--runner", "openai"})
	if err == nil {
		t.Fatal("expected invalid runner to fail")
	}
}

func TestParseReviewFlagsResume(t *testing.T) {
	opts, err := parseReviewFlags([]string{"--from", "main", "--to", "feature", "--runner", "codex", "--resume", "session-123"})
	if err != nil {
		t.Fatalf("parseReviewFlags: %v", err)
	}
	if opts.resume != "session-123" {
		t.Errorf("resume = %q, want session-123", opts.resume)
	}
}

func TestParseReviewFlags_PreviewWithResume(t *testing.T) {
	_, err := parseReviewFlags([]string{"--commit", "abc123", "--preview", "--resume", "session-123"})
	if err == nil {
		t.Fatal("expected error for --preview with --resume")
	}
}

func TestParseReviewFlagsPreviewDoesNotRequireRunner(t *testing.T) {
	opts, err := parseReviewFlags([]string{"--preview"})
	if err != nil {
		t.Fatalf("preview should not require --runner: %v", err)
	}
	if !opts.preview {
		t.Fatal("expected preview=true")
	}
}

func TestParseReviewFlags_InvalidAudience(t *testing.T) {
	_, err := parseReviewFlags([]string{"--runner", "codex", "--audience", "robot"})
	if err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestParseReviewFlags_NegativeMaxGitProcs(t *testing.T) {
	_, err := parseReviewFlags([]string{"--runner", "codex", "--max-git-procs", "-1"})
	if err == nil {
		t.Fatal("expected error for negative max-git-procs")
	}
}

func TestParseReviewFlags_ConflictingModes(t *testing.T) {
	_, err := parseReviewFlags([]string{"--from", "main", "--to", "dev", "--commit", "abc", "--runner", "codex"})
	if err == nil {
		t.Fatal("expected error for conflicting modes")
	}
}

func TestParseReviewFlags_FromWithoutTo(t *testing.T) {
	_, err := parseReviewFlags([]string{"--from", "main", "--runner", "codex"})
	if err == nil {
		t.Fatal("expected error for --from without --to")
	}
}

func TestParseReviewFlags_ToWithoutFrom(t *testing.T) {
	_, err := parseReviewFlags([]string{"--to", "dev", "--runner", "codex"})
	if err == nil {
		t.Fatal("expected error for --to without --from")
	}
}

func TestParseReviewFlags_ShortFlags(t *testing.T) {
	opts, err := parseReviewFlags([]string{"-c", "abc123", "-f", "json", "-p"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.commit != "abc123" {
		t.Errorf("commit = %q, want abc123", opts.commit)
	}
	if opts.outputFormat != "json" {
		t.Errorf("outputFormat = %q, want json", opts.outputFormat)
	}
	if !opts.preview {
		t.Error("expected preview=true")
	}
}

func TestParseReviewFlagsRejectsNegativeRunnerTimeout(t *testing.T) {
	_, err := parseReviewFlags([]string{"--runner", "codex", "--timeout", "-1"})
	if err == nil || !strings.Contains(err.Error(), "--timeout") {
		t.Fatalf("err = %v", err)
	}
}

func TestReviewExamplesRequireRunnerForRealExecution(t *testing.T) {
	for _, forbidden := range []string{"ocr review\n", "ocr review --from", "ocr review --commit", "ocr review --format", "ocr review --audience", "ocr review --exclude", "ocr review --background"} {
		if strings.Contains(reviewCmd.Example, forbidden) {
			t.Fatalf("review example contains runner-less real invocation %q:\n%s", forbidden, reviewCmd.Example)
		}
	}
}

func TestCLIHelpUsesNativeLocalRunnerWording(t *testing.T) {
	combined := strings.Join([]string{
		rootCmd.Long,
		reviewCmd.Long,
		reviewCmd.Example,
		scanCmd.Long,
		scanCmd.Example,
		reviewCmd.Flags().Lookup("runner").Usage,
		scanCmd.Flags().Lookup("runner").Usage,
	}, "\n")
	for _, forbidden := range []string{"subscription", "ChatGPT", "claude.ai", "claude auth login --claudeai"} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("CLI help contains %q:\n%s", forbidden, combined)
		}
	}
	for _, want := range []string{"authenticated local CLI", "native", "codex or claude"} {
		if !strings.Contains(combined, want) {
			t.Fatalf("CLI help missing %q:\n%s", want, combined)
		}
	}
}
