package runner

import "github.com/alibaba/open-code-review/internal/model"

// Kind identifies a supported local runner backend.
type Kind string

const (
	Codex  Kind = "codex"
	Claude Kind = "claude"
)

// Operation identifies the review operation requested from a runner.
type Operation string

const (
	Review Operation = "review"
)

// File is one selected manifest file the runner is allowed to review.
type File struct {
	Path string
	Rule string
}

// Request is the complete prompt input for a local runner invocation.
type Request struct {
	Operation  Operation
	Repository string
	Files      []File
	Background string
	Rules      []string
}

// Result is the strict JSON contract returned by a local runner.
type Result struct {
	ReviewedFiles []string  `json:"reviewed_files"`
	Findings      []Finding `json:"findings"`
	Summary       string    `json:"summary,omitempty"`
}

// Finding is one local-runner code-review finding.
type Finding struct {
	Path           string `json:"path"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
	Content        string `json:"content"`
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	SuggestionCode string `json:"suggestion_code,omitempty"`
	ExistingCode   string `json:"existing_code,omitempty"`
	Thinking       string `json:"thinking,omitempty"`
}

// AsComment converts a runner Finding to the shared OCR review comment model.
func (f Finding) AsComment() model.LlmComment {
	return model.LlmComment{
		Path:           f.Path,
		Content:        f.Content,
		SuggestionCode: f.SuggestionCode,
		ExistingCode:   f.ExistingCode,
		StartLine:      f.StartLine,
		EndLine:        f.EndLine,
		Category:       f.Category,
		Severity:       f.Severity,
		Thinking:       f.Thinking,
	}
}

// FindingFromComment converts the shared OCR review comment model to a runner Finding.
func FindingFromComment(c model.LlmComment) Finding {
	return Finding{
		Path:           c.Path,
		Content:        c.Content,
		SuggestionCode: c.SuggestionCode,
		ExistingCode:   c.ExistingCode,
		StartLine:      c.StartLine,
		EndLine:        c.EndLine,
		Category:       c.Category,
		Severity:       c.Severity,
		Thinking:       c.Thinking,
	}
}
