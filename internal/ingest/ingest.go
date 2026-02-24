package ingest

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-shiori/go-readability"
)

// LLMProvider interface for embeddings and summarization
type LLMProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Stream(ctx context.Context, messages []Message, w io.Writer) (string, error)
}

// Message represents a chat message for LLM
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Store interface for saving chunks
type Store interface {
	SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error
}

// Chunker interface for text chunking
type Chunker interface {
	ChunkText(text string) []string
}

// Ingester orchestrates document ingestion
type Ingester struct {
	provider    LLMProvider
	store       Store
	chunker     Chunker
	piiDetector *PIIDetector
	guardrails  *Guardrails
	privacyMode bool
	summarize   bool
}

// NewIngester creates a new Ingester with all dependencies
func NewIngester(provider LLMProvider, store Store, chunker Chunker, privacyMode, summarize bool) *Ingester {
	return &Ingester{
		provider:    provider,
		store:       store,
		chunker:     chunker,
		piiDetector: NewPIIDetector(),
		guardrails:  NewGuardrails(),
		privacyMode: privacyMode,
		summarize:   summarize,
	}
}

// IngestText processes plain text with chunking, embedding, and storage
func (ing *Ingester) IngestText(ctx context.Context, source, text string, tags []string) error {
	// Check guardrails
	if err := ing.guardrails.Check(source, text); err != nil {
		return fmt.Errorf("guardrails check failed: %w", err)
	}

	// Detect PII
	if piiTypes := ing.piiDetector.Detect(text); len(piiTypes) > 0 {
		return fmt.Errorf("PII detected: %v - ingestion blocked", piiTypes)
	}

	// Generate summary if enabled
	var summary string
	if ing.summarize {
		var err error
		summary, err = ing.generateSummary(ctx, text)
		if err != nil {
			// Log error but don't fail ingestion - fall back to no summary
			summary = ""
		}
	}

	// Chunk text
	chunks := ing.chunker.ChunkText(text)

	// Embed and save each chunk
	for _, chunk := range chunks {
		embedding, err := ing.provider.Embed(ctx, chunk)
		if err != nil {
			return fmt.Errorf("embedding failed: %w", err)
		}

		if err := ing.store.SaveChunk(ctx, source, chunk, embedding, tags, summary); err != nil {
			return fmt.Errorf("save chunk failed: %w", err)
		}
	}

	return nil
}

// IngestURL fetches and processes a web page
func (ing *Ingester) IngestURL(ctx context.Context, url string, tags []string) error {
	if ing.privacyMode {
		return fmt.Errorf("URL ingestion is disabled in privacy mode")
	}

	// Fetch URL content
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML using go-readability
	article, err := readability.FromReader(resp.Body, nil)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	return ing.IngestText(ctx, url, article.TextContent, tags)
}

// IngestFile processes an uploaded file based on MIME type
func (ing *Ingester) IngestFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, tags []string) error {
	// Check file size
	if header.Size > ing.guardrails.MaxFileSize {
		return fmt.Errorf("file size %d exceeds limit %d", header.Size, ing.guardrails.MaxFileSize)
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !ing.guardrails.IsAllowedExtension(ext) {
		return fmt.Errorf("file extension %s is not allowed", ext)
	}

	// Parse based on file extension
	var text string
	var err error

	switch ext {
	case ".txt", ".md":
		text, err = ing.parseText(file)
	case ".pdf":
		text, err = ing.parsePDF(file)
	case ".html":
		text, err = ing.parseHTML(file)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	return ing.IngestText(ctx, header.Filename, text, tags)
}

// parseText reads plain text from a reader
func (ing *Ingester) parseText(r io.Reader) (string, error) {
	bytes, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// parsePDF parses a PDF file (placeholder implementation)
func (ing *Ingester) parsePDF(r io.Reader) (string, error) {
	// TODO: Implement PDF parsing using a library like pdfcpu or unidoc
	return "", fmt.Errorf("PDF parsing not yet implemented")
}

// parseHTML parses an HTML file using go-readability
func (ing *Ingester) parseHTML(r io.Reader) (string, error) {
	article, err := readability.FromReader(r, nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	return article.TextContent, nil
}

// generateSummary creates a 2-3 sentence summary using the LLM
func (ing *Ingester) generateSummary(ctx context.Context, text string) (string, error) {
	// Take first 1000 characters as input
	input := text
	if len(input) > 1000 {
		input = input[:1000]
	}

	// Build prompt
	messages := []Message{
		{Role: "user", Content: "Summarize this document in 2-3 sentences:\n\n" + input},
	}

	// Stream to a buffer
	var buf strings.Builder
	summary, err := ing.provider.Stream(ctx, messages, &buf)
	if err != nil {
		return "", err
	}

	return summary, nil
}
