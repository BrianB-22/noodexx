package rag

import (
	"context"
	"math"
	"noodexx/internal/logging"
)

// Store interface for RAG operations
type Store interface {
	Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error)
}

// Chunk represents a search result
type Chunk struct {
	Source string
	Text   string
	Score  float64
}

// Searcher performs vector similarity search
type Searcher struct {
	store  Store // Interface to database
	logger *logging.Logger
}

// NewSearcher creates a new Searcher with the given store
func NewSearcher(store Store, logger *logging.Logger) *Searcher {
	return &Searcher{
		store:  store,
		logger: logger,
	}
}

// Search finds relevant chunks using cosine similarity
func (s *Searcher) Search(ctx context.Context, queryVec []float32, topK int) ([]Chunk, error) {
	logger := s.logger.WithFields(map[string]interface{}{
		"vector_size": len(queryVec),
		"limit":       topK,
	})
	logger.Debug("starting RAG search")

	results, err := s.store.Search(ctx, queryVec, topK)
	if err != nil {
		logger.WithContext("error", err.Error()).Error("search failed")
		return nil, err
	}

	minScore := 0.0
	if len(results) > 0 {
		minScore = results[len(results)-1].Score
	}

	logger.WithFields(map[string]interface{}{
		"result_count": len(results),
		"min_score":    minScore,
	}).Debug("search completed")
	return results, nil
}

// CosineSimilarity computes similarity between two vectors
// Returns a value between -1.0 and 1.0, where 1.0 means identical vectors
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
