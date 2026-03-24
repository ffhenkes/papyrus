package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewClient creates a new client with correct values
func TestNewClient(t *testing.T) {
	url := "http://localhost:11434"
	model := "test-model"
	maxTokens := 2048

	client := NewClient(url, model, maxTokens)

	if client.URL != url {
		t.Errorf("NewClient() URL = %q, want %q", client.URL, url)
	}
	if client.ModelName != model {
		t.Errorf("NewClient() ModelName = %q, want %q", client.ModelName, model)
	}
	if client.MaxTokens != maxTokens {
		t.Errorf("NewClient() MaxTokens = %d, want %d", client.MaxTokens, maxTokens)
	}
}

// TestSendMessage sends a message successfully
func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify system message is present
		if len(req.Messages) < 1 || req.Messages[0].Role != "system" {
			t.Errorf("Expected system message first, got %d messages", len(req.Messages))
		}

		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Test response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)

	response, err := client.SendMessage([]ChatMessage{}, "Test question")

	if err != nil {
		t.Fatalf("SendMessage() returned error: %v", err)
	}

	if response != "Test response" {
		t.Errorf("SendMessage() = %q, want 'Test response'", response)
	}
}

// TestSendMessageWithHistory preserves conversation history
func TestSendMessageWithHistory(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		callCount++

		// First call: expect system + new user message
		// Second call: expect system + previous user + previous assistant + new user message
		if callCount == 1 && len(req.Messages) < 2 {
			t.Errorf("First call: Expected at least 2 messages, got %d", len(req.Messages))
		}
		if callCount == 2 && len(req.Messages) < 4 {
			t.Errorf("Second call: Expected at least 4 messages (history), got %d", len(req.Messages))
		}

		var response string
		if callCount == 1 {
			response = "First response"
		} else {
			response = "Second response"
		}

		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: response,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)

	// First message
	response1, err := client.SendMessage([]ChatMessage{}, "First question")
	if err != nil {
		t.Fatalf("First SendMessage() failed: %v", err)
	}

	if response1 != "First response" {
		t.Errorf("First response = %q, want 'First response'", response1)
	}

	// Second message with history
	history := []ChatMessage{
		{Role: "user", Content: "First question"},
		{Role: "assistant", Content: "First response"},
	}
	response2, err := client.SendMessage(history, "Follow-up question")
	if err != nil {
		t.Fatalf("Second SendMessage() failed: %v", err)
	}

	if response2 != "Second response" {
		t.Errorf("Second response = %q, want 'Second response'", response2)
	}
}

// TestSendMessageAPIError handles API errors
func TestSendMessageAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Error: "model not found",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "nonexistent-model", 4096)
	_, err := client.SendMessage([]ChatMessage{}, "test")
	if err == nil {
		t.Error("SendMessage() should return error when API returns error")
	}
}

// TestSendMessageConnectionError handles connection errors
func TestSendMessageConnectionError(t *testing.T) {
	client := NewClient("http://127.0.0.1:1/invalid", "test-model", 4096)
	_, err := client.SendMessage([]ChatMessage{}, "test")
	if err == nil {
		t.Error("SendMessage() should return error when connection fails")
	}
}

// TestSendMessageInvalidJSON handles invalid JSON response
func TestSendMessageInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)
	_, err := client.SendMessage([]ChatMessage{}, "test")
	if err == nil {
		t.Error("SendMessage() should return error for invalid JSON response")
	}
}

// BenchmarkSendMessage benchmarks the SendMessage method
func BenchmarkSendMessage(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "test response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = client.SendMessage([]ChatMessage{}, "test")
	}
}

// TestSendMessageWithDoc sends a message with document context
func TestSendMessageWithDoc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify system message contains document
		if len(req.Messages) < 1 || req.Messages[0].Role != "system" {
			t.Errorf("Expected system message first, got %d messages", len(req.Messages))
		}
		if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
			if len(req.Messages[0].Content) == 0 {
				t.Error("System message should contain content")
			}
		}

		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Document analyzed",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)
	client.DocumentText = "This is a test document with sample content."

	response, err := client.SendMessageWithDoc([]ChatMessage{}, "Analyze this document", "This is a test document with sample content.")

	if err != nil {
		t.Fatalf("SendMessageWithDoc() returned error: %v", err)
	}

	if response != "Document analyzed" {
		t.Errorf("SendMessageWithDoc() = %q, want 'Document analyzed'", response)
	}
}

// TestSendMessageWithDocHistory verifies document context isn't duplicated in history
func TestSendMessageWithDocHistory(t *testing.T) {
	callCount := 0
	var firstRequestSize int
	var secondRequestSize int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		callCount++
		reqBody, _ := json.Marshal(req)
		switch callCount {
		case 1:
			firstRequestSize = len(reqBody)
		case 2:
			secondRequestSize = len(reqBody)
		}

		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", 4096)
	documentText := "This is a test document with sample content that should be in system message."
	client.DocumentText = documentText

	// First message with document
	_, err := client.SendMessageWithDoc([]ChatMessage{}, "First question", documentText)
	if err != nil {
		t.Fatalf("First SendMessageWithDoc() failed: %v", err)
	}

	// Second message with history (document should not be duplicated in user message history)
	history := []ChatMessage{
		{Role: "user", Content: "First question"},
		{Role: "assistant", Content: "Response"},
	}
	_, err = client.SendMessageWithDoc(history, "Follow-up question", documentText)
	if err != nil {
		t.Fatalf("Second SendMessageWithDoc() failed: %v", err)
	}

	// The second request should be smaller than 2x the first
	// because document isn't in the user message history
	if secondRequestSize >= firstRequestSize*2 {
		t.Errorf("Second request size (%d) should be significantly smaller than 2x first request size (%d), suggests document duplication",
			secondRequestSize, firstRequestSize*2)
	}
}
