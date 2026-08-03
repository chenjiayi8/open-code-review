package runner

import (
	"slices"
	"testing"
)

func TestCodexCommandIsReadOnlyAndUsesSchema(t *testing.T) {
	args := codexArgs("/repo", "/tmp/schema.json", "prompt", "")
	if !slices.Contains(args, "--sandbox") || !slices.Contains(args, "read-only") || !slices.Contains(args, "--output-schema") {
		t.Fatalf("unsafe codex args: %#v", args)
	}
}

func TestSafeChildEnvRemovesCredentialAndOCRLLMVariables(t *testing.T) {
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
	if !slices.Contains(env, "PATH=/bin") || !slices.Contains(env, "OCR_MAX_RETRIES=3") {
		t.Fatalf("safe env removed non-credential entries: %#v", env)
	}
	for _, blocked := range []string{"OPENAI_API_KEY=openai", "CODEX_API_KEY=codex", "CODEX_ACCESS_TOKEN=token", "ANTHROPIC_API_KEY=anthropic", "ANTHROPIC_BASE_URL=https://example.invalid", "OCR_LLM_URL=https://example.invalid", "OCR_LLM_AUTH_TOKEN=ocr-token", "OCR_PROVIDER_TOKEN=provider-token"} {
		if slices.Contains(env, blocked) {
			t.Fatalf("safe env kept credential entry %q in %#v", blocked, env)
		}
	}
}

func TestClaudeCommandRestrictsToolsAndUsesSchema(t *testing.T) {
	args := claudeArgs("/tmp/schema.json", "")
	for _, want := range []string{"-p", "--output-format", "json", "--json-schema", "/tmp/schema.json", "--permission-mode", "dontAsk", "--allowedTools", `Read,Glob,Grep,Bash(git\ *)`} {
		if !slices.Contains(args, want) {
			t.Fatalf("claude args missing %q: %#v", want, args)
		}
	}
}
