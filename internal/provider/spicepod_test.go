// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
			input: `version: v1beta1
kind: Spicepod
name: test-app`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name: "yaml with datasets",
			input: `version: v1beta1
kind: Spicepod
name: test-app
datasets:
  - name: test_dataset
    from: s3://bucket/path/
    params:
      file_format: parquet`,
			expected: `{"datasets":[{"from":"s3://bucket/path/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name:     "json already normalized",
			input:    `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name:     "json with different key order",
			input:    `{"version":"v1beta1","name":"test-app","kind":"Spicepod"}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name:     "json with whitespace",
			input:    `{  "version": "v1beta1",  "name": "test-app",  "kind": "Spicepod"  }`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name: "json pretty printed",
			input: `{
  "version": "v1beta1",
  "kind": "Spicepod",
  "name": "test-app"
}`,
			expected: `{"kind":"Spicepod","name":"test-app","version":"v1beta1"}`,
		},
		{
			name: "yaml with multiple datasets",
			input: `version: v1beta1
kind: Spicepod
name: multi-dataset-app
datasets:
  - name: dataset1
    from: s3://bucket1/
  - name: dataset2
    from: s3://bucket2/`,
			expected: `{"datasets":[{"from":"s3://bucket1/","name":"dataset1"},{"from":"s3://bucket2/","name":"dataset2"}],"kind":"Spicepod","name":"multi-dataset-app","version":"v1beta1"}`,
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
			input: `version: v1beta1
kind: Spicepod
name: nested-app
datasets:
  - name: test
    from: source
    params:
      nested:
        key1: value1
        key2: value2`,
			expected: `{"datasets":[{"from":"source","name":"test","params":{"nested":{"key1":"value1","key2":"value2"}}}],"kind":"Spicepod","name":"nested-app","version":"v1beta1"}`,
		},
		{
			name: "yaml with boolean and numeric values",
			input: `version: v1beta1
kind: Spicepod
name: typed-app
settings:
  enabled: true
  count: 42
  ratio: 3.14`,
			expected: `{"kind":"Spicepod","name":"typed-app","settings":{"count":42,"enabled":true,"ratio":3.14},"version":"v1beta1"}`,
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
	yamlInput := `version: v1beta1
kind: Spicepod
name: terraform-test-app
datasets:
  - name: test_dataset
    from: s3://spiceai-demo-datasets/taxi_trips/2024/
    params:
      file_format: parquet`

	jsonInput := `{"datasets":[{"from":"s3://spiceai-demo-datasets/taxi_trips/2024/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"terraform-test-app","version":"v1beta1"}`

	yamlNormalized := normalizeSpicepodToJSON(yamlInput)
	jsonNormalized := normalizeSpicepodToJSON(jsonInput)

	if yamlNormalized != jsonNormalized {
		t.Errorf("YAML and JSON normalization mismatch:\nYAML normalized: %s\nJSON normalized: %s", yamlNormalized, jsonNormalized)
	}
}

func TestNormalizeSpicepodToJSON_Idempotent(t *testing.T) {
	// Test that normalizing an already normalized value produces the same result
	inputs := []string{
		`{"kind":"Spicepod","name":"test","version":"v1beta1"}`,
		`version: v1beta1
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

func TestSpicepodNormalizePlanModifier_Description(t *testing.T) {
	modifier := spicepodNormalizePlanModifier{}
	ctx := context.Background()

	desc := modifier.Description(ctx)
	if desc == "" {
		t.Error("Description() should not return empty string")
	}

	mdDesc := modifier.MarkdownDescription(ctx)
	if mdDesc == "" {
		t.Error("MarkdownDescription() should not return empty string")
	}
}

func TestSpicepodNormalizePlanModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name          string
		planValue     types.String
		expectedValue types.String
	}{
		{
			name:          "null value unchanged",
			planValue:     types.StringNull(),
			expectedValue: types.StringNull(),
		},
		{
			name:          "unknown value unchanged",
			planValue:     types.StringUnknown(),
			expectedValue: types.StringUnknown(),
		},
		{
			name:          "yaml normalized to json",
			planValue:     types.StringValue("version: v1beta1\nkind: Spicepod\nname: test"),
			expectedValue: types.StringValue(`{"kind":"Spicepod","name":"test","version":"v1beta1"}`),
		},
		{
			name:          "json normalized",
			planValue:     types.StringValue(`{"version":"v1beta1","kind":"Spicepod","name":"test"}`),
			expectedValue: types.StringValue(`{"kind":"Spicepod","name":"test","version":"v1beta1"}`),
		},
		{
			name:          "empty string becomes null",
			planValue:     types.StringValue(""),
			expectedValue: types.StringValue("null"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifier := spicepodNormalizePlanModifier{}
			ctx := context.Background()

			req := planmodifier.StringRequest{
				PlanValue: tt.planValue,
			}
			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			if tt.planValue.IsNull() {
				if !resp.PlanValue.IsNull() {
					t.Errorf("expected null, got %v", resp.PlanValue)
				}
				return
			}

			if tt.planValue.IsUnknown() {
				if !resp.PlanValue.IsUnknown() {
					t.Errorf("expected unknown, got %v", resp.PlanValue)
				}
				return
			}

			if resp.PlanValue.ValueString() != tt.expectedValue.ValueString() {
				t.Errorf("PlanModifyString() =\n%s\nwant:\n%s", resp.PlanValue.ValueString(), tt.expectedValue.ValueString())
			}
		})
	}
}

func TestSpicepodNormalizePlanModifier_RealWorldExample(t *testing.T) {
	// Simulate the real-world scenario from the bug report
	modifier := spicepodNormalizePlanModifier{}
	ctx := context.Background()

	// User provides YAML (like from file("spicepod.yaml"))
	userYAML := `version: v1beta1
kind: Spicepod
name: terraform-test-app

datasets:
  - name: test_dataset
    from: s3://spiceai-demo-datasets/taxi_trips/2024/
    params:
      file_format: parquet
`

	// API returns JSON
	apiJSON := `{"datasets":[{"from":"s3://spiceai-demo-datasets/taxi_trips/2024/","name":"test_dataset","params":{"file_format":"parquet"}}],"kind":"Spicepod","name":"terraform-test-app","version":"v1beta1"}`

	// Normalize user's YAML through plan modifier
	req := planmodifier.StringRequest{
		PlanValue: types.StringValue(userYAML),
	}
	resp := &planmodifier.StringResponse{
		PlanValue: types.StringValue(userYAML),
	}
	modifier.PlanModifyString(ctx, req, resp)
	normalizedUserValue := resp.PlanValue.ValueString()

	// Normalize API response (as done in mapAppToModel)
	normalizedAPIValue := normalizeSpicepodToJSON(apiJSON)

	// They should match!
	if normalizedUserValue != normalizedAPIValue {
		t.Errorf("User value and API value should match after normalization:\nUser (normalized): %s\nAPI (normalized):  %s", normalizedUserValue, normalizedAPIValue)
	}
}
