package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCodexStatusRequiresSavedChatGPTLogin(t *testing.T) {
	if _, err := parseCodexStatus([]byte("Logged in using ChatGPT\n")); err != nil {
		t.Fatalf("expected saved ChatGPT login acceptance: %v", err)
	}
	cases := map[string][]byte{
		"logged out with ChatGPT help": []byte("Not logged in. Run codex login to log in with ChatGPT.\n"),
		"login instruction":            []byte("Log in with ChatGPT to continue.\n"),
		"api key":                      []byte("Logged in using API key\n"),
		"empty":                        []byte("  "),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCodexStatus(raw); err == nil {
				t.Fatal("expected codex status rejection")
			}
		})
	}
}

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

func TestParseClaudeResultRequiresFinalSuccessEnvelope(t *testing.T) {
	validResult := `"{\"reviewed_files\":[\"src/a.go\"],\"findings\":[]}"`
	cases := map[string]string{
		"missing type":       `{"subtype":"success","is_error":false,"result":` + validResult + `}`,
		"wrong type":         `{"type":"assistant","subtype":"success","is_error":false,"result":` + validResult + `}`,
		"missing subtype":    `{"type":"result","is_error":false,"result":` + validResult + `}`,
		"unexpected subtype": `{"type":"result","subtype":"partial","is_error":false,"result":` + validResult + `}`,
		"error flag":         `{"type":"result","subtype":"success","is_error":true,"result":` + validResult + `}`,
		"error subtype":      `{"type":"result","subtype":"error","is_error":false,"result":` + validResult + `}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseClaudeResult([]byte(raw)); err == nil {
				t.Fatal("expected claude envelope rejection")
			}
		})
	}
}
