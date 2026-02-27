package api

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// TestLoadingStateComponentTemplate verifies the loading-state component template is valid and renders correctly
func TestLoadingStateComponentTemplate(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		contains    []string
		notContains []string
	}{
		{
			name: "spinner type (default)",
			data: map[string]interface{}{
				"Type": "spinner",
			},
			contains: []string{
				`role="status"`,
				`aria-label="Loading"`,
				`animate-spin`,
				`<span class="sr-only">Loading...</span>`,
			},
			notContains: []string{
				"animate-pulse",
			},
		},
		{
			name: "skeleton type with default rows (3)",
			data: map[string]interface{}{
				"Type": "skeleton",
			},
			contains: []string{
				`role="status"`,
				`aria-label="Loading content"`,
				`animate-pulse`,
				`<span class="sr-only">Loading...</span>`,
				`bg-surface-200 dark:bg-surface-700`,
			},
			notContains: []string{
				"animate-spin",
			},
		},
		{
			name: "skeleton type with 5 rows",
			data: map[string]interface{}{
				"Type": "skeleton",
				"Rows": 5,
			},
			contains: []string{
				`role="status"`,
				`animate-pulse`,
			},
		},
		{
			name: "skeleton type with 1 row",
			data: map[string]interface{}{
				"Type": "skeleton",
				"Rows": 1,
			},
			contains: []string{
				`role="status"`,
				`animate-pulse`,
			},
		},
		{
			name: "invalid type defaults to spinner",
			data: map[string]interface{}{
				"Type": "invalid",
			},
			contains: []string{
				`animate-spin`,
			},
			notContains: []string{
				"animate-pulse",
			},
		},
		{
			name: "spinner with custom class",
			data: map[string]interface{}{
				"Type":  "spinner",
				"Class": "my-custom-class",
			},
			contains: []string{
				`my-custom-class`,
				`animate-spin`,
			},
		},
		{
			name: "skeleton with custom class",
			data: map[string]interface{}{
				"Type":  "skeleton",
				"Class": "my-custom-class",
				"Rows":  2,
			},
			contains: []string{
				`my-custom-class`,
				`animate-pulse`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the loading-state component template
			tmpl, err := template.ParseFiles("../../web/templates/components/loading-state.html")
			if err != nil {
				t.Fatalf("Failed to parse loading-state template: %v", err)
			}

			// Execute the template
			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, "loading-state", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute loading-state template: %v", err)
			}

			output := buf.String()

			// Check for expected content
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}

			// Check for unexpected content
			for _, unexpected := range tt.notContains {
				if strings.Contains(output, unexpected) {
					t.Errorf("Expected output to NOT contain %q, but it did.\nOutput: %s", unexpected, output)
				}
			}
		})
	}
}

// TestLoadingStateComponentDarkModeClasses verifies dark mode classes are present
func TestLoadingStateComponentDarkModeClasses(t *testing.T) {
	tmpl, err := template.ParseFiles("../../web/templates/components/loading-state.html")
	if err != nil {
		t.Fatalf("Failed to parse loading-state template: %v", err)
	}

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "skeleton dark mode",
			data: map[string]interface{}{
				"Type": "skeleton",
				"Rows": 2,
			},
		},
		{
			name: "spinner dark mode",
			data: map[string]interface{}{
				"Type": "spinner",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, "loading-state", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute loading-state template: %v", err)
			}

			output := buf.String()

			// Check for dark mode classes
			if !strings.Contains(output, "dark:") {
				t.Errorf("Expected output to contain dark mode classes (dark:), but it didn't.\nOutput: %s", output)
			}
		})
	}
}

// TestLoadingStateComponentAccessibility verifies accessibility attributes
func TestLoadingStateComponentAccessibility(t *testing.T) {
	tmpl, err := template.ParseFiles("../../web/templates/components/loading-state.html")
	if err != nil {
		t.Fatalf("Failed to parse loading-state template: %v", err)
	}

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "skeleton accessibility",
			data: map[string]interface{}{
				"Type": "skeleton",
			},
		},
		{
			name: "spinner accessibility",
			data: map[string]interface{}{
				"Type": "spinner",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, "loading-state", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute loading-state template: %v", err)
			}

			output := buf.String()

			// Check for required accessibility attributes
			requiredAttrs := []string{
				`role="status"`,
				`aria-label=`,
				`<span class="sr-only">Loading...</span>`,
			}

			for _, attr := range requiredAttrs {
				if !strings.Contains(output, attr) {
					t.Errorf("Expected output to contain accessibility attribute %q, but it didn't.\nOutput: %s", attr, output)
				}
			}
		})
	}
}
