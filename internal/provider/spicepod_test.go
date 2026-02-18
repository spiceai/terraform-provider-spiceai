// Copyright (c) Spice AI, Inc. 2025, 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNormalizeSpicepodToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "yaml simple",
			input: `version: v1
kind: Spicepod
name: test-app`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name: "yaml with datasets",
			input: `version: v1
kind: Spicepod
name: test-app
datasets:
  - name: test_dataset
    from: s3://bucket/path/
    params:
      file_format: parquet`,
			expected: `{"datasets":[{"from":"s3://bucket/path/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name:     "json already normalized",
			input:    `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name:     "json with different key order",
			input:    `{"version":"v1","name":"test-app","kind":"Spicepod"}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name:     "json with whitespace",
			input:    `{  "version": "v1",  "name": "test-app",  "kind": "Spicepod"  }`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name: "json pretty printed",
			input: `{
  "version": "v1",
  "kind": "Spicepod",
  "name": "test-app"
}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1"}`,
		},
		{
			name: "yaml with multiple datasets",
			input: `version: v1
kind: Spicepod
name: multi-dataset-app
datasets:
  - name: dataset1
    from: s3://bucket1/
  - name: dataset2
    from: s3://bucket2/`,
			expected: `{"datasets":[{"from":"s3://bucket1/","name":"dataset1"},{"from":"s3://bucket2/","name":"dataset2"}],"kind":"Spicepod","name":"multi-dataset-app","version":"v1"}`,
		},
		{
			name:     "empty string returns null",
			input:    "",
			expected: "null",
		},
		{
			name:     "invalid yaml/json gets quoted as yaml string",
			input:    "not valid yaml or json {{{",
			expected: `"not valid yaml or json {{{"`,
		},
		{
			name: "yaml with nested params",
			input: `version: v1
kind: Spicepod
name: nested-app
datasets:
  - name: test
    from: source
    params:
      nested:
        key1: value1
        key2: value2`,
			expected: `{"datasets":[{"from":"source","name":"test","params":{"nested":{"key1":"value1","key2":"value2"}}}],"kind":"Spicepod","name":"nested-app","version":"v1"}`,
		},
		{
			name: "yaml with boolean and numeric values",
			input: `version: v1
kind: Spicepod
name: typed-app
settings:
  enabled: true
  count: 42
  ratio: 3.14`,
			expected: `{"kind":"Spicepod","name":"typed-app","settings":{"count":42,"enabled":true,"ratio":3.14},"version":"v1"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSpicepodToJSON(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSpicepodToJSON() =\n%s\nwant:\n%s", result, tt.expected)
			}
		})
	}
}

func TestNormalizeSpicepodToJSON_YAMLAndJSONEquivalence(t *testing.T) {
	// Test that equivalent YAML and JSON produce the same normalized output
	yamlInput := `version: v1
kind: Spicepod
name: terraform-test-app
datasets:
  - name: test_dataset
    from: s3://spiceai-demo-datasets/taxi_trips/2024/
    params:
      file_format: parquet`

	jsonInput := `{"datasets":[{"from":"s3://spiceai-demo-datasets/taxi_trips/2024/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"terraform-test-app","version":"v1"}`

	yamlNormalized := normalizeSpicepodToJSON(yamlInput)
	jsonNormalized := normalizeSpicepodToJSON(jsonInput)

	if yamlNormalized != jsonNormalized {
		t.Errorf("YAML and JSON normalization mismatch:\nYAML normalized: %s\nJSON normalized: %s", yamlNormalized, jsonNormalized)
	}
}

func TestNormalizeSpicepodToJSON_Idempotent(t *testing.T) {
	// Test that normalizing an already normalized value produces the same result
	inputs := []string{
		`{"kind":"Spicepod","name":"test","version":"v1"}`,
		`version: v1
kind: Spicepod
name: test`,
	}

	for _, input := range inputs {
		first := normalizeSpicepodToJSON(input)
		second := normalizeSpicepodToJSON(first)
		third := normalizeSpicepodToJSON(second)

		if first != second || second != third {
			t.Errorf("normalizeSpicepodToJSON is not idempotent:\nfirst:  %s\nsecond: %s\nthird:  %s", first, second, third)
		}
	}
}

func TestSpicepodStringType_ValueFromString(t *testing.T) {
	ctx := context.Background()
	spicepodType := SpicepodStringType{}

	tests := []struct {
		name  string
		input types.String
	}{
		{
			name:  "normal string",
			input: types.StringValue("version: v1"),
		},
		{
			name:  "null string",
			input: types.StringNull(),
		},
		{
			name:  "unknown string",
			input: types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, diags := spicepodType.ValueFromString(ctx, tt.input)
			if diags.HasError() {
				t.Errorf("unexpected error: %v", diags)
			}

			spicepodValue, ok := result.(SpicepodStringValue)
			if !ok {
				t.Errorf("expected SpicepodStringValue, got %T", result)
			}

			if spicepodValue.ValueString() != tt.input.ValueString() {
				t.Errorf("value mismatch: got %s, want %s", spicepodValue.ValueString(), tt.input.ValueString())
			}
		})
	}
}

func TestSpicepodStringValue_StringSemanticEquals(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		value1   SpicepodStringValue
		value2   SpicepodStringValue
		expected bool
	}{
		{
			name:     "yaml equals equivalent json",
			value1:   SpicepodStringValue{StringValue: types.StringValue("version: v1\nkind: Spicepod\nname: test")},
			value2:   SpicepodStringValue{StringValue: types.StringValue(`{"kind":"Spicepod","name":"test","version":"v1"}`)},
			expected: true,
		},
		{
			name:     "json equals json with different key order",
			value1:   SpicepodStringValue{StringValue: types.StringValue(`{"version":"v1","name":"test","kind":"Spicepod"}`)},
			value2:   SpicepodStringValue{StringValue: types.StringValue(`{"kind":"Spicepod","name":"test","version":"v1"}`)},
			expected: true,
		},
		{
			name:     "different values are not equal",
			value1:   SpicepodStringValue{StringValue: types.StringValue("version: v1\nkind: Spicepod\nname: test1")},
			value2:   SpicepodStringValue{StringValue: types.StringValue("version: v1\nkind: Spicepod\nname: test2")},
			expected: false,
		},
		{
			name:     "null values are equal",
			value1:   SpicepodStringValue{StringValue: types.StringNull()},
			value2:   SpicepodStringValue{StringValue: types.StringNull()},
			expected: true,
		},
		{
			name:     "null and non-null are not equal",
			value1:   SpicepodStringValue{StringValue: types.StringNull()},
			value2:   SpicepodStringValue{StringValue: types.StringValue("version: v1")},
			expected: false,
		},
		{
			name:     "complex yaml equals complex json",
			value1:   SpicepodStringValue{StringValue: types.StringValue("version: v1\nkind: Spicepod\nname: test\ndatasets:\n  - name: ds1\n    from: s3://bucket/")},
			value2:   SpicepodStringValue{StringValue: types.StringValue(`{"datasets":[{"from":"s3://bucket/","name":"ds1"}],"kind":"Spicepod","name":"test","version":"v1"}`)},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, diags := tt.value1.StringSemanticEquals(ctx, tt.value2)
			if diags.HasError() {
				t.Errorf("unexpected error: %v", diags)
			}

			if result != tt.expected {
				t.Errorf("StringSemanticEquals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSpicepodStringValue_SemanticEquals_RealWorldExample(t *testing.T) {
	ctx := context.Background()

	// User provides YAML (like from file("spicepod.yaml"))
	userYAML := SpicepodStringValue{StringValue: types.StringValue(`version: v1
kind: Spicepod
name: terraform-test-app

datasets:
  - name: test_dataset
    from: s3://spiceai-demo-datasets/taxi_trips/2024/
    params:
      file_format: parquet
`)}

	// API returns JSON
	apiJSON := SpicepodStringValue{StringValue: types.StringValue(`{"datasets":[{"from":"s3://spiceai-demo-datasets/taxi_trips/2024/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"terraform-test-app","version":"v1"}`)}

	// They should be semantically equal
	equal, diags := userYAML.StringSemanticEquals(ctx, apiJSON)
	if diags.HasError() {
		t.Errorf("unexpected error: %v", diags)
	}

	if !equal {
		t.Errorf("User YAML and API JSON should be semantically equal")
	}

	// Reverse comparison should also work
	equal, diags = apiJSON.StringSemanticEquals(ctx, userYAML)
	if diags.HasError() {
		t.Errorf("unexpected error: %v", diags)
	}

	if !equal {
		t.Errorf("API JSON and User YAML should be semantically equal (reverse)")
	}
}

func TestSpicepodStringValue_SemanticEquals_DetectsRealChanges(t *testing.T) {
	ctx := context.Background()

	// Old config
	oldYAML := SpicepodStringValue{StringValue: types.StringValue(`version: v1
kind: Spicepod
name: terraform-test-app-old
`)}

	// New config with different name
	newYAML := SpicepodStringValue{StringValue: types.StringValue(`version: v1
kind: Spicepod
name: terraform-test-app-new
`)}

	// They should NOT be equal (name changed)
	equal, diags := oldYAML.StringSemanticEquals(ctx, newYAML)
	if diags.HasError() {
		t.Errorf("unexpected error: %v", diags)
	}

	if equal {
		t.Errorf("Different spicepod names should NOT be semantically equal")
	}
}
