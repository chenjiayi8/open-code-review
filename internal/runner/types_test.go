package runner

import (
	"reflect"
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
	}
	if got := finding.AsComment(); got != want {
		t.Fatalf("AsComment() = %+v, want %+v", got, want)
	}
	if got := FindingFromComment(want); got != finding {
		t.Fatalf("FindingFromComment() = %+v, want %+v", got, finding)
	}
}

func TestFindingFromCommentIgnoresThinking(t *testing.T) {
	comment := model.LlmComment{Path: "src/a.go", Content: "fix", Thinking: "private reasoning"}
	finding := FindingFromComment(comment)
	if !reflect.DeepEqual(finding, Finding{Path: "src/a.go", Content: "fix"}) {
		t.Fatalf("FindingFromComment() = %+v", finding)
	}
}
