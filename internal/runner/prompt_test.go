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

func TestRenderPromptIncludesReviewIdentityAndDiffContext(t *testing.T) {
	prompt, err := RenderPrompt(Request{
		Operation:  Review,
		Repository: "/repo",
		Review: ReviewContext{
			Mode:          "range",
			RequestedFrom: "main",
			RequestedHead: "feature",
			ResolvedBase:  "base-sha",
			ResolvedHead:  "head-sha",
			ExactRange:    "base-sha..head-sha",
		},
		Files: []File{{
			Path:          "src/new.go",
			OldPath:       "src/old.go",
			NewPath:       "src/new.go",
			Rule:          "check regressions",
			UnifiedDiff:   "diff --git a/src/old.go b/src/new.go\n@@ -10,2 +20,3 @@\n-old\n+new\n+more",
			ChangedRanges: []ChangedRange{{OldStart: 10, OldEnd: 11, NewStart: 20, NewEnd: 22}},
		}},
	})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, want := range []string{
		"Review input:",
		"mode: range",
		"requested_from: main",
		"requested_head: feature",
		"resolved_base: base-sha",
		"resolved_head: head-sha",
		"exact_range: base-sha..head-sha",
		"old_path: src/old.go",
		"new_path: src/new.go",
		"changed_ranges:",
		"old: 10-11, new: 20-22",
		"unified_diff:",
		"@@ -10,2 +20,3 @@",
		"+more",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRenderPromptDiffersByReviewMode(t *testing.T) {
	base := func(ctx ReviewContext) string {
		t.Helper()
		prompt, err := RenderPrompt(Request{
			Operation:  Review,
			Repository: "/repo",
			Review:     ctx,
			Files:      []File{{Path: "src/a.go", NewPath: "src/a.go", UnifiedDiff: "@@ -1 +1 @@\n-old\n+new"}},
		})
		if err != nil {
			t.Fatalf("RenderPrompt: %v", err)
		}
		return prompt
	}
	workspace := base(ReviewContext{Mode: "workspace", ResolvedBase: "workspace-base"})
	rangePrompt := base(ReviewContext{Mode: "range", RequestedFrom: "main", RequestedHead: "feature", ResolvedBase: "base", ResolvedHead: "head", ExactRange: "base..head"})
	commit := base(ReviewContext{Mode: "commit", RequestedHead: "abc123", ResolvedBase: "parent", ResolvedHead: "abc123", ExactRange: "parent..abc123"})
	if workspace == rangePrompt || workspace == commit || rangePrompt == commit {
		t.Fatalf("prompts should differ by review mode")
	}
	for prompt, want := range map[string]string{workspace: "mode: workspace", rangePrompt: "mode: range", commit: "mode: commit"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRenderPromptUsesFullManifestGuidanceForScanStyleRequests(t *testing.T) {
	prompt, err := RenderPrompt(Request{
		Operation:  Review,
		Repository: "/repo",
		Files:      []File{{Path: "src/a.go", Rule: "scan whole file"}},
	})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, forbidden := range []string{"changed hunks", "changed hunk context", "changed_ranges", "unified_diff"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("scan-style prompt mentioned %q:\n%s", forbidden, prompt)
		}
	}
	for _, want := range []string{"Review exactly the manifest files listed below.", "Do not review, mention, or infer findings for files outside this manifest."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("scan-style prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestRenderPromptUsesHunkOnlyGuidanceForDiffRequestsWithContext(t *testing.T) {
	prompt, err := RenderPrompt(Request{
		Operation:  Review,
		Repository: "/repo",
		Review:     ReviewContext{Mode: "workspace"},
		Files: []File{{
			Path:          "src/a.go",
			ChangedRanges: []ChangedRange{{OldStart: 1, OldEnd: 1, NewStart: 1, NewEnd: 2}},
			UnifiedDiff:   "@@ -1 +1,2 @@\n-old\n+new\n+more",
		}},
	})
	if err != nil {
		t.Fatalf("RenderPrompt: %v", err)
	}
	for _, want := range []string{"only the changed hunks described for each file", "outside the changed hunk context", "changed_ranges:", "unified_diff:"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("diff prompt missing %q:\n%s", want, prompt)
		}
	}
}
