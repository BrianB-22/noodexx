package uistyle

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// UIStyleConfig represents the centralized theme configuration
type UIStyleConfig struct {
	Colors       ColorScheme      `json:"colors"`
	Typography   TypographyConfig `json:"typography"`
	Spacing      SpacingConfig    `json:"spacing"`
	BorderRadius RadiusConfig     `json:"border_radius"`
	Shadows      ShadowConfig     `json:"shadows"`
}

// ColorScheme contains all semantic color palettes
type ColorScheme struct {
	Primary   ColorPalette `json:"primary"`
	Secondary ColorPalette `json:"secondary"`
	Success   ColorPalette `json:"success"`
	Warning   ColorPalette `json:"warning"`
	Error     ColorPalette `json:"error"`
	Info      ColorPalette `json:"info"`
	Surface   ColorPalette `json:"surface"`
}

// ColorPalette represents a color with shades from 50 to 900
type ColorPalette struct {
	Shade50  string `json:"50"`
	Shade100 string `json:"100"`
	Shade200 string `json:"200"`
	Shade300 string `json:"300"`
	Shade400 string `json:"400"`
	Shade500 string `json:"500"`
	Shade600 string `json:"600"`
	Shade700 string `json:"700"`
	Shade800 string `json:"800"`
	Shade900 string `json:"900"`
}

// TypographyConfig contains font family and size definitions
type TypographyConfig struct {
	FontFamily FontFamilies `json:"font_family"`
	FontSizes  FontSizes    `json:"font_sizes"`
}

// FontFamilies defines font stacks for different text types
type FontFamilies struct {
	Sans []string `json:"sans"`
	Mono []string `json:"mono"`
}

// FontSizes defines the type scale
type FontSizes struct {
	XS   string `json:"xs"`
	SM   string `json:"sm"`
	Base string `json:"base"`
	LG   string `json:"lg"`
	XL   string `json:"xl"`
	XL2  string `json:"2xl"`
	XL3  string `json:"3xl"`
}

// SpacingConfig defines the spacing scale
type SpacingConfig struct {
	Unit  string            `json:"unit"`
	Scale map[string]string `json:"scale"`
}

// RadiusConfig defines border radius values
type RadiusConfig struct {
	None string `json:"none"`
	SM   string `json:"sm"`
	Base string `json:"base"`
	MD   string `json:"md"`
	LG   string `json:"lg"`
	XL   string `json:"xl"`
	Full string `json:"full"`
}

// ShadowConfig defines box shadow values
type ShadowConfig struct {
	SM   string `json:"sm"`
	Base string `json:"base"`
	MD   string `json:"md"`
	LG   string `json:"lg"`
	XL   string `json:"xl"`
}

// LoadUIStyle loads and validates the UIStyle configuration from the specified path
func LoadUIStyle(path string) (*UIStyleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load uistyle.json: %w", err)
	}

	var config UIStyleConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse uistyle.json: %w", err)
	}

	if err := validateUIStyle(&config); err != nil {
		return nil, fmt.Errorf("invalid uistyle.json: %w", err)
	}

	return &config, nil
}

// validateUIStyle validates the entire UIStyle configuration
func validateUIStyle(config *UIStyleConfig) error {
	// Validate all color palettes
	palettes := []struct {
		name    string
		palette ColorPalette
	}{
		{"primary", config.Colors.Primary},
		{"secondary", config.Colors.Secondary},
		{"success", config.Colors.Success},
		{"warning", config.Colors.Warning},
		{"error", config.Colors.Error},
		{"info", config.Colors.Info},
		{"surface", config.Colors.Surface},
	}

	for _, p := range palettes {
		if err := validateColorPalette(p.name, p.palette); err != nil {
			return err
		}
	}

	return nil
}

// validateColorPalette validates that a color palette has all required shades with valid hex colors
func validateColorPalette(name string, palette ColorPalette) error {
	requiredShades := map[string]string{
		"50":  palette.Shade50,
		"100": palette.Shade100,
		"200": palette.Shade200,
		"300": palette.Shade300,
		"400": palette.Shade400,
		"500": palette.Shade500,
		"600": palette.Shade600,
		"700": palette.Shade700,
		"800": palette.Shade800,
		"900": palette.Shade900,
	}

	for shade, color := range requiredShades {
		if color == "" {
			return fmt.Errorf("%s palette missing shade: %s", name, shade)
		}
		if !isValidHexColor(color) {
			return fmt.Errorf("%s palette has invalid hex color for shade %s: %s", name, shade, color)
		}
	}

	return nil
}

// isValidHexColor validates that a color string is a valid hex color in #RRGGBB format
func isValidHexColor(color string) bool {
	matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, color)
	return matched
}
