package config

// Default configuration values.
const (
	DefaultOllamaURL = "http://host.docker.internal:11434"
	DefaultModel     = "qwen3:8b"
	MaxTokens        = 4096

	// RAG Defaults
	DefaultVectorDBURL  = "http://localhost:8000"
	DefaultEmbedModel   = "nomic-embed-text"
	DefaultTopK         = 5
	DefaultChunkSize    = 512
	DefaultChunkOverlap = 50
)
