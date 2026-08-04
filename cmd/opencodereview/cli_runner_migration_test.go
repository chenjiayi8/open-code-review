package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/open-code-review/internal/model"
)

func TestRemovedDirectLLMFlagsFail(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{"review provider", func() error { _, err := parseReviewFlags([]string{"--provider", "x"}); return err }},
		{"scan model", func() error { _, err := parseScanFlags([]string{"--model", "x"}); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), "unknown flag") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestRemovedDirectLLMCommandsFail(t *testing.T) {
	tests := [][]string{{"config", "provider"}, {"llm", "test"}}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			rootCmd.SetArgs(args)
			t.Cleanup(func() { rootCmd.SetArgs(nil) })
			err := rootCmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestRunnerValidationFailuresWriteNoArtifacts(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{"review", func() error { return runReview([]string{}) }},
		{"scan", func() error { return runScan([]string{}) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			err := tt.run()
			if err == nil || !strings.Contains(err.Error(), "--runner") {
				t.Fatalf("err = %v", err)
			}
			if _, statErr := os.Stat(filepath.Join(home, ".opencodereview")); !os.IsNotExist(statErr) {
				t.Fatalf("validation failure left artifacts under HOME (stat err = %v)", statErr)
			}
		})
	}
}

func TestJSONOutputIncludesLocalRunnerIdentityCommentsAndSession(t *testing.T) {
	ag := &mockResultProvider{filesReviewed: 1, sessionID: "session-123"}
	identity := &jsonLLMIdentity{Provider: "local-runner", Runner: "claude", Model: "sonnet"}
	comments := []model.LlmComment{{Path: "main.go", Content: "fix", StartLine: 1, EndLine: 1, Severity: "medium", Category: "bug"}}
	got := captureStdout(t, func() {
		if err := emitRunResult(context.Background(), ag, comments, time.Now(), "json", "developer", nil, identity); err != nil {
			t.Fatalf("emitRunResult: %v", err)
		}
	})
	var out jsonOutput
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.LLM == nil || out.LLM.Provider != "local-runner" || out.LLM.Runner != "claude" || out.LLM.Model != "sonnet" {
		t.Fatalf("llm = %+v", out.LLM)
	}
	if out.SessionID != "session-123" {
		t.Fatalf("session_id = %q", out.SessionID)
	}
	if len(out.Comments) != 1 || out.Comments[0].Path != "main.go" || out.Comments[0].StartLine != 1 || out.Comments[0].EndLine != 1 {
		t.Fatalf("comments = %+v", out.Comments)
	}
}
