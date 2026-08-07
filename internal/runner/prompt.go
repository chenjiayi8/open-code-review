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
	if requestHasDiffContext(req) {
		buf.WriteString("Review exactly the manifest files listed below and only the changed hunks described for each file. Do not review, mention, or infer findings for files outside this manifest or outside the changed hunk context. Use repository reads only; do not use network access or external services.\n")
	} else {
		buf.WriteString("Review exactly the manifest files listed below. Do not review, mention, or infer findings for files outside this manifest. Use repository reads only; do not use network access or external services.\n")
	}
	buf.WriteString("Return only a single JSON object that conforms to the schema below. Do not write prose, Markdown fences, or explanations outside that JSON object.\n")
	buf.WriteString("Report every file you completed in reviewed_files, even when findings is empty.\n\n")

	if hasReviewContext(req.Review) {
		buf.WriteString("Review input:\n")
		writePromptField(&buf, "mode", req.Review.Mode)
		writePromptField(&buf, "requested_from", req.Review.RequestedFrom)
		writePromptField(&buf, "requested_head", req.Review.RequestedHead)
		writePromptField(&buf, "resolved_base", req.Review.ResolvedBase)
		writePromptField(&buf, "resolved_head", req.Review.ResolvedHead)
		writePromptField(&buf, "exact_range", req.Review.ExactRange)
		buf.WriteString("\n")
	}

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
		if file.OldPath != "" {
			fmt.Fprintf(&buf, "  old_path: %s\n", file.OldPath)
		}
		if file.NewPath != "" {
			fmt.Fprintf(&buf, "  new_path: %s\n", file.NewPath)
		}
		if file.Rule != "" {
			fmt.Fprintf(&buf, "  rule: %s\n", file.Rule)
		}
		if len(file.ChangedRanges) > 0 {
			buf.WriteString("  changed_ranges:\n")
			for _, r := range file.ChangedRanges {
				fmt.Fprintf(&buf, "    - old: %s, new: %s\n", formatRange(r.OldStart, r.OldEnd), formatRange(r.NewStart, r.NewEnd))
			}
		}
		if file.UnifiedDiff != "" {
			buf.WriteString("  unified_diff:\n")
			buf.WriteString(indentBlock(file.UnifiedDiff, "    "))
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\nResult schema:\n")
	buf.Write(Schema())
	buf.WriteByte('\n')
	return buf.String(), nil
}

func requestHasDiffContext(req Request) bool {
	if !hasReviewContext(req.Review) {
		return false
	}
	for _, file := range req.Files {
		if len(file.ChangedRanges) > 0 || file.UnifiedDiff != "" {
			return true
		}
	}
	return false
}

func hasReviewContext(ctx ReviewContext) bool {
	return ctx.Mode != "" || ctx.RequestedFrom != "" || ctx.RequestedHead != "" || ctx.ResolvedBase != "" || ctx.ResolvedHead != "" || ctx.ExactRange != ""
}

func writePromptField(buf *bytes.Buffer, name, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(buf, "%s: %s\n", name, value)
}

func formatRange(start, end int) string {
	if start <= 0 || end <= 0 {
		return "none"
	}
	if start == end {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d-%d", start, end)
}

func indentBlock(text, prefix string) string {
	if text == "" {
		return ""
	}
	lines := bytes.Split([]byte(text), []byte("\n"))
	var out bytes.Buffer
	for _, line := range lines {
		out.WriteString(prefix)
		out.Write(line)
		out.WriteByte('\n')
	}
	return out.String()
}
