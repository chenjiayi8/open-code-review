package runner

import (
	"testing"

	"github.com/alibaba/open-code-review/internal/model"
)

func TestFindingMapsToLlmComment(t *testing.T) {
	finding := Finding{
		Path:           "src/a.go",
		StartLine:      10,
		EndLine:        12,
		Content:        "fix this",
		Severity:       model.SeverityHigh,
		Category:       model.CategoryBug,
		SuggestionCode: "new code",
		ExistingCode:   "old code",
		Thinking:       "normalized reasoning",
	}

	want := model.LlmComment{
		Path:           "src/a.go",
		StartLine:      10,
		EndLine:        12,
		Content:        "fix this",
		Severity:       model.SeverityHigh,
		Category:       model.CategoryBug,
		SuggestionCode: "new code",
		ExistingCode:   "old code",
		Thinking:       "normalized reasoning",
	}
	if got := finding.AsComment(); got != want {
		t.Fatalf("AsComment() = %+v, want %+v", got, want)
	}
	if got := FindingFromComment(want); got != finding {
		t.Fatalf("FindingFromComment() = %+v, want %+v", got, finding)
	}
}

func TestFindingPreservesThinkingInBothConversions(t *testing.T) {
	comment := model.LlmComment{Path: "src/a.go", Content: "fix", Thinking: "normalized reasoning"}
	finding := FindingFromComment(comment)
	if finding.Thinking != comment.Thinking {
		t.Fatalf("FindingFromComment() thinking = %q, want %q", finding.Thinking, comment.Thinking)
	}
	if got := finding.AsComment(); got.Thinking != comment.Thinking {
		t.Fatalf("AsComment() thinking = %q, want %q", got.Thinking, comment.Thinking)
	}
}
