package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// --- Test: getEnv with fallback ---
func TestGetEnvWithoutVariable(t *testing.T) {
	// Ensure variable doesn't exist
	os.Unsetenv("TEST_VAR_NONEXISTENT")

	result := getEnv("TEST_VAR_NONEXISTENT", "fallback_value")
	if result != "fallback_value" {
		t.Errorf("getEnv() = %q, want %q", result, "fallback_value")
	}
}

// --- Test: getEnv with set variable ---
func TestGetEnvWithVariable(t *testing.T) {
	os.Setenv("TEST_VAR_EXISTS", "actual_value")
	defer os.Unsetenv("TEST_VAR_EXISTS")

	result := getEnv("TEST_VAR_EXISTS", "fallback_value")
	if result != "actual_value" {
		t.Errorf("getEnv() = %q, want %q", result, "actual_value")
	}
}

// --- Test: getEnv with empty string ---
func TestGetEnvEmptyString(t *testing.T) {
	os.Setenv("TEST_EMPTY", "")
	defer os.Unsetenv("TEST_EMPTY")

	result := getEnv("TEST_EMPTY", "fallback_value")
	if result != "fallback_value" {
		t.Errorf("getEnv() with empty env = %q, want %q", result, "fallback_value")
	}
}

// --- Test: printUsage doesn't panic ---
func TestPrintUsage(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Should not panic
	printUsage()

	w.Close()
	os.Stderr = old

	// Read output
	output, _ := io.ReadAll(r)
	if len(output) == 0 {
		t.Error("printUsage() produced no output")
	}

	if !bytes.Contains(output, []byte("Usage:")) {
		t.Errorf("printUsage() output doesn't contain 'Usage:'")
	}
}

// --- Test: PDF extraction with missing file ---
func TestExtractPDFTextFileNotFound(t *testing.T) {
	_, err := extractPDFText("nonexistent_file.pdf")
	if err == nil {
		t.Error("extractPDFText() should return error for nonexistent file")
	}
}

// --- Test: explainText with successful API response ---
func TestExplainTextSuccess(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected POST to /api/chat, got %s", r.URL.Path)
		}

		// Verify request format
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify model is set
		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got %q", req.Model)
		}

		// Verify messages array has system and user messages
		if len(req.Messages) < 2 {
			t.Errorf("Expected at least 2 messages, got %d", len(req.Messages))
		}

		// Send mock response
		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "This is a test document analysis.",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	testText := "Sample PDF text content for testing."
	result, err := explainText(server.URL, "test-model", testText, "")

	if err != nil {
		t.Fatalf("explainText() returned error: %v", err)
	}

	if result != "This is a test document analysis." {
		t.Errorf("explainText() = %q, want 'This is a test document analysis.'", result)
	}
}

// --- Test: explainText with custom prompt ---
func TestExplainTextCustomPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Verify custom prompt is in user message
		userMessage := req.Messages[1]
		if !bytes.Contains([]byte(userMessage.Content), []byte("Custom analysis")) {
			t.Errorf("Custom prompt not found in message: %s", userMessage.Content)
		}

		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Custom response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	result, err := explainText(server.URL, "test-model", "test text", "Custom analysis prompt")
	if err != nil {
		t.Fatalf("explainText() returned error: %v", err)
	}

	if result != "Custom response" {
		t.Errorf("explainText() = %q, want 'Custom response'", result)
	}
}

// --- Test: explainText with API error response ---
func TestExplainTextAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Error: "model not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	_, err := explainText(server.URL, "nonexistent-model", "test text", "")
	if err == nil {
		t.Error("explainText() should return error when API returns error")
	}
}

// --- Test: explainText with connection error ---
func TestExplainTextConnectionError(t *testing.T) {
	_, err := explainText("http://127.0.0.1:1/invalid", "test-model", "test text", "")
	if err == nil {
		t.Error("explainText() should return error when connection fails")
	}
}

// --- Test: explainText with invalid JSON response ---
func TestExplainTextInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	_, err := explainText(server.URL, "test-model", "test text", "")
	if err == nil {
		t.Error("explainText() should return error for invalid JSON response")
	}
}

// --- Benchmark: getEnv performance ---
func BenchmarkGetEnv(b *testing.B) {
	os.Setenv("BENCH_VAR", "test_value")
	defer os.Unsetenv("BENCH_VAR")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getEnv("BENCH_VAR", "fallback")
	}
}

// --- Benchmark: explainText with mock server ---
func BenchmarkExplainText(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatResponse{
			Message: ChatMessage{
				Role:    "assistant",
				Content: "test response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	testText := "Sample text for benchmarking"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		explainText(server.URL, "test-model", testText, "")
	}
}

// --- Test: NewConversation creates conversation with correct initial state ---
func TestNewConversation(t *testing.T) {
	fileName := "test.pdf"
	docText := "Sample document text"

	conv := NewConversation(fileName, docText)

	if conv.FileName != fileName {
		t.Errorf("NewConversation() FileName = %q, want %q", conv.FileName, fileName)
	}

	if conv.DocumentText != docText {
		t.Errorf("NewConversation() DocumentText = %q, want %q", conv.DocumentText, docText)
	}

	if len(conv.Messages) != 0 {
		t.Errorf("NewConversation() Messages length = %d, want 0", len(conv.Messages))
	}

	if conv.CreatedAt.IsZero() {
		t.Error("NewConversation() CreatedAt should not be zero")
	}

	if conv.SessionID != "" {
		t.Errorf("NewConversation() SessionID = %q, want empty", conv.SessionID)
	}
}

// --- Test: AddMessage appends message to conversation ---
func TestAddMessage(t *testing.T) {
	conv := NewConversation("test.pdf", "test content")

	AddMessage(conv, "user", "First question")
	if len(conv.Messages) != 1 {
		t.Errorf("After first AddMessage, length = %d, want 1", len(conv.Messages))
	}

	if conv.Messages[0].Role != "user" {
		t.Errorf("First message role = %q, want 'user'", conv.Messages[0].Role)
	}

	if conv.Messages[0].Content != "First question" {
		t.Errorf("First message content = %q, want 'First question'", conv.Messages[0].Content)
	}

	AddMessage(conv, "assistant", "First answer")
	if len(conv.Messages) != 2 {
		t.Errorf("After second AddMessage, length = %d, want 2", len(conv.Messages))
	}

	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Second message role = %q, want 'assistant'", conv.Messages[1].Role)
	}
}

// --- Test: GetHistory returns copy of messages ---
func TestGetHistory(t *testing.T) {
	conv := NewConversation("test.pdf", "test content")

	AddMessage(conv, "user", "Question 1")
	AddMessage(conv, "assistant", "Answer 1")

	history := GetHistory(conv)

	if len(history) != 2 {
		t.Errorf("GetHistory() length = %d, want 2", len(history))
	}

	if history[0].Content != "Question 1" {
		t.Errorf("GetHistory()[0].Content = %q, want 'Question 1'", history[0].Content)
	}

	if history[1].Content != "Answer 1" {
		t.Errorf("GetHistory()[1].Content = %q, want 'Answer 1'", history[1].Content)
	}

	// Verify it's a copy (modifying history shouldn't affect conversation)
	history[0].Content = "Modified"
	if conv.Messages[0].Content == "Modified" {
		t.Error("GetHistory() should return a copy, not a reference")
	}
}

// --- Test: sendMessage adds user and assistant messages to conversation ---
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
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	conv := NewConversation("test.pdf", "test document content")

	response, err := sendMessage(server.URL, "test-model", conv, "Test question")

	if err != nil {
		t.Fatalf("sendMessage() returned error: %v", err)
	}

	if response != "Test response" {
		t.Errorf("sendMessage() = %q, want 'Test response'", response)
	}

	// Verify message was added to conversation
	if len(conv.Messages) != 2 {
		t.Errorf("After sendMessage, Messages length = %d, want 2", len(conv.Messages))
	}

	if conv.Messages[0].Role != "user" || conv.Messages[0].Content != "Test question" {
		t.Error("User message not properly added to conversation")
	}

	if conv.Messages[1].Role != "assistant" || conv.Messages[1].Content != "Test response" {
		t.Error("Assistant message not properly added to conversation")
	}
}

// --- Test: sendMessage preserves conversation history ---
func TestSendMessagePreservesHistory(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

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
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	conv := NewConversation("test.pdf", "test content")

	// First message
	response1, err := sendMessage(server.URL, "test-model", conv, "First question")
	if err != nil {
		t.Fatalf("First sendMessage() failed: %v", err)
	}

	if response1 != "First response" {
		t.Errorf("First response = %q, want 'First response'", response1)
	}

	if len(conv.Messages) != 2 {
		t.Errorf("After first sendMessage, Messages length = %d, want 2", len(conv.Messages))
	}

	// Second message (should include first in history)
	response2, err := sendMessage(server.URL, "test-model", conv, "Follow-up question")
	if err != nil {
		t.Fatalf("Second sendMessage() failed: %v", err)
	}

	if response2 != "Second response" {
		t.Errorf("Second response = %q, want 'Second response'", response2)
	}

	if len(conv.Messages) != 4 {
		t.Errorf("After second sendMessage, Messages length = %d, want 4", len(conv.Messages))
	}

	// Verify order
	if conv.Messages[0].Content != "First question" {
		t.Error("First message not preserved in history")
	}

	if conv.Messages[1].Content != "First response" {
		t.Error("First assistant response not preserved")
	}

	if conv.Messages[2].Content != "Follow-up question" {
		t.Error("Second user message not added")
	}

	if conv.Messages[3].Content != "Second response" {
		t.Error("Second assistant response not added")
	}
}
