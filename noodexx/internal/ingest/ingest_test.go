package ingest

import (
	"context"
	"io"
	"mime/multipart"
	"noodexx/internal/logging"
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
		userID    int64
		source    string
		text      string
		embedding []float32
		tags      []string
		summary   string
	}
}

func (m *mockStore) SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error {
	m.chunks = append(m.chunks, struct {
		userID    int64
		source    string
		text      string
		embedding []float32
		tags      []string
		summary   string
	}{userID, source, text, embedding, tags, summary})
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

// Helper function to create a test logger
func newTestLogger() *logging.Logger {
	return logging.NewLogger("test", logging.DEBUG, io.Discard)
}

func TestIngestText_Basic(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()
	err := ingester.IngestText(ctx, 1, "test.txt", "This is a test document.", []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, true, newTestLogger())

	ctx := context.Background()
	err := ingester.IngestText(ctx, 1, "test.txt", "This is a test document.", []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()
	// Text with SSN pattern
	err := ingester.IngestText(ctx, 1, "test.txt", "My SSN is 123-45-6789", []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()
	// Sensitive filename
	err := ingester.IngestText(ctx, 1, ".env", "SECRET_KEY=abc123", []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, true, false, newTestLogger())

	ctx := context.Background()
	err := ingester.IngestURL(ctx, 1, "https://example.com", []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()

	// Create a mock file
	var file multipart.File = nil
	header := &multipart.FileHeader{
		Filename: "test.exe",
		Size:     100,
	}

	err := ingester.IngestFile(ctx, 1, file, header, []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()

	// Create a mock file that exceeds size limit
	var file multipart.File = nil
	header := &multipart.FileHeader{
		Filename: "test.txt",
		Size:     20 * 1024 * 1024, // 20MB (exceeds 10MB limit)
	}

	err := ingester.IngestFile(ctx, 1, file, header, []string{"test"})

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

	ingester := NewIngester(provider, store, chunker, false, true, newTestLogger())

	ctx := context.Background()
	// Create a long text (more than 1000 characters)
	longText := strings.Repeat("This is a long document. ", 100)

	err := ingester.IngestText(ctx, 1, "test.txt", longText, []string{"test"})

	if err != nil {
		t.Fatalf("IngestText failed: %v", err)
	}
}

func TestIngestFile_TextFile(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()

	// Create a mock text file
	content := "This is a test text file content."
	file := &mockFile{content: content}
	header := &multipart.FileHeader{
		Filename: "test.txt",
		Size:     int64(len(content)),
	}

	err := ingester.IngestFile(ctx, 1, file, header, []string{"test"})

	if err != nil {
		t.Fatalf("IngestFile failed: %v", err)
	}

	if len(store.chunks) == 0 {
		t.Fatal("Expected chunks to be saved")
	}

	if store.chunks[0].source != "test.txt" {
		t.Errorf("Expected source 'test.txt', got '%s'", store.chunks[0].source)
	}

	if store.chunks[0].text != content {
		t.Errorf("Expected text '%s', got '%s'", content, store.chunks[0].text)
	}
}

func TestIngestFile_MarkdownFile(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()

	// Create a mock markdown file
	content := "# Test Markdown\n\nThis is a test markdown file."
	file := &mockFile{content: content}
	header := &multipart.FileHeader{
		Filename: "test.md",
		Size:     int64(len(content)),
	}

	err := ingester.IngestFile(ctx, 1, file, header, []string{"test"})

	if err != nil {
		t.Fatalf("IngestFile failed: %v", err)
	}

	if len(store.chunks) == 0 {
		t.Fatal("Expected chunks to be saved")
	}

	if store.chunks[0].source != "test.md" {
		t.Errorf("Expected source 'test.md', got '%s'", store.chunks[0].source)
	}
}

func TestIngestFile_PDFFile(t *testing.T) {
	store := &mockStore{}
	provider := &mockProvider{}
	chunker := &mockChunker{chunkSize: 100}

	ingester := NewIngester(provider, store, chunker, false, false, newTestLogger())

	ctx := context.Background()

	// Create a mock PDF file
	file := &mockFile{content: "PDF content"}
	header := &multipart.FileHeader{
		Filename: "test.pdf",
		Size:     100,
	}

	err := ingester.IngestFile(ctx, 1, file, header, []string{"test"})

	// Should fail because PDF parsing is not implemented yet
	if err == nil {
		t.Fatal("Expected PDF parsing error, got nil")
	}

	if !strings.Contains(err.Error(), "PDF parsing not yet implemented") {
		t.Errorf("Expected PDF parsing error, got: %v", err)
	}
}

// mockFile implements multipart.File for testing
type mockFile struct {
	content string
	pos     int
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	n = copy(p, m.content[m.pos:])
	m.pos += n
	return n, nil
}

func (m *mockFile) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.content)) {
		return 0, io.EOF
	}
	n = copy(p, m.content[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

func (m *mockFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.pos = int(offset)
	case io.SeekCurrent:
		m.pos += int(offset)
	case io.SeekEnd:
		m.pos = len(m.content) + int(offset)
	}
	return int64(m.pos), nil
}

func (m *mockFile) Close() error {
	return nil
}
