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
