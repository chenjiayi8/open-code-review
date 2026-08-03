package runner

import (
	"bytes"
	"errors"
	"fmt"
)

// RenderPrompt renders the strict instructions for a local runner invocation.
func RenderPrompt(req Request) (string, error) {
	if req.Operation == "" {
		return "", errors.New("runner prompt: operation is required")
	}
	if req.Repository == "" {
		return "", errors.New("runner prompt: repository is required")
	}
	if len(req.Files) == 0 {
		return "", errors.New("runner prompt: at least one selected file is required")
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "You are the local open-code-review runner for operation %q.\n", req.Operation)
	fmt.Fprintf(&buf, "Repository root: %s\n\n", req.Repository)
	buf.WriteString("Review exactly the manifest files listed below. Do not review, mention, or infer findings for files outside this manifest. Use repository reads only; do not use network access or external services.\n")
	buf.WriteString("Return only a single JSON object that conforms to the schema below. Do not write prose, Markdown fences, or explanations outside that JSON object.\n")
	buf.WriteString("Report every file you completed in reviewed_files, even when findings is empty.\n\n")

	if req.Background != "" {
		buf.WriteString("Background:\n")
		buf.WriteString(req.Background)
		buf.WriteString("\n\n")
	}
	if len(req.Rules) > 0 {
		buf.WriteString("Global rules:\n")
		for _, rule := range req.Rules {
			if rule == "" {
				continue
			}
			fmt.Fprintf(&buf, "- %s\n", rule)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("Manifest files:\n")
	for i, file := range req.Files {
		if file.Path == "" {
			return "", fmt.Errorf("runner prompt: files[%d].path is required", i)
		}
		fmt.Fprintf(&buf, "- path: %s\n", file.Path)
		if file.Rule != "" {
			fmt.Fprintf(&buf, "  rule: %s\n", file.Rule)
		}
	}
	buf.WriteString("\nResult schema:\n")
	buf.Write(Schema())
	buf.WriteByte('\n')
	return buf.String(), nil
}
