package ingest

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"noodexx/internal/logging"
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
	SaveChunk(ctx context.Context, userID int64, source, text string, embedding []float32, tags []string, summary string) error
	DeleteChunksBySource(ctx context.Context, userID int64, source string) error
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
	logger      *logging.Logger
}

// NewIngester creates a new Ingester with all dependencies
func NewIngester(provider LLMProvider, store Store, chunker Chunker, privacyMode, summarize bool, logger *logging.Logger) *Ingester {
	return &Ingester{
		provider:    provider,
		store:       store,
		chunker:     chunker,
		piiDetector: NewPIIDetector(),
		guardrails:  NewGuardrails(),
		privacyMode: privacyMode,
		summarize:   summarize,
		logger:      logger,
	}
}

// IngestText processes plain text with chunking, embedding, and storage
func (ing *Ingester) IngestText(ctx context.Context, userID int64, source, text string, tags []string) error {
	logger := ing.logger.WithFields(map[string]interface{}{
		"source":     source,
		"text_size":  len(text),
		"tags_count": len(tags),
	})
	logger.Debug("starting text ingestion")

	// Delete existing chunks for this source (replace behavior)
	if err := ing.store.DeleteChunksBySource(ctx, userID, source); err != nil {
		logger.WithContext("error", err.Error()).Warn("failed to delete existing chunks")
		// Don't fail ingestion if delete fails - continue with ingestion
	}

	// Check guardrails
	if err := ing.guardrails.Check(source, text); err != nil {
		logger.WithContext("error", err.Error()).Error("guardrails check failed")
		return fmt.Errorf("guardrails check failed: %w", err)
	}

	// Detect PII
	if piiTypes := ing.piiDetector.Detect(text); len(piiTypes) > 0 {
		logger.WithContext("pii_types", piiTypes).Error("PII detected")
		return fmt.Errorf("PII detected: %v - ingestion blocked", piiTypes)
	}

	// Generate summary if enabled
	var summary string
	if ing.summarize {
		var err error
		summary, err = ing.generateSummary(ctx, text)
		if err != nil {
			// Log error but don't fail ingestion - fall back to no summary
			logger.WithContext("error", err.Error()).Warn("summary generation failed")
			summary = ""
		}
	}

	// Chunk text
	chunks := ing.chunker.ChunkText(text)
	logger.WithContext("total_chunks", len(chunks)).Debug("text chunked")

	// Embed and save each chunk
	for i, chunk := range chunks {
		embedding, err := ing.provider.Embed(ctx, chunk)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"chunk_index": i,
				"error":       err.Error(),
			}).Error("embedding failed")
			return fmt.Errorf("embedding failed: %w", err)
		}

		if err := ing.store.SaveChunk(ctx, userID, source, chunk, embedding, tags, summary); err != nil {
			logger.WithFields(map[string]interface{}{
				"chunk_index": i,
				"error":       err.Error(),
			}).Error("save chunk failed")
			return fmt.Errorf("save chunk failed: %w", err)
		}
		logger.WithFields(map[string]interface{}{
			"chunk_index":  i,
			"total_chunks": len(chunks),
		}).Debug("chunk processed")
	}

	logger.WithContext("total_chunks", len(chunks)).Debug("text ingestion completed")
	return nil
}

// IngestURL fetches and processes a web page
func (ing *Ingester) IngestURL(ctx context.Context, userID int64, urlStr string, tags []string) error {
	logger := ing.logger.WithContext("url", urlStr)
	logger.Debug("starting URL ingestion")

	if ing.privacyMode {
		logger.Error("URL ingestion disabled in privacy mode")
		return fmt.Errorf("URL ingestion is disabled in privacy mode")
	}

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("invalid URL")
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Fetch URL content
	resp, err := http.Get(urlStr)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to fetch URL")
		return fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML using go-readability
	article, err := readability.FromReader(resp.Body, parsedURL)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to parse HTML")
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	logger.WithContext("text_size", len(article.TextContent)).Debug("URL content fetched and parsed")
	return ing.IngestText(ctx, userID, urlStr, article.TextContent, tags)
}

// IngestFile processes an uploaded file based on MIME type
func (ing *Ingester) IngestFile(ctx context.Context, userID int64, file multipart.File, header *multipart.FileHeader, tags []string) error {
	logger := ing.logger.WithFields(map[string]interface{}{
		"file_path": header.Filename,
		"file_size": header.Size,
	})
	logger.Debug("starting file ingestion")

	// Check file size
	if header.Size > ing.guardrails.MaxFileSize {
		logger.WithContext("limit", ing.guardrails.MaxFileSize).Error("file size exceeds limit")
		return fmt.Errorf("file size %d exceeds limit %d", header.Size, ing.guardrails.MaxFileSize)
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !ing.guardrails.IsAllowedExtension(ext) {
		logger.WithContext("extension", ext).Error("file extension not allowed")
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
		logger.WithContext("extension", ext).Error("unsupported file type")
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		logger.WithContext("error", err.Error()).Error("failed to parse file")
		return fmt.Errorf("failed to parse file: %w", err)
	}

	logger.WithContext("text_size", len(text)).Debug("file parsed successfully")
	return ing.IngestText(ctx, userID, header.Filename, text, tags)
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
