package runner

import "strings"

var blockedEnvNames = map[string]struct{}{
	"OPENAI_API_KEY":        {},
	"CODEX_API_KEY":         {},
	"CODEX_ACCESS_TOKEN":    {},
	"ANTHROPIC_API_KEY":     {},
	"ANTHROPIC_BASE_URL":    {},
	"OCR_LLM_URL":           {},
	"OCR_LLM_TOKEN":         {},
	"OCR_LLM_AUTH_TOKEN":    {},
	"OCR_LLM_MODEL":         {},
	"OCR_LLM_USE_ANTHROPIC": {},
	"OCR_USE_ANTHROPIC":     {},
}

func codexArgs(repo, schemaPath, resultPath, model string) []string {
	args := []string{"exec"}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args,
		"-C", repo,
		"--sandbox", "read-only",
		"--output-schema", schemaPath,
		"-o", resultPath,
		"-",
	)
}

func claudeArgs(schemaPath, model string) []string {
	args := []string{"-p"}
	if model != "" {
		args = append(args, "--model", model)
	}
	return append(args,
		"--output-format", "json",
		"--json-schema", schemaPath,
		"--permission-mode", "dontAsk",
		"--allowedTools", "Read,Glob,Grep",
	)
}

func safeChildEnv(environ []string) []string {
	filtered := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, _, ok := strings.Cut(entry, "=")
		if !ok || blockedEnv(name) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func blockedEnv(name string) bool {
	if _, ok := blockedEnvNames[name]; ok {
		return true
	}
	return strings.HasPrefix(name, "OCR_LLM_") || strings.HasPrefix(name, "OCR_PROVIDER_")
}
