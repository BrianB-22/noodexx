package ingest

import (
	"testing"
)

func TestNewGuardrails(t *testing.T) {
	g := NewGuardrails()

	// Verify defaults
	if g.MaxFileSize != 10*1024*1024 {
		t.Errorf("Expected MaxFileSize to be 10MB, got %d", g.MaxFileSize)
	}

	if len(g.AllowedExtensions) != 4 {
		t.Errorf("Expected 4 allowed extensions, got %d", len(g.AllowedExtensions))
	}

	if len(g.BlockedExtensions) == 0 {
		t.Error("Expected blocked extensions to be populated")
	}

	if len(g.SensitiveFilenames) == 0 {
		t.Error("Expected sensitive filenames to be populated")
	}

	if g.MaxConcurrent != 3 {
		t.Errorf("Expected MaxConcurrent to be 3, got %d", g.MaxConcurrent)
	}
}

func TestIsAllowedExtension(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		ext      string
		expected bool
	}{
		{".txt", true},
		{".md", true},
		{".pdf", true},
		{".html", true},
		{".TXT", true}, // Case insensitive
		{".exe", false},
		{".zip", false},
		{".doc", false},
	}

	for _, tt := range tests {
		result := g.IsAllowedExtension(tt.ext)
		if result != tt.expected {
			t.Errorf("IsAllowedExtension(%q) = %v, expected %v", tt.ext, result, tt.expected)
		}
	}
}

func TestCheckBlockedExtensions(t *testing.T) {
	g := NewGuardrails()

	blockedFiles := []string{
		"malware.exe",
		"library.dll",
		"library.so",
		"app.dylib",
		"program.app",
		"archive.zip",
		"backup.tar",
		"compressed.gz",
		"archive.rar",
		"disk.iso",
		"installer.dmg",
		"disk.img",
	}

	for _, filename := range blockedFiles {
		err := g.Check(filename, "")
		if err == nil {
			t.Errorf("Expected Check(%q) to return error for blocked extension", filename)
		}
	}
}

func TestCheckSensitiveFilenames(t *testing.T) {
	g := NewGuardrails()

	sensitiveFiles := []string{
		".env",
		"id_rsa",
		"id_ed25519",
		"credentials.json",
		".aws/credentials",
		".ssh/id_rsa",
		"my_id_rsa.txt",    // Contains sensitive pattern
		"backup/.env",      // Contains sensitive pattern
		"CREDENTIALS.JSON", // Case insensitive
	}

	for _, filename := range sensitiveFiles {
		err := g.Check(filename, "")
		if err == nil {
			t.Errorf("Expected Check(%q) to return error for sensitive filename", filename)
		}
	}
}

func TestCheckAllowedFiles(t *testing.T) {
	g := NewGuardrails()

	allowedFiles := []string{
		"document.txt",
		"README.md",
		"report.pdf",
		"page.html",
		"notes.txt",
	}

	for _, filename := range allowedFiles {
		err := g.Check(filename, "")
		if err != nil {
			t.Errorf("Expected Check(%q) to pass, got error: %v", filename, err)
		}
	}
}
