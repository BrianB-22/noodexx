package ingest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Guardrails enforces safety checks on ingestion
type Guardrails struct {
	MaxFileSize        int64
	AllowedExtensions  []string
	BlockedExtensions  []string
	SensitiveFilenames []string
	MaxConcurrent      int
}

// NewGuardrails creates guardrails with safe defaults
func NewGuardrails() *Guardrails {
	return &Guardrails{
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		AllowedExtensions: []string{".txt", ".md", ".pdf", ".html"},
		BlockedExtensions: []string{
			// Executables
			".exe", ".dll", ".so", ".dylib", ".app",
			// Archives
			".zip", ".tar", ".gz", ".rar",
			// Disk images
			".iso", ".dmg", ".img",
		},
		SensitiveFilenames: []string{
			".env", "id_rsa", "id_ed25519", "credentials.json",
			".aws/credentials", ".ssh/id_rsa",
		},
		MaxConcurrent: 3,
	}
}

// Check validates a file for ingestion
func (g *Guardrails) Check(filename, content string) error {
	// Check sensitive filenames
	for _, sensitive := range g.SensitiveFilenames {
		if strings.Contains(strings.ToLower(filename), strings.ToLower(sensitive)) {
			return fmt.Errorf("sensitive filename detected: %s", filename)
		}
	}

	// Check blocked extensions
	ext := strings.ToLower(filepath.Ext(filename))
	for _, blocked := range g.BlockedExtensions {
		if ext == blocked {
			return fmt.Errorf("blocked file extension: %s", ext)
		}
	}

	return nil
}

// IsAllowedExtension checks if a file extension is allowed
func (g *Guardrails) IsAllowedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	for _, allowed := range g.AllowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}
