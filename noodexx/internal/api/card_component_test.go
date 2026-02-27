package api

import (
	"bytes"
	"html/template"
	"testing"
)

// TestCardComponentTemplate verifies the card component template is valid and renders correctly
func TestCardComponentTemplate(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		contains []string
	}{
		{
			name: "card with title and content",
			data: map[string]interface{}{
				"Title":   "Test Card",
				"Content": template.HTML("<p>Card content</p>"),
			},
			contains: []string{
				"bg-white",
				"dark:bg-surface-800",
				"rounded-lg",
				"shadow-md",
				"border",
				"border-surface-200",
				"dark:border-surface-700",
				"p-6",
				"<h3",
				"Test Card",
				"<p>Card content</p>",
			},
		},
		{
			name: "card without title",
			data: map[string]interface{}{
				"Content": template.HTML("<div>Content only</div>"),
			},
			contains: []string{
				"bg-white",
				"dark:bg-surface-800",
				"<div>Content only</div>",
			},
		},
		{
			name: "card with additional classes",
			data: map[string]interface{}{
				"Class":   "mt-4 custom-class",
				"Content": template.HTML("<span>Test</span>"),
			},
			contains: []string{
				"bg-white",
				"mt-4",
				"custom-class",
			},
		},
		{
			name: "card with ID",
			data: map[string]interface{}{
				"ID":      "my-card",
				"Content": template.HTML("<p>Test</p>"),
			},
			contains: []string{
				`id="my-card"`,
			},
		},
		{
			name: "card with all props",
			data: map[string]interface{}{
				"ID":      "full-card",
				"Title":   "Full Card",
				"Class":   "extra-class",
				"Content": template.HTML("<div>Full content</div>"),
			},
			contains: []string{
				`id="full-card"`,
				"Full Card",
				"extra-class",
				"<div>Full content</div>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the card component template
			tmpl, err := template.ParseFiles("../../web/templates/components/card.html")
			if err != nil {
				t.Fatalf("Failed to parse card template: %v", err)
			}

			// Execute the template
			var buf bytes.Buffer
			err = tmpl.ExecuteTemplate(&buf, "card", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute card template: %v", err)
			}

			// Check that all expected strings are present
			output := buf.String()
			for _, expected := range tt.contains {
				if !bytes.Contains([]byte(output), []byte(expected)) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput: %s", expected, output)
				}
			}
		})
	}
}

// TestCardComponentDarkModeClasses verifies dark mode classes are present
func TestCardComponentDarkModeClasses(t *testing.T) {
	tmpl, err := template.ParseFiles("../../web/templates/components/card.html")
	if err != nil {
		t.Fatalf("Failed to parse card template: %v", err)
	}

	data := map[string]interface{}{
		"Title":   "Test Title",
		"Content": template.HTML("<p>Test</p>"),
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "card", data)
	if err != nil {
		t.Fatalf("Failed to execute card template: %v", err)
	}

	output := buf.String()
	darkModeClasses := []string{
		"dark:bg-surface-800",
		"dark:border-surface-700",
		"dark:text-surface-100", // This is on the title element
	}

	for _, class := range darkModeClasses {
		if !bytes.Contains([]byte(output), []byte(class)) {
			t.Errorf("Expected dark mode class %q to be present in output", class)
		}
	}
}
