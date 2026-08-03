package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClaudeStatusRequiresClaudeAISubscription(t *testing.T) {
	_, err := parseClaudeStatus([]byte(`{"loggedIn":true,"authMethod":"api_key"}`))
	if err == nil {
		t.Fatal("expected subscription-auth rejection")
	}
}

func TestPreflightRejectsAbsentExecutableFromInjectedLookup(t *testing.T) {
	r := &Runner{
		kind: Claude,
		lookPath: func(string) (string, error) {
			return "", ErrExecutableNotFound
		},
		runCommand: func(context.Context, string, []string, []byte, []string) ([]byte, error) {
			t.Fatal("runCommand must not be called after lookup failure")
			return nil, nil
		},
		environ: func() []string { return nil },
	}
	_, err := r.Preflight(context.Background())
	if !errors.Is(err, ErrExecutableNotFound) {
		t.Fatalf("Preflight error = %v, want ErrExecutableNotFound", err)
	}
}

func TestPreflightClassifiesTimeoutFromInjectedCommand(t *testing.T) {
	r := &Runner{
		kind:     Claude,
		timeout:  time.Nanosecond,
		lookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
		runCommand: func(ctx context.Context, _ string, _ []string, _ []byte, _ []string) ([]byte, error) {
			<-ctx.Done()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrRunnerTimeout
			}
			return nil, ctx.Err()
		},
		environ: func() []string { return nil },
	}
	_, err := r.Preflight(context.Background())
	if !errors.Is(err, ErrRunnerTimeout) {
		t.Fatalf("Preflight error = %v, want ErrRunnerTimeout", err)
	}
}

func TestClaudeStatusRejectsMalformedExtraAndUnauthenticatedJSON(t *testing.T) {
	cases := map[string][]byte{
		"malformed":       []byte(`{"loggedIn":true`),
		"extra field":     []byte(`{"loggedIn":true,"authMethod":"claude.ai","extra":true}`),
		"multiple values": []byte(`{"loggedIn":true,"authMethod":"claude.ai"} {}`),
		"logged out":      []byte(`{"loggedIn":false,"authMethod":"claude.ai"}`),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseClaudeStatus(raw); err == nil {
				t.Fatal("expected claude status rejection")
			}
		})
	}
}

func TestParseClaudeResultExtractsFinalResultEnvelope(t *testing.T) {
	raw := []byte(`{"type":"result","subtype":"success","is_error":false,"result":"{\"reviewed_files\":[\"src/a.go\"],\"findings\":[]}"}`)
	got, err := parseClaudeResult(raw)
	if err != nil {
		t.Fatalf("parseClaudeResult: %v", err)
	}
	if len(got.ReviewedFiles) != 1 || got.ReviewedFiles[0] != "src/a.go" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestParseClaudeResultRejectsMalformedInnerResult(t *testing.T) {
	if _, err := parseClaudeResult([]byte(`{"type":"result","subtype":"success","is_error":false,"result":"{\"reviewed_files\":[\"src/a.go\"],\"findings\":[],\"extra\":true}"}`)); err == nil {
		t.Fatal("expected malformed inner result rejection")
	}
}
