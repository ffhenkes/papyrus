package vectordb

import "context"

// Document represents a retrieved chunk with its metadata.
type Document struct {
	ID       string            `json:"id"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
	Score    float64           `json:"score"`
}

// Retriever defines the interface for storing and retrieving document chunks.
type Retriever interface {
	// Query searches for the most relevant documents for a given query string.
	Query(ctx context.Context, query string, topK int) ([]Document, error)

	// Ingest stores text content by splitting it into chunks and embedding them.
	// docID is used to identify the document (e.g., a file hash).
	Ingest(ctx context.Context, docID string, text string) error

	// CollectionExists checks if a collection for the given docID already exists and has data.
	CollectionExists(ctx context.Context, docID string) (bool, error)

	// Close performs any necessary cleanup.
	Close() error
}
