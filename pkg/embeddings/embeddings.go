package embeddings

import "context"

// Embedder generates vector embeddings for text strings.
type Embedder interface {
	// Embed generates an embedding vector for a single string.
	Embed(ctx context.Context, text string) ([]float64, error)
	// EmbedBatch generates embedding vectors for multiple strings.
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
}
