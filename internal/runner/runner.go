package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var (
	ErrExecutableNotFound = errors.New("runner executable not found")
	ErrUnauthenticated    = errors.New("runner subscription authentication required")
	ErrRunnerTimeout      = errors.New("runner command timed out")

	bearerTokenPattern = regexp.MustCompile(`(?i)(bearer\s+)[^\s]+`)
	apiKeyPattern      = regexp.MustCompile(`(?i)([A-Z0-9_]*API_KEY=)[^\s]+`)
)

type Identity struct {
	Kind       Kind
	Executable string
	AuthMethod string
}

type commandFunc func(context.Context, string, []string, []byte, []string, string) ([]byte, error)
type lookPathFunc func(string) (string, error)
type environFunc func() []string

type Executor interface {
	Run(context.Context, Request) (Result, error)
}

type Runner struct {
	executor       Executor
	kind           Kind
	model          string
	timeout        time.Duration
	lookPath       lookPathFunc
	runCommand     commandFunc
	environ        environFunc
	lastExecutable string
}

func NewWithExecutor(executor Executor) *Runner {
	return &Runner{executor: executor}
}

func New(kind Kind, model string, timeout time.Duration) (*Runner, error) {
	if kind != Codex && kind != Claude {
		return nil, fmt.Errorf("runner: unsupported kind %q", kind)
	}
	return &Runner{
		kind:       kind,
		model:      model,
		timeout:    timeout,
		lookPath:   exec.LookPath,
		runCommand: runExecCommand,
		environ:    os.Environ,
	}, nil
}

func (r *Runner) Preflight(ctx context.Context) (Identity, error) {
	if r == nil {
		return Identity{}, errors.New("runner: nil runner")
	}
	if err := r.initDefaults(); err != nil {
		return Identity{}, err
	}

	name := string(r.kind)
	executable, err := r.lookPath(name)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %s", ErrExecutableNotFound, name)
	}
	r.lastExecutable = executable
	return Identity{Kind: r.kind, Executable: executable, AuthMethod: "native-cli"}, nil
}

func (r *Runner) Run(ctx context.Context, request Request) (Result, error) {
	if r == nil {
		return Result{}, errors.New("runner: nil runner")
	}
	if r.executor != nil {
		return r.executor.Run(ctx, request)
	}
	identity, err := r.Preflight(ctx)
	if err != nil {
		return Result{}, err
	}

	prompt, err := RenderPrompt(request)
	if err != nil {
		return Result{}, err
	}
	tmp, err := os.MkdirTemp("", "ocr-runner-*")
	if err != nil {
		return Result{}, fmt.Errorf("runner: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	schemaPath := tmp + string(os.PathSeparator) + "schema.json"
	resultPath := tmp + string(os.PathSeparator) + "result.json"
	if err := os.WriteFile(schemaPath, Schema(), 0o600); err != nil {
		return Result{}, fmt.Errorf("runner: write schema: %w", err)
	}

	runCtx, cancel := r.withTimeout(ctx)
	defer cancel()
	env := safeChildEnv(r.environ())

	switch r.kind {
	case Codex:
		_, err := r.runCommand(runCtx, identity.Executable, codexArgs(request.Repository, schemaPath, resultPath, r.model), []byte(prompt), env, request.Repository)
		if err != nil {
			return Result{}, err
		}
		data, err := os.ReadFile(resultPath)
		if err != nil {
			return Result{}, fmt.Errorf("runner: read codex result: %w", err)
		}
		return ParseResult(data)
	case Claude:
		out, err := r.runCommand(runCtx, identity.Executable, claudeArgs(schemaPath, r.model), []byte(prompt), env, request.Repository)
		if err != nil {
			return Result{}, err
		}
		return parseClaudeResult(out)
	default:
		return Result{}, fmt.Errorf("runner: unsupported kind %q", r.kind)
	}
}

func (r *Runner) initDefaults() error {
	if r.kind != Codex && r.kind != Claude {
		return fmt.Errorf("runner: unsupported kind %q", r.kind)
	}
	if r.lookPath == nil {
		r.lookPath = exec.LookPath
	}
	if r.runCommand == nil {
		r.runCommand = runExecCommand
	}
	if r.environ == nil {
		r.environ = os.Environ
	}
	return nil
}

func (r *Runner) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, r.timeout)
}

func runExecCommand(ctx context.Context, executable string, args []string, stdin []byte, env []string, cwd string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = env
	if cwd != "" {
		cmd.Dir = cwd
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %s", ErrRunnerTimeout, executable)
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		message = redactRunnerFailure(message, env)
		return nil, fmt.Errorf("runner command %s failed: %s", executable, message)
	}
	return out, nil
}

func redactRunnerFailure(message string, env []string) string {
	redacted := message
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || value == "" || !sensitiveNativeEnv(name) {
			continue
		}
		redacted = strings.ReplaceAll(redacted, value, "[redacted]")
	}
	redacted = bearerTokenPattern.ReplaceAllString(redacted, "${1}[redacted]")
	redacted = apiKeyPattern.ReplaceAllString(redacted, "${1}[redacted]")
	return redacted
}

func sensitiveNativeEnv(name string) bool {
	switch name {
	case "OPENAI_API_KEY", "CODEX_API_KEY", "CODEX_ACCESS_TOKEN", "ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL":
		return true
	default:
		return false
	}
}

func parseCodexStatus(data []byte) (Identity, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return Identity{}, fmt.Errorf("%w: codex login status was empty", ErrUnauthenticated)
	}
	if strings.EqualFold(text, "Logged in using ChatGPT") {
		return Identity{Kind: Codex, AuthMethod: "chatgpt"}, nil
	}
	return Identity{}, fmt.Errorf("%w: codex must be logged in with ChatGPT", ErrUnauthenticated)
}

func parseClaudeStatus(data []byte) (Identity, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var status struct {
		LoggedIn   bool   `json:"loggedIn"`
		AuthMethod string `json:"authMethod"`
	}
	if err := dec.Decode(&status); err != nil {
		return Identity{}, fmt.Errorf("claude auth status: decode: %w", err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return Identity{}, errors.New("claude auth status: multiple JSON values")
	}
	if !status.LoggedIn || status.AuthMethod != "claude.ai" {
		return Identity{}, fmt.Errorf("%w: claude must be logged in with claude.ai", ErrUnauthenticated)
	}
	return Identity{Kind: Claude, AuthMethod: status.AuthMethod}, nil
}

func parseClaudeResult(data []byte) (Result, error) {
	var envelope struct {
		Type    string          `json:"type"`
		Subtype string          `json:"subtype"`
		IsError bool            `json:"is_error"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Result{}, fmt.Errorf("claude result: decode envelope: %w", err)
	}
	if envelope.Type != "result" {
		return Result{}, fmt.Errorf("claude result: unexpected type %q", envelope.Type)
	}
	if envelope.Subtype != "success" {
		return Result{}, fmt.Errorf("claude result: unexpected subtype %q", envelope.Subtype)
	}
	if envelope.IsError {
		return Result{}, errors.New("claude result: runner reported error")
	}
	if len(envelope.Result) == 0 {
		return Result{}, errors.New("claude result: missing result field")
	}
	if envelope.Result[0] == '"' {
		var text string
		if err := json.Unmarshal(envelope.Result, &text); err != nil {
			return Result{}, fmt.Errorf("claude result: decode result string: %w", err)
		}
		return ParseResult([]byte(text))
	}
	return ParseResult(envelope.Result)
}
