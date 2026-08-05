package runner

import (
	"context"
	"errors"
	"os"
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

func TestClaudeStatusAcceptsObservedAPIProviderField(t *testing.T) {
	identity, err := parseClaudeStatus([]byte(`{"loggedIn":true,"authMethod":"claude.ai","apiProvider":"anthropic"}`))
	if err != nil {
		t.Fatalf("parseClaudeStatus: %v", err)
	}
	if identity.Kind != Claude || identity.AuthMethod != "claude.ai" {
		t.Fatalf("identity = %+v, want claude.ai subscription", identity)
	}
}

func TestPreflightRejectsAbsentExecutableFromInjectedLookup(t *testing.T) {
	r := &Runner{
		kind: Claude,
		lookPath: func(string) (string, error) {
			return "", ErrExecutableNotFound
		},
		runCommand: func(context.Context, string, []string, []byte, []string, string) ([]byte, error) {
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

func TestRunClassifiesTimeoutFromInjectedCommand(t *testing.T) {
	r := &Runner{
		kind:     Claude,
		timeout:  time.Nanosecond,
		lookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
		runCommand: func(ctx context.Context, _ string, _ []string, _ []byte, _ []string, _ string) ([]byte, error) {
			<-ctx.Done()
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrRunnerTimeout
			}
			return nil, ctx.Err()
		},
		environ: func() []string { return nil },
	}
	_, err := r.Run(context.Background(), Request{
		Operation:  Review,
		Repository: t.TempDir(),
		Files:      []File{{Path: "src/a.go", Rule: "review"}},
	})
	if !errors.Is(err, ErrRunnerTimeout) {
		t.Fatalf("Run error = %v, want ErrRunnerTimeout", err)
	}
}

func TestRunDoesNotIssueAuthStatusCommand(t *testing.T) {
	repo := t.TempDir()
	calls := make([][]string, 0, 1)
	r := &Runner{
		kind:     Codex,
		lookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
		runCommand: func(_ context.Context, _ string, args []string, _ []byte, _ []string, _ string) ([]byte, error) {
			calls = append(calls, append([]string(nil), args...))
			for i := 0; i+1 < len(args); i++ {
				if args[i] == "-o" {
					if err := os.WriteFile(args[i+1], []byte(`{"reviewed_files":["src/a.go"],"findings":[]}`), 0o600); err != nil {
						t.Fatalf("write codex result: %v", err)
					}
				}
			}
			return nil, nil
		},
		environ: func() []string { return nil },
	}

	_, _ = r.Run(context.Background(), Request{
		Operation:  Review,
		Repository: repo,
		Files:      []File{{Path: "src/a.go", Rule: "review"}},
	})
	if len(calls) == 0 || calls[0][0] != "exec" {
		t.Fatalf("calls = %#v, want first command to start with codex exec", calls)
	}
}

func TestPreflightDiscoversExecutableWithoutStatusCommand(t *testing.T) {
	for _, kind := range []Kind{Codex, Claude} {
		t.Run(string(kind), func(t *testing.T) {
			r := &Runner{
				kind: kind,
				lookPath: func(name string) (string, error) {
					return "/usr/bin/" + name, nil
				},
				runCommand: func(context.Context, string, []string, []byte, []string, string) ([]byte, error) {
					t.Fatal("Preflight must not execute login, auth, or status commands")
					return nil, nil
				},
				environ: func() []string { return []string{"OPENAI_API_KEY=token", "ANTHROPIC_API_KEY=token"} },
			}

			identity, err := r.Preflight(context.Background())
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if identity.Kind != kind || identity.Executable != "/usr/bin/"+string(kind) || identity.AuthMethod != "native-cli" {
				t.Fatalf("identity = %+v, want native-cli executable discovery", identity)
			}
			if r.lastExecutable != identity.Executable {
				t.Fatalf("lastExecutable = %q, want %q", r.lastExecutable, identity.Executable)
			}
		})
	}
}

func TestClaudeRunDoesNotIssueAuthStatusCommand(t *testing.T) {
	repo := t.TempDir()
	calls := make([][]string, 0, 1)
	r := &Runner{
		kind:     Claude,
		lookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
		runCommand: func(_ context.Context, _ string, args []string, _ []byte, _ []string, _ string) ([]byte, error) {
			calls = append(calls, append([]string(nil), args...))
			return []byte(`{"type":"result","subtype":"success","is_error":false,"result":"{\"reviewed_files\":[\"src/a.go\"],\"findings\":[]}"}`), nil
		},
		environ: func() []string { return nil },
	}

	_, _ = r.Run(context.Background(), Request{
		Operation:  Review,
		Repository: repo,
		Files:      []File{{Path: "src/a.go", Rule: "review"}},
	})
	if len(calls) == 0 || calls[0][0] != "-p" {
		t.Fatalf("calls = %#v, want first command to start with claude -p", calls)
	}
}

func TestClaudeStatusRejectsMalformedAndUnauthenticatedJSON(t *testing.T) {
	cases := map[string][]byte{
		"malformed":       []byte(`{"loggedIn":true`),
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

func TestClaudeRunExecutesFromRequestRepository(t *testing.T) {
	repo := t.TempDir()
	processCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if processCWD == repo {
		t.Fatalf("test setup requires process cwd %q to differ from request repository", processCWD)
	}
	var commandCWDs []string
	r := &Runner{
		kind:     Claude,
		lookPath: func(string) (string, error) { return "/usr/bin/claude", nil },
		runCommand: func(_ context.Context, _ string, args []string, _ []byte, _ []string, cwd string) ([]byte, error) {
			commandCWDs = append(commandCWDs, cwd)
			if len(args) >= 3 && args[0] == "auth" {
				return []byte(`{"loggedIn":true,"authMethod":"claude.ai"}`), nil
			}
			return []byte(`{"type":"result","subtype":"success","is_error":false,"result":"{\"reviewed_files\":[\"src/a.go\"],\"findings\":[]}"}`), nil
		},
		environ: func() []string { return nil },
	}

	_, err = r.Run(context.Background(), Request{
		Operation:  Review,
		Repository: repo,
		Files:      []File{{Path: "src/a.go", Rule: "review"}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(commandCWDs) != 1 {
		t.Fatalf("command cwd calls = %#v, want only run command", commandCWDs)
	}
	if got := commandCWDs[0]; got != repo {
		t.Fatalf("claude run cwd = %q, want request repository %q", got, repo)
	}
}
