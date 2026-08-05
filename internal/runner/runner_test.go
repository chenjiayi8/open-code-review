package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunExecCommandRedactsSecretsFromFailureStderr(t *testing.T) {
	env := []string{
		"OPENAI_API_KEY=openai-secret-sentinel",
		"CODEX_API_KEY=codex-secret-sentinel",
		"CODEX_ACCESS_TOKEN=access-secret-sentinel",
		"ANTHROPIC_API_KEY=anthropic-secret-sentinel",
		"ANTHROPIC_BASE_URL=https://anthropic-secret.example/v1",
		"CLAUDE_CODE_OAUTH_TOKEN=claude-oauth-secret-sentinel",
	}
	_, err := runExecCommand(context.Background(), "/bin/sh", []string{"-c", `if [ "$CLAUDE_CODE_OAUTH_TOKEN" != "claude-oauth-secret-sentinel" ]; then echo 'missing native token env' >&2; exit 8; fi; printf '%s\n' 'non-secret diagnostic before auth leak' 'openai-secret-sentinel' 'CODEX_API_KEY=codex-secret-sentinel' 'Authorization: Bearer bearer-secret-sentinel' 'EXTRA_API_KEY=extra-secret-sentinel' 'https://anthropic-secret.example/v1' 'claude-oauth-secret-sentinel' >&2; exit 7`}, nil, env, "")
	if err == nil {
		t.Fatal("expected command failure")
	}
	message := err.Error()
	for _, secret := range []string{
		"openai-secret-sentinel",
		"codex-secret-sentinel",
		"access-secret-sentinel",
		"anthropic-secret-sentinel",
		"https://anthropic-secret.example/v1",
		"claude-oauth-secret-sentinel",
		"bearer-secret-sentinel",
		"extra-secret-sentinel",
	} {
		if strings.Contains(message, secret) {
			t.Fatalf("error leaked secret %q: %s", secret, message)
		}
	}
	if !strings.Contains(message, "non-secret diagnostic before auth leak") {
		t.Fatalf("error lost non-secret diagnostic: %s", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("error did not include redaction marker: %s", message)
	}
}

func TestRunExecCommandClassifiesNativeAuthenticationFailure(t *testing.T) {
	env := []string{"CODEX_API_KEY=codex-secret-sentinel"}
	_, err := runExecCommand(context.Background(), "/bin/sh", []string{"-c", `printf '%s\n' 'Not logged in. Run the native CLI auth flow.' 'CODEX_API_KEY=codex-secret-sentinel' >&2; exit 1`}, nil, env, "")
	if err == nil {
		t.Fatal("expected command failure")
	}
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want ErrUnauthenticated", err)
	}
	message := err.Error()
	if !strings.Contains(message, "runner native command authentication failed") || !strings.Contains(message, "Not logged in") {
		t.Fatalf("error = %q, want native-auth diagnostic with sanitized runner output", message)
	}
	if strings.Contains(message, "codex-secret-sentinel") {
		t.Fatalf("error leaked auth env value: %s", message)
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

func TestRunPropagatesNativeAuthFailureWithoutStatusCommand(t *testing.T) {
	repo := t.TempDir()
	calls := make([][]string, 0, 1)
	r := &Runner{
		kind:     Codex,
		lookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
		runCommand: func(_ context.Context, _ string, args []string, _ []byte, _ []string, _ string) ([]byte, error) {
			calls = append(calls, append([]string(nil), args...))
			return nil, fmt.Errorf("%w: runner command /usr/bin/codex failed: Not logged in", ErrUnauthenticated)
		},
		environ: func() []string { return []string{"CODEX_API_KEY=codex-secret-sentinel"} },
	}

	_, err := r.Run(context.Background(), Request{
		Operation:  Review,
		Repository: repo,
		Files:      []File{{Path: "src/a.go", Rule: "review"}},
	})
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("Run error = %v, want ErrUnauthenticated", err)
	}
	if err == nil || !strings.Contains(err.Error(), "runner native command authentication failed") || !strings.Contains(err.Error(), "Not logged in") {
		t.Fatalf("Run error = %v, want native-auth diagnostic", err)
	}
	if len(calls) != 1 || calls[0][0] != "exec" {
		t.Fatalf("calls = %#v, want only codex exec command and no auth/status probe", calls)
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
		runCommand: func(_ context.Context, _ string, _ []string, _ []byte, _ []string, cwd string) ([]byte, error) {
			commandCWDs = append(commandCWDs, cwd)
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
