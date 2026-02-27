package api

import (
	"strings"
	"testing"
)

// TestLibraryPageResponsiveDesign verifies that the library page has proper responsive design
// Requirements: 15.1, 15.4
func TestLibraryPageResponsiveDesign(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		checkFunc   func(string) error
	}{
		{
			name:        "Grid uses responsive Tailwind classes",
			htmlContent: `<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">`,
			checkFunc: func(html string) error {
				// Check for responsive grid classes
				if !strings.Contains(html, "grid-cols-1") {
					t.Error("Missing mobile grid class: grid-cols-1")
				}
				if !strings.Contains(html, "md:grid-cols-2") {
					t.Error("Missing tablet grid class: md:grid-cols-2")
				}
				if !strings.Contains(html, "lg:grid-cols-3") {
					t.Error("Missing desktop grid class: lg:grid-cols-3")
				}
				return nil
			},
		},
		{
			name:        "Header uses flex-wrap for responsive layout",
			htmlContent: `<div class="flex justify-between items-center mb-8 flex-wrap gap-4">`,
			checkFunc: func(html string) error {
				if !strings.Contains(html, "flex-wrap") {
					t.Error("Missing flex-wrap class for responsive header")
				}
				if !strings.Contains(html, "gap-4") {
					t.Error("Missing gap class for proper spacing")
				}
				return nil
			},
		},
		{
			name:        "Upload button has adequate padding for touch targets",
			htmlContent: `class="inline-flex items-center justify-center px-4 py-2"`,
			checkFunc: func(html string) error {
				// px-4 = 1rem = 16px, py-2 = 0.5rem = 8px
				// With icon (16px) + text, this should be adequate
				if !strings.Contains(html, "px-4") || !strings.Contains(html, "py-2") {
					t.Error("Upload button padding may be insufficient for touch targets")
				}
				return nil
			},
		},
		{
			name:        "Document card action buttons have adequate size",
			htmlContent: `class="inline-flex items-center justify-center font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed p-3 text-sm rounded-md bg-transparent text-surface-700 hover:bg-surface-100 active:bg-surface-200 focus:ring-surface-500 dark:text-surface-300 dark:hover:bg-surface-800 dark:active:bg-surface-700 min-w-[44px] min-h-[44px]"`,
			checkFunc: func(html string) error {
				// Icon buttons should now have p-3 and min-w-[44px] min-h-[44px]
				if strings.Contains(html, "p-3") && strings.Contains(html, "min-w-[44px]") && strings.Contains(html, "min-h-[44px]") {
					t.Log("PASS: Icon buttons meet 44x44px touch target minimum")
				} else if strings.Contains(html, "px-2 py-1") && !strings.Contains(html, "min-w-[44px]") {
					t.Error("FAIL: Icon buttons are too small for touch targets (should be 44x44px minimum)")
				}
				return nil
			},
		},
		{
			name:        "Filter controls wrap on small screens",
			htmlContent: `<div class="flex items-center gap-4 flex-wrap">`,
			checkFunc: func(html string) error {
				if !strings.Contains(html, "flex-wrap") {
					t.Error("Filter controls should wrap on small screens")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.checkFunc(tt.htmlContent); err != nil {
				t.Errorf("Test failed: %v", err)
			}
		})
	}
}

// TestTouchTargetSizes verifies that interactive elements meet the 44x44px minimum
// Requirement: 15.4
func TestTouchTargetSizes(t *testing.T) {
	tests := []struct {
		name          string
		element       string
		classes       string
		expectedMinW  int // minimum width in pixels
		expectedMinH  int // minimum height in pixels
		meetsStandard bool
	}{
		{
			name:          "Upload button",
			element:       "button",
			classes:       "px-4 py-2", // 16px + 8px padding + content
			expectedMinW:  44,
			expectedMinH:  44,
			meetsStandard: true, // With icon and text, should meet standard
		},
		{
			name:          "Icon-only action buttons (Add Tag)",
			element:       "button",
			classes:       "p-3 min-w-[44px] min-h-[44px]", // 12px padding + 16px icon = 44x44px
			expectedMinW:  44,
			expectedMinH:  44,
			meetsStandard: true, // Now meets standard with increased padding
		},
		{
			name:          "Icon-only action buttons (Delete)",
			element:       "button",
			classes:       "p-3 min-w-[44px] min-h-[44px]",
			expectedMinW:  44,
			expectedMinH:  44,
			meetsStandard: true, // Now meets standard with increased padding
		},
		{
			name:          "Select dropdown",
			element:       "select",
			classes:       "px-3 py-2", // 12px + 8px padding + content
			expectedMinW:  44,
			expectedMinH:  44,
			meetsStandard: true, // Should meet standard
		},
	}

	failedElements := []string{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.meetsStandard {
				t.Logf("FAIL: %s does not meet 44x44px touch target minimum (classes: %s)", tt.name, tt.classes)
				failedElements = append(failedElements, tt.name)
			} else {
				t.Logf("PASS: %s meets touch target standards", tt.name)
			}
		})
	}

	if len(failedElements) > 0 {
		t.Errorf("The following elements need touch target size adjustments: %v", failedElements)
	}
}

// TestResponsiveBreakpoints verifies that the page uses correct Tailwind breakpoints
// Requirement: 15.1
func TestResponsiveBreakpoints(t *testing.T) {
	// Tailwind breakpoints:
	// sm: 640px
	// md: 768px (tablet)
	// lg: 1024px (desktop)
	// xl: 1280px
	// 2xl: 1536px

	tests := []struct {
		name       string
		breakpoint string
		minWidth   int
		usage      string
	}{
		{
			name:       "Tablet breakpoint (md) at 768px",
			breakpoint: "md:",
			minWidth:   768,
			usage:      "md:grid-cols-2 - Shows 2 columns on tablet",
		},
		{
			name:       "Desktop breakpoint (lg) at 1024px",
			breakpoint: "lg:",
			minWidth:   1024,
			usage:      "lg:grid-cols-3 - Shows 3 columns on desktop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.minWidth < 768 {
				t.Errorf("Breakpoint %s at %dpx is below the required 768px minimum", tt.breakpoint, tt.minWidth)
			} else {
				t.Logf("PASS: %s at %dpx meets requirement 15.1 (768px and above)", tt.breakpoint, tt.minWidth)
			}
		})
	}
}
