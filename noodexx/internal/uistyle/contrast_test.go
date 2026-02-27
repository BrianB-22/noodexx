package uistyle

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
)

// TestDarkModeContrastRatios verifies that all text/background combinations
// in dark mode meet WCAG AA contrast requirements (4.5:1 for normal text)
func TestDarkModeContrastRatios(t *testing.T) {
	// Load uistyle.json
	data, err := os.ReadFile("../../uistyle.json")
	if err != nil {
		t.Fatalf("Failed to load uistyle.json: %v", err)
	}

	var config UIStyleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse uistyle.json: %v", err)
	}

	// Define dark mode text/background combinations used in dashboard
	testCases := []struct {
		name      string
		textColor string
		bgColor   string
		minRatio  float64
		location  string
	}{
		// Dashboard header
		{
			name:      "Dashboard header text on dark background",
			textColor: config.Colors.Surface.Shade100, // dark:text-surface-100
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "dashboard.html - h1 title",
		},
		// Card backgrounds and text
		{
			name:      "Card title text on dark card background",
			textColor: config.Colors.Surface.Shade100, // dark:text-surface-100
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "card.html - h3 title",
		},
		{
			name:      "Card body text on dark card background",
			textColor: config.Colors.Surface.Shade100, // dark:text-surface-100
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "dashboard.html - stat values",
		},
		{
			name:      "Card secondary text on dark card background",
			textColor: config.Colors.Surface.Shade400, // dark:text-surface-400
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "dashboard.html - stat labels",
		},
		// Primary button
		{
			name:      "Primary button text on dark primary background",
			textColor: "#ffffff",                      // text-white
			bgColor:   config.Colors.Primary.Shade600, // dark:bg-primary-600
			minRatio:  4.5,
			location:  "button.html - primary variant",
		},
		// Secondary button
		{
			name:      "Secondary button text on dark secondary background",
			textColor: config.Colors.Surface.Shade100, // dark:text-surface-100
			bgColor:   config.Colors.Surface.Shade700, // dark:bg-surface-700
			minRatio:  4.5,
			location:  "button.html - secondary variant",
		},
		// Ghost button
		{
			name:      "Ghost button text on dark page background",
			textColor: config.Colors.Surface.Shade300, // dark:text-surface-300
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800 (page background)
			minRatio:  4.5,
			location:  "button.html - ghost variant",
		},
		// Success text (Privacy Mode card)
		{
			name:      "Success text on dark card background",
			textColor: config.Colors.Success.Shade400, // dark:text-success-400
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "dashboard.html - Privacy Mode card",
		},
		// Sidebar navigation
		{
			name:      "Sidebar text on dark sidebar background",
			textColor: config.Colors.Surface.Shade400, // dark:text-surface-400
			bgColor:   config.Colors.Surface.Shade900, // dark:bg-surface-900
			minRatio:  4.5,
			location:  "base.html - sidebar navigation",
		},
		// Main content area
		{
			name:      "Main content text on dark main background",
			textColor: config.Colors.Surface.Shade100, // dark:text-surface-100
			bgColor:   config.Colors.Surface.Shade800, // dark:bg-surface-800
			minRatio:  4.5,
			location:  "base.html - main content area",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ratio := calculateContrastRatio(tc.textColor, tc.bgColor)

			if ratio < tc.minRatio {
				t.Errorf(
					"Contrast ratio %.2f:1 is below minimum %.1f:1\n"+
						"  Location: %s\n"+
						"  Text color: %s\n"+
						"  Background color: %s\n"+
						"  WCAG AA requires 4.5:1 for normal text",
					ratio, tc.minRatio, tc.location, tc.textColor, tc.bgColor,
				)
			} else {
				t.Logf(
					"✓ Contrast ratio %.2f:1 meets WCAG AA (%.1f:1)\n"+
						"  Location: %s\n"+
						"  Text: %s on Background: %s",
					ratio, tc.minRatio, tc.location, tc.textColor, tc.bgColor,
				)
			}
		})
	}
}

// calculateContrastRatio calculates the WCAG contrast ratio between two colors
// Formula: (L1 + 0.05) / (L2 + 0.05) where L1 is the lighter color
func calculateContrastRatio(color1, color2 string) float64 {
	l1 := relativeLuminance(color1)
	l2 := relativeLuminance(color2)

	// Ensure L1 is the lighter color
	if l1 < l2 {
		l1, l2 = l2, l1
	}

	return (l1 + 0.05) / (l2 + 0.05)
}

// relativeLuminance calculates the relative luminance of a color
// Formula from WCAG 2.1: https://www.w3.org/TR/WCAG21/#dfn-relative-luminance
func relativeLuminance(hexColor string) float64 {
	r, g, b := hexToRGB(hexColor)

	// Convert to 0-1 range
	rSRGB := float64(r) / 255.0
	gSRGB := float64(g) / 255.0
	bSRGB := float64(b) / 255.0

	// Apply gamma correction
	rLinear := gammaCorrect(rSRGB)
	gLinear := gammaCorrect(gSRGB)
	bLinear := gammaCorrect(bSRGB)

	// Calculate relative luminance
	return 0.2126*rLinear + 0.7152*gLinear + 0.0722*bLinear
}

// gammaCorrect applies gamma correction to an sRGB color component
func gammaCorrect(component float64) float64 {
	if component <= 0.03928 {
		return component / 12.92
	}
	return math.Pow((component+0.055)/1.055, 2.4)
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hexColor string) (uint8, uint8, uint8) {
	// Remove # prefix if present
	if len(hexColor) > 0 && hexColor[0] == '#' {
		hexColor = hexColor[1:]
	}

	// Parse hex string
	if len(hexColor) != 6 {
		panic(fmt.Sprintf("Invalid hex color: %s", hexColor))
	}

	r, _ := strconv.ParseUint(hexColor[0:2], 16, 8)
	g, _ := strconv.ParseUint(hexColor[2:4], 16, 8)
	b, _ := strconv.ParseUint(hexColor[4:6], 16, 8)

	return uint8(r), uint8(g), uint8(b)
}
