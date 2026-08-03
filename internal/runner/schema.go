package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/alibaba/open-code-review/internal/model"
)

var resultSchema = []byte(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["reviewed_files", "findings"],
  "properties": {
    "reviewed_files": {
      "type": "array",
      "items": { "type": "string", "minLength": 1 },
      "uniqueItems": true
    },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["path", "start_line", "end_line", "content", "severity", "category"],
        "properties": {
          "path": { "type": "string", "minLength": 1 },
          "start_line": { "type": "integer", "minimum": 1 },
          "end_line": { "type": "integer", "minimum": 1 },
          "content": { "type": "string", "minLength": 1 },
          "severity": { "type": "string", "enum": ["critical", "high", "medium", "low"] },
          "category": { "type": "string", "enum": ["bug", "security", "performance", "maintainability", "test", "style", "documentation", "other"] },
          "suggestion_code": { "type": "string" },
          "existing_code": { "type": "string" }
        }
      }
    },
    "summary": { "type": "string" }
  }
}`)

// Schema returns the strict local-runner JSON result schema.
func Schema() []byte {
	var compact bytes.Buffer
	if err := json.Compact(&compact, resultSchema); err != nil {
		return bytes.Clone(resultSchema)
	}
	return compact.Bytes()
}

// ParseResult decodes and validates a local-runner JSON result.
func ParseResult(data []byte) (Result, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var result Result
	if err := dec.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("runner result: decode: %w", err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return Result{}, errors.New("runner result: multiple JSON values")
	}
	if result.ReviewedFiles == nil {
		return Result{}, errors.New("runner result: reviewed_files is required")
	}
	if result.Findings == nil {
		return Result{}, errors.New("runner result: findings is required")
	}

	reviewed := make(map[string]struct{}, len(result.ReviewedFiles))
	for i, path := range result.ReviewedFiles {
		if path == "" {
			return Result{}, fmt.Errorf("runner result: reviewed_files[%d] is empty", i)
		}
		if _, ok := reviewed[path]; ok {
			return Result{}, fmt.Errorf("runner result: duplicate reviewed file %q", path)
		}
		reviewed[path] = struct{}{}
	}

	for i, finding := range result.Findings {
		if err := validateFinding(finding, reviewed); err != nil {
			return Result{}, fmt.Errorf("runner result: findings[%d]: %w", i, err)
		}
	}
	return result, nil
}

func validateFinding(f Finding, reviewed map[string]struct{}) error {
	if f.Path == "" {
		return errors.New("path is required")
	}
	if _, ok := reviewed[f.Path]; !ok {
		return fmt.Errorf("path %q is not in reviewed_files", f.Path)
	}
	if f.StartLine < 1 {
		return errors.New("start_line must be >= 1")
	}
	if f.EndLine < f.StartLine {
		return errors.New("end_line must be >= start_line")
	}
	if f.Content == "" {
		return errors.New("content is required")
	}
	if !contains(model.ReviewSeverities(), f.Severity) {
		return fmt.Errorf("invalid severity %q", f.Severity)
	}
	if !contains(model.ReviewCategories(), f.Category) {
		return fmt.Errorf("invalid category %q", f.Category)
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
