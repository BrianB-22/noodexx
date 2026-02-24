package ingest

import (
	"testing"
)

func TestPIIDetector_SSN(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "valid SSN",
			text:     "My SSN is 123-45-6789",
			expected: true,
		},
		{
			name:     "no SSN",
			text:     "This is just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasSSN := false
			for _, piiType := range found {
				if piiType == "ssn" {
					hasSSN = true
					break
				}
			}
			if hasSSN != tt.expected {
				t.Errorf("expected SSN detection = %v, got %v", tt.expected, hasSSN)
			}
		})
	}
}

func TestPIIDetector_CreditCard(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "credit card with spaces",
			text:     "Card: 4532 1234 5678 9010",
			expected: true,
		},
		{
			name:     "credit card with dashes",
			text:     "Card: 4532-1234-5678-9010",
			expected: true,
		},
		{
			name:     "credit card no separators",
			text:     "Card: 4532123456789010",
			expected: true,
		},
		{
			name:     "no credit card",
			text:     "Just some text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasCC := false
			for _, piiType := range found {
				if piiType == "credit_card" {
					hasCC = true
					break
				}
			}
			if hasCC != tt.expected {
				t.Errorf("expected credit card detection = %v, got %v", tt.expected, hasCC)
			}
		})
	}
}

func TestPIIDetector_APIKey(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "OpenAI API key",
			text:     "My key is sk-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "GitHub token",
			text:     "Token: ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: true,
		},
		{
			name:     "Slack token",
			text:     "Slack: xoxb-FAKE-TEST-TOKEN-NOT-REAL",
			expected: true,
		},
		{
			name:     "no API key",
			text:     "Just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasKey := false
			for _, piiType := range found {
				if piiType == "api_key" {
					hasKey = true
					break
				}
			}
			if hasKey != tt.expected {
				t.Errorf("expected API key detection = %v, got %v", tt.expected, hasKey)
			}
		})
	}
}

func TestPIIDetector_PrivateKey(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "RSA private key",
			text:     "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...",
			expected: true,
		},
		{
			name:     "EC private key",
			text:     "-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEII...",
			expected: true,
		},
		{
			name:     "OpenSSH private key",
			text:     "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEA...",
			expected: true,
		},
		{
			name:     "generic private key",
			text:     "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0B...",
			expected: true,
		},
		{
			name:     "no private key",
			text:     "Just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasKey := false
			for _, piiType := range found {
				if piiType == "private_key" {
					hasKey = true
					break
				}
			}
			if hasKey != tt.expected {
				t.Errorf("expected private key detection = %v, got %v", tt.expected, hasKey)
			}
		})
	}
}

func TestPIIDetector_Email(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "valid email",
			text:     "Contact me at user@example.com",
			expected: true,
		},
		{
			name:     "email with subdomain",
			text:     "Email: admin@mail.example.com",
			expected: true,
		},
		{
			name:     "no email",
			text:     "Just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasEmail := false
			for _, piiType := range found {
				if piiType == "email" {
					hasEmail = true
					break
				}
			}
			if hasEmail != tt.expected {
				t.Errorf("expected email detection = %v, got %v", tt.expected, hasEmail)
			}
		})
	}
}

func TestPIIDetector_Phone(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "phone with dashes",
			text:     "Call me at 555-123-4567",
			expected: true,
		},
		{
			name:     "phone with dots",
			text:     "Phone: 555.123.4567",
			expected: true,
		},
		{
			name:     "phone no separators",
			text:     "Number: 5551234567",
			expected: true,
		},
		{
			name:     "no phone",
			text:     "Just regular text",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := detector.Detect(tt.text)
			hasPhone := false
			for _, piiType := range found {
				if piiType == "phone" {
					hasPhone = true
					break
				}
			}
			if hasPhone != tt.expected {
				t.Errorf("expected phone detection = %v, got %v", tt.expected, hasPhone)
			}
		})
	}
}

func TestPIIDetector_MultiplePII(t *testing.T) {
	detector := NewPIIDetector()

	text := `
		Contact: user@example.com
		Phone: 555-123-4567
		SSN: 123-45-6789
		API Key: sk-1234567890abcdefghijklmnopqrstuvwxyz
	`

	found := detector.Detect(text)

	expectedTypes := map[string]bool{
		"email":   true,
		"phone":   true,
		"ssn":     true,
		"api_key": true,
	}

	if len(found) != len(expectedTypes) {
		t.Errorf("expected %d PII types, got %d", len(expectedTypes), len(found))
	}

	for _, piiType := range found {
		if !expectedTypes[piiType] {
			t.Errorf("unexpected PII type detected: %s", piiType)
		}
	}
}

func TestPIIDetector_NoPII(t *testing.T) {
	detector := NewPIIDetector()

	text := "This is just regular text with no sensitive information."

	found := detector.Detect(text)

	if len(found) != 0 {
		t.Errorf("expected no PII, but found: %v", found)
	}
}
