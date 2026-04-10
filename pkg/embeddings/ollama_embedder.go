package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ollamaEmbedder struct {
	url   string
	model string
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

// NewOllamaEmbedder creates a new embedder that uses Ollama's /api/embed endpoint.
// This endpoint supports batching.
func NewOllamaEmbedder(url, model string) Embedder {
	return &ollamaEmbedder{
		url:   strings.TrimRight(url, "/"),
		model: model,
	}
}

func (o *ollamaEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	batch, err := o.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(batch) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return batch[0], nil
}

func (o *ollamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	reqBody := ollamaEmbedRequest{
		Model: o.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := o.url + "/api/embed"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embedResp.Embeddings, nil
}
