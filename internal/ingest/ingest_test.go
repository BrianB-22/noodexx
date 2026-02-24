package ingest

import (
	"context"
	"io"
	"mime/multipart"
	"strings"
	"testing"
)

// Mock implementations for testing

type mockProvider struct {
	embedFunc  func(ctx context.Context, text string) ([]float32, error)
	streamFunc func(ctx context.Context, messages []Message, w io.Writer) (string, error)
}

func (m *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFunc != nil {
		return m.embedFunc(ctx, text)
	}
	// Return a simple mock embedding
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockProvider) Stream(ctx context.Context, messages []Message, w io.Writer) (string, error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, messages, w)
	}
	// Return a simple mock summary
	summary := "This is a test summary."
	if w != nil {
		w.Write([]byte(summary))
	}
	return summary, nil
}

type mockStore struct {
	chunks []struct {
		source    string
		text      string
		embedding []float32
		tags      []string
		summary   string
	}
}

func (m *mockStore) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	m.chunks = append(m.chunks, struct {
		source    string
		text      string
		embedding []float32
		tags      []string
		summary   string
	}{source, text, embedding, tags, summary})
	return nil
}

type mockChunker struct {
	chunkSize int
}

func (m *mockChunker) ChunkText(text string) []string {
	// Simple chunking for testing
	if len(text) <= m.chunkSize {
		return []string{text}
	}
	var chunks []string
	for i := 0; i < len(text); i += m.chunkSize {
		end := i + m.chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}

func TestIngestText_Basic(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false)

	ctx := context.Background()
	err := ingester.IngestText(ctx, "test.txt", "This is a test document.", []string{"test"})

	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	if len(store.chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(store.chunks))
	}

	if store.chunks[0].source != "test.txt" {
		t.Errorf("Expected source 'test.txt', got '%s'", store.chunks[0].source)
	}
}

func TestIngestText_WithSummary(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, true)

	ctx := context.Background()
	err := ingester.IngestText(ctx, "test.txt", "This is a test document.", []string{"test"})

	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}

	if len(store.chunks) != 1 {
		t.Errorf("Expected 1 chunk, got %d", len(store.chunks))
	}

	if store.chunks[0].summary == "" {
		t.Error("Expected summary to be generated, but it was empty")
	}
}

func TestIngestText_PIIDetection(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false)

	ctx := context.Background()
	// Text with SSN pattern
	err := ingester.IngestText(ctx, "test.txt", "My SSN is 123-45-6789", []string{"test"})

	if err == nil {
		t.Fatal("Expected PII detection error, got nil")
	}

	if !strings.Contains(err.Error(), "PII detected") {
		t.Errorf("Expected PII detection error, got: %v", err)
	}

	if len(store.chunks) != 0 {
		t.Error("Expected no chunks to be saved when PII is detected")
	}
}

func TestIngestText_GuardrailsCheck(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false)

	ctx := context.Background()
	// Sensitive filename
	err := ingester.IngestText(ctx, ".env", "SECRET_KEY=abc123", []string{"test"})

	if err == nil {
		t.Fatal("Expected guardrails error, got nil")
	}

	if !strings.Contains(err.Error(), "guardrails check failed") {
		t.Errorf("Expected guardrails error, got: %v", err)
	}

	if len(store.chunks) != 0 {
		t.Error("Expected no chunks to be saved when guardrails fail")
	}
}

func TestIngestURL_PrivacyMode(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, true, false)

	ctx := context.Background()
	err := ingester.IngestURL(ctx, "https://example.com", []string{"test"})

	if err == nil {
		t.Fatal("Expected privacy mode error, got nil")
	}

	if !strings.Contains(err.Error(), "privacy mode") {
		t.Errorf("Expected privacy mode error, got: %v", err)
	}
}

func TestIngestFile_InvalidExtension(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false)

	ctx := context.Background()

	// Create a mock file
	var file multipart.File = nil
	header := &multipart.FileHeader{
		Filename: "test.exe",
		Size:     100,
	}

	err := ingester.IngestFile(ctx, file, header, []string{"test"})

	if err == nil {
		t.Fatal("Expected extension error, got nil")
	}

	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("Expected extension error, got: %v", err)
	}
}

func TestIngestFile_OversizedFile(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false)

	ctx := context.Background()

	// Create a mock file that exceeds size limit
	var file multipart.File = nil
	header := &multipart.FileHeader{
		Filename: "test.txt",
		Size:     20 * 1024 * 1024, // 20MB (exceeds 10MB limit)
	}

	err := ingester.IngestFile(ctx, file, header, []string{"test"})

	if err == nil {
		t.Fatal("Expected size limit error, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

func TestGenerateSummary_TruncatesLongText(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{
		streamFunc: func(ctx context.Context, messages []Message, w io.Writer) (string, error) {
			// Verify that the input was truncated to 1000 characters
			if len(messages) > 0 && len(messages[0].Content) > 1100 {
				t.Error("Expected input to be truncated to ~1000 characters")
			}
			summary := "Short summary"
			if w != nil {
				w.Write([]byte(summary))
			}
			return summary, nil
		},
	}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, true)

	ctx := context.Background()
	// Create a long text (more than 1000 characters)
	longText := strings.Repeat("This is a long document. ", 100)

	err := ingester.IngestText(ctx, "test.txt", longText, []string{"test"})

	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}
}
