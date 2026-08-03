package runner

import (
	"strings"
	"testing"
)

func TestRenderPromptListsOnlySelectedFiles(t *testing.T) {
	prompt, err := RenderPrompt(Request{Operation: Review, Repository: "/repo", Files: []File{{Path: "src/a.go", Rule: "check errors"}}})
	if err != nil || !strings.Contains(prompt, "src/a.go") || strings.Contains(prompt, "unselected.go") {
		t.Fatalf("unexpected prompt: %q, %v", prompt, err)
	}
}

func TestRenderPromptIncludesBackgroundAndRulesWithoutUnselectedPaths(t *testing.T) {
	prompt, err := RenderPrompt(Request{
		Operation:  Review,
		Repository: "/repo",
		Background: "Focus on concurrency regressions.",
		Files: []File{
			{Path: "src/a.go", Rule: "check errors"},
			{Path: "src/b.go", Rule: "check races"},
		},
		Rules: []string{"Favor high-confidence findings."},
	})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, want := range []string{"Focus on concurrency regressions.", "Favor high-confidence findings.", "src/a.go", "check errors", "src/b.go", "check races", "reviewed_files"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
	if strings.Contains(prompt, "unselected.go") || strings.Contains(prompt, "Review every file in the repository") {
		t.Fatalf("prompt includes unselected scope: %s", prompt)
	}
}

func TestRenderPromptRejectsEmptyManifest(t *testing.T) {
	if _, err := RenderPrompt(Request{Operation: Review, Repository: "/repo"}); err == nil {
		t.Fatal("expected empty manifest rejection")
	}
}
