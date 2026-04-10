package vectordb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"papyrus/pkg/chunker"
	"papyrus/pkg/embeddings"
)

type chromaRetriever struct {
	url            string
	embedder       embeddings.Embedder
	collectionName string
	collectionID   string
	chunkSize      int
	overlap        int
}

// NewChromaRetriever creates a new retriever implementation for ChromaDB.
func NewChromaRetriever(url string, embedder embeddings.Embedder, chunkSize, overlap int) Retriever {
	return &chromaRetriever{
		url:       strings.TrimRight(url, "/"),
		embedder:  embedder,
		chunkSize: chunkSize,
		overlap:   overlap,
	}
}

func (c *chromaRetriever) CollectionExists(ctx context.Context, docID string) (bool, error) {
	name := "papyrus-" + docID
	endpoint := fmt.Sprintf("%s/api/v1/collections/%s", c.url, name)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("chroma returned status %d", resp.StatusCode)
	}

	var collection struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return false, err
	}

	c.collectionID = collection.ID
	c.collectionName = collection.Name

	// Check if it has items
	countEndpoint := fmt.Sprintf("%s/api/v1/collections/%s/count", c.url, collection.ID)
	reqCount, _ := http.NewRequestWithContext(ctx, "GET", countEndpoint, nil)
	respCount, err := http.DefaultClient.Do(reqCount)
	if err == nil {
		defer func() { _ = respCount.Body.Close() }()
		var count int
		if json.NewDecoder(respCount.Body).Decode(&count) == nil {
			return count > 0, nil
		}
	}

	return true, nil
}

func (c *chromaRetriever) Ingest(ctx context.Context, docID string, text string) error {
	c.collectionName = "papyrus-" + docID

	// 1. Create collection if not exists
	if err := c.createCollection(ctx); err != nil {
		return err
	}

	// 2. Chunk text
	chunks := chunker.Split(text, c.chunkSize, c.overlap)
	if len(chunks) == 0 {
		return nil
	}

	// 3. Embed chunks in batches
	batchSize := 20
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		var texts []string
		for _, b := range batch {
			texts = append(texts, b.Content)
		}

		vecs, err := c.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return fmt.Errorf("failed to embed batch: %w", err)
		}

		// 4. Add to Chroma
		if err := c.addToCollection(ctx, batch, vecs); err != nil {
			return err
		}
	}

	return nil
}

func (c *chromaRetriever) Query(ctx context.Context, query string, topK int) ([]Document, error) {
	if c.collectionID == "" {
		return nil, fmt.Errorf("no collection active: call Ingest or CollectionExists first")
	}

	vec, err := c.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/collections/%s/query", c.url, c.collectionID)

	queryReq := map[string]interface{}{
		"query_embeddings": [][]float64{vec},
		"n_results":        topK,
	}

	jsonData, _ := json.Marshal(queryReq)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chroma query failed: status %d", resp.StatusCode)
	}

	var results struct {
		IDs       [][]string              `json:"ids"`
		Documents [][]string              `json:"documents"`
		Metadatas [][](map[string]string) `json:"metadatas"`
		Distances [][]float64             `json:"distances"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results.IDs) == 0 || len(results.IDs[0]) == 0 {
		return nil, nil
	}

	var docs []Document
	for i := range results.IDs[0] {
		docs = append(docs, Document{
			ID:       results.IDs[0][i],
			Content:  results.Documents[0][i],
			Metadata: results.Metadatas[0][i],
			Score:    1.0 - results.Distances[0][i], // Convert distance to similarity score
		})
	}

	return docs, nil
}

func (c *chromaRetriever) Close() error {
	return nil
}

func (c *chromaRetriever) createCollection(ctx context.Context) error {
	endpoint := c.url + "/api/v1/collections"
	body := map[string]string{"name": c.collectionName}
	jsonData, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// If already exists, just get it
		endpointGet := fmt.Sprintf("%s/api/v1/collections/%s", c.url, c.collectionName)
		reqGet, _ := http.NewRequestWithContext(ctx, "GET", endpointGet, nil)
		respGet, err := http.DefaultClient.Do(reqGet)
		if err != nil {
			return err
		}
		defer func() { _ = respGet.Body.Close() }()

		var collection struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(respGet.Body).Decode(&collection); err != nil {
			return err
		}
		c.collectionID = collection.ID
		return nil
	}

	var collection struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return err
	}
	c.collectionID = collection.ID
	return nil
}

func (c *chromaRetriever) addToCollection(ctx context.Context, chunks []chunker.Chunk, embeddings [][]float64) error {
	endpoint := fmt.Sprintf("%s/api/v1/collections/%s/add", c.url, c.collectionID)

	ids := make([]string, len(chunks))
	documents := make([]string, len(chunks))
	metadatas := make([]map[string]string, len(chunks))

	for i, chk := range chunks {
		ids[i] = fmt.Sprintf("chunk-%d", chk.Index)
		documents[i] = chk.Content
		metadatas[i] = map[string]string{"index": fmt.Sprintf("%d", chk.Index)}
	}

	body := map[string]interface{}{
		"ids":        ids,
		"embeddings": embeddings,
		"documents":  documents,
		"metadatas":  metadatas,
	}

	jsonData, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add to collection: status %d", resp.StatusCode)
	}

	return nil
}
