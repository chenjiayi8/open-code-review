package runner

import (
	"slices"
	"strings"
	"testing"
)

func TestCodexCommandIsReadOnlyAndUsesSchema(t *testing.T) {
	args := codexArgs("/repo", "/tmp/schema.json", "prompt", "")
	if !slices.Contains(args, "--sandbox") || !slices.Contains(args, "read-only") || !slices.Contains(args, "--output-schema") {
		t.Fatalf("unsafe codex args: %#v", args)
	}
}

func TestSafeChildEnvRemovesOnlyLegacyOCRLLMAndProviderVariables(t *testing.T) {
	env := safeChildEnv([]string{
		"PATH=/bin",
		"OPENAI_API_KEY=openai",
		"CODEX_API_KEY=codex",
		"CODEX_ACCESS_TOKEN=token",
		"ANTHROPIC_API_KEY=anthropic",
		"ANTHROPIC_BASE_URL=https://example.invalid",
		"OCR_LLM_URL=https://example.invalid",
		"OCR_LLM_AUTH_TOKEN=ocr-token",
		"OCR_PROVIDER_TOKEN=provider-token",
		"OCR_MAX_RETRIES=3",
	})
	for _, want := range []string{
		"PATH=/bin",
		"OCR_MAX_RETRIES=3",
		"OPENAI_API_KEY=openai",
		"CODEX_API_KEY=codex",
		"CODEX_ACCESS_TOKEN=token",
		"ANTHROPIC_API_KEY=anthropic",
		"ANTHROPIC_BASE_URL=https://example.invalid",
	} {
		if !slices.Contains(env, want) {
			t.Fatalf("safe env removed %q from %#v", want, env)
		}
	}
	for _, blocked := range []string{"OCR_LLM_URL=https://example.invalid", "OCR_LLM_AUTH_TOKEN=ocr-token", "OCR_PROVIDER_TOKEN=provider-token"} {
		if slices.Contains(env, blocked) {
			t.Fatalf("safe env kept legacy OCR entry %q in %#v", blocked, env)
		}
	}
}

func TestSafeChildEnvPreservesNativeCLIAuthentication(t *testing.T) {
	env := safeChildEnv([]string{
		"OPENAI_API_KEY=token",
		"CODEX_API_KEY=codex",
		"CODEX_ACCESS_TOKEN=access",
		"ANTHROPIC_API_KEY=anthropic",
		"ANTHROPIC_BASE_URL=https://anthropic.example",
		"OCR_LLM_TOKEN=legacy",
		"OCR_PROVIDER_TOKEN=legacy-provider",
	})
	for _, want := range []string{
		"OPENAI_API_KEY=token",
		"CODEX_API_KEY=codex",
		"CODEX_ACCESS_TOKEN=access",
		"ANTHROPIC_API_KEY=anthropic",
		"ANTHROPIC_BASE_URL=https://anthropic.example",
	} {
		if !slices.Contains(env, want) {
			t.Fatalf("safe env removed native auth entry %q from %#v", want, env)
		}
	}
	for _, blocked := range []string{"OCR_LLM_TOKEN=legacy", "OCR_PROVIDER_TOKEN=legacy-provider"} {
		if slices.Contains(env, blocked) {
			t.Fatalf("safe env kept legacy OCR entry %q in %#v", blocked, env)
		}
	}
}

func TestClaudeCommandRestrictsToolsAndUsesSchema(t *testing.T) {
	args := claudeArgs("/tmp/schema.json", "")
	for _, want := range []string{"-p", "--output-format", "json", "--json-schema", "/tmp/schema.json", "--permission-mode", "dontAsk", "--allowedTools", "Read,Glob,Grep"} {
		if !slices.Contains(args, want) {
			t.Fatalf("claude args missing %q: %#v", want, args)
		}
	}
}

func TestClaudeCommandDoesNotAllowMutatingGitSubcommands(t *testing.T) {
	args := claudeArgs("/tmp/schema.json", "")
	for i, arg := range args {
		if arg == "--allowedTools" && i+1 < len(args) {
			tools := args[i+1]
			if strings.Contains(tools, "Bash(git") {
				t.Fatalf("claude allowedTools contains git bash access: %q", tools)
			}
			for _, mutating := range []string{"commit", "checkout", "switch", "reset", "clean", "apply", "am", "merge", "rebase", "push", "pull", "fetch"} {
				if strings.Contains(tools, "git "+mutating) {
					t.Fatalf("claude allowedTools contains mutating git subcommand %q: %q", mutating, tools)
				}
			}
		}
	}
}
