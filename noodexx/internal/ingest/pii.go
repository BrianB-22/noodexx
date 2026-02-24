package ingest

import "regexp"

// PIIDetector scans text for personally identifiable information
type PIIDetector struct {
	patterns map[string]*regexp.Regexp
}

// NewPIIDetector creates a detector with common PII patterns
func NewPIIDetector() *PIIDetector {
	return &PIIDetector{
		patterns: map[string]*regexp.Regexp{
			"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
			"credit_card": regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
			"api_key":     regexp.MustCompile(`\b(sk-[a-zA-Z0-9]{32,}|ghp_[a-zA-Z0-9]{36}|xox[baprs]-[a-zA-Z0-9-]+)\b`),
			"private_key": regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----`),
			"email":       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
			"phone":       regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		},
	}
}

// Detect returns a list of PII types found in the text
func (d *PIIDetector) Detect(text string) []string {
	var found []string

	for piiType, pattern := range d.patterns {
		if pattern.MatchString(text) {
			found = append(found, piiType)
		}
	}

	return found
}
