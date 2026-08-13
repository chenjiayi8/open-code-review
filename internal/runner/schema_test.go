package runner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alibaba/open-code-review/internal/model"
)

func TestParseResultRejectsUnknownReviewedFile(t *testing.T) {
	_, err := ParseResult([]byte(`{"reviewed_files":[],"findings":[],"extra":true}`))
	if err == nil {
		t.Fatal("expected additional property rejection")
	}
}

func TestParseResultAcceptsEmptyFindings(t *testing.T) {
	got, err := ParseResult([]byte(`{"reviewed_files":["src/a.go"],"findings":[]}`))
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	if len(got.ReviewedFiles) != 1 || got.ReviewedFiles[0] != "src/a.go" || len(got.Findings) != 0 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestParseResultAcceptsOptionalFindingFields(t *testing.T) {
	raw := `{
		"reviewed_files":["src/a.go"],
		"findings":[{
			"path":"src/a.go",
			"start_line":3,
			"end_line":5,
			"content":"fix this",
			"severity":"medium",
			"category":"maintainability",
			"suggestion_code":"new code",
			"existing_code":"old code",
			"thinking":"normalized reasoning"
		}],
		"summary":"reviewed one file"
	}`
	got, err := ParseResult([]byte(raw))
	if err != nil {
		t.Fatalf("ParseResult: %v", err)
	}
	finding := got.Findings[0]
	if finding.SuggestionCode != "new code" || finding.ExistingCode != "old code" || finding.Thinking != "normalized reasoning" || got.Summary != "reviewed one file" {
		t.Fatalf("optional fields not preserved: %+v", got)
	}
}

func TestParseResultRejectsInvalidLineNumbers(t *testing.T) {
	cases := map[string]string{
		"zero start":       `{"reviewed_files":["src/a.go"],"findings":[{"path":"src/a.go","start_line":0,"end_line":1,"content":"fix","severity":"low","category":"bug"}]}`,
		"end before start": `{"reviewed_files":["src/a.go"],"findings":[{"path":"src/a.go","start_line":4,"end_line":3,"content":"fix","severity":"low","category":"bug"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResult([]byte(raw)); err == nil {
				t.Fatal("expected line-number validation error")
			}
		})
	}
}

func TestParseResultRejectsInvalidSeverityAndCategory(t *testing.T) {
	cases := map[string]string{
		"severity": `{"reviewed_files":["src/a.go"],"findings":[{"path":"src/a.go","start_line":1,"end_line":1,"content":"fix","severity":"info","category":"bug"}]}`,
		"category": `{"reviewed_files":["src/a.go"],"findings":[{"path":"src/a.go","start_line":1,"end_line":1,"content":"fix","severity":"low","category":"nit"}]}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseResult([]byte(raw)); err == nil {
				t.Fatal("expected enum validation error")
			}
		})
	}
}

func TestParseResultRejectsDuplicateReviewedFilePaths(t *testing.T) {
	_, err := ParseResult([]byte(`{"reviewed_files":["src/a.go","src/a.go"],"findings":[]}`))
	if err == nil {
		t.Fatal("expected duplicate reviewed file rejection")
	}
}

func TestParseResultRejectsFindingOutsideReviewedFiles(t *testing.T) {
	_, err := ParseResult([]byte(`{"reviewed_files":[],"findings":[{"path":"src/a.go","start_line":1,"end_line":1,"content":"fix","severity":"low","category":"bug"}]}`))
	if err == nil {
		t.Fatal("expected finding path to be listed in reviewed_files")
	}
}

func TestParseResultRequiresReviewedFilesAndFindings(t *testing.T) {
	for _, raw := range []string{`{"findings":[]}`, `{"reviewed_files":[]}`} {
		if _, err := ParseResult([]byte(raw)); err == nil {
			t.Fatalf("expected required field rejection for %s", raw)
		}
	}
}

func TestSchemaIsClosedAndEnumerated(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal(Schema(), &schema); err != nil {
		t.Fatalf("Schema is invalid JSON: %v", err)
	}
	encoded := string(Schema())
	for _, want := range []string{`"additionalProperties":false`, model.SeverityCritical, model.SeverityHigh, model.SeverityMedium, model.SeverityLow, model.CategoryDocumentation, model.CategoryOther, "thinking"} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("schema missing %q: %s", want, encoded)
		}
	}
}

func TestSchemaIsCompatibleWithCodexStrictOutput(t *testing.T) {
	var schema struct {
		Required   []string                   `json:"required"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(Schema(), &schema); err != nil {
		t.Fatalf("Schema is invalid JSON: %v", err)
	}
	assertAllPropertiesRequired(t, "root", schema.Required, schema.Properties)

	var reviewedFiles struct {
		UniqueItems json.RawMessage `json:"uniqueItems"`
	}
	if err := json.Unmarshal(schema.Properties["reviewed_files"], &reviewedFiles); err != nil {
		t.Fatalf("decode reviewed_files schema: %v", err)
	}
	if reviewedFiles.UniqueItems != nil {
		t.Fatal("Codex output schemas do not support uniqueItems; ParseResult enforces duplicate-path rejection")
	}

	var findings struct {
		Items struct {
			Required   []string                   `json:"required"`
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"items"`
	}
	if err := json.Unmarshal(schema.Properties["findings"], &findings); err != nil {
		t.Fatalf("decode findings schema: %v", err)
	}
	assertAllPropertiesRequired(t, "findings item", findings.Items.Required, findings.Items.Properties)
}

func assertAllPropertiesRequired(t *testing.T, scope string, required []string, properties map[string]json.RawMessage) {
	t.Helper()
	requiredSet := make(map[string]struct{}, len(required))
	for _, name := range required {
		requiredSet[name] = struct{}{}
	}
	if len(requiredSet) != len(properties) {
		t.Fatalf("%s schema has %d required properties, want %d: %v", scope, len(requiredSet), len(properties), required)
	}
	for name := range properties {
		if _, ok := requiredSet[name]; !ok {
			t.Errorf("%s schema property %q must be required for Codex strict output", scope, name)
		}
	}
}
