package repl

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"papyrus/pkg/conversation"
	"papyrus/pkg/llm"
)

// TestNewREPL creates a new REPL with correct values
func TestNewREPL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")
	repl := New(client, conv)

	if repl.client != client {
		t.Error("New() client not set correctly")
	}
	if repl.conv != conv {
		t.Error("New() conversation not set correctly")
	}
	if repl.reader == nil {
		t.Error("New() reader should be set")
	}
	if repl.writer == nil {
		t.Error("New() writer should be set")
	}
}

// TestHandleCommandQuit handles 'quit' command
func TestHandleCommandQuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	shouldExit := repl.handleCommand("quit")
	if !shouldExit {
		t.Error("handleCommand('quit') should return true")
	}
	if !strings.Contains(output.String(), "Goodbye") {
		t.Error("Output should contain 'Goodbye'")
	}
}

// TestHandleCommandExit handles 'exit' command
func TestHandleCommandExit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	shouldExit := repl.handleCommand("EXIT")
	if !shouldExit {
		t.Error("handleCommand('EXIT') should return true (case-insensitive)")
	}
}

// TestHandleCommandHistory displays conversation history
func TestHandleCommandHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "test response",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")
	conv.AddMessage("user", "test question")
	conv.AddMessage("assistant", "test response")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	shouldExit := repl.handleCommand("history")
	if shouldExit {
		t.Error("handleCommand('history') should return false")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Conversation History") {
		t.Error("Output should contain 'Conversation History'")
	}
	if !strings.Contains(outputStr, "test question") {
		t.Error("Output should contain the test question")
	}
	if !strings.Contains(outputStr, "test response") {
		t.Error("Output should contain the test response")
	}
}

// TestHandleCommandHistoryEmpty handles history when no messages exist
func TestHandleCommandHistoryEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "test",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	repl.handleCommand("history")

	outputStr := output.String()
	if !strings.Contains(outputStr, "No messages yet") {
		t.Error("Output should indicate no messages when history is empty")
	}
}

// TestSendMessage sends a message and appends to conversation
func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    "assistant",
				Content: "response content",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "test-model", 4096)
	conv := conversation.New("test.pdf", "test content")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	shouldExit := repl.sendMessage("test user message")

	if shouldExit {
		t.Error("sendMessage should return false")
	}

	// Check that both messages were added to conversation
	history := conv.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 messages in history, got %d", len(history))
	}
	if history[0].Role != "user" || history[0].Content != "test user message" {
		t.Error("First message should be user message")
	}
	if history[1].Role != "assistant" || history[1].Content != "response content" {
		t.Error("Second message should be assistant response")
	}

	if !strings.Contains(output.String(), "response content") {
		t.Error("Output should contain the response")
	}
}

// TestSendMessageError handles API errors gracefully
func TestSendMessageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := llm.ChatResponse{
			Error: "model not found",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := llm.NewClient(server.URL, "nonexistent-model", 4096)
	conv := conversation.New("test.pdf", "test content")

	var output bytes.Buffer
	repl := New(client, conv)
	repl.writer = &output

	shouldExit := repl.sendMessage("test message")

	if shouldExit {
		t.Error("sendMessage should return false even on error")
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Error") {
		t.Error("Output should contain error message")
	}

	// Verify message was NOT added to conversation on error
	history := conv.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected 0 messages after error, got %d", len(history))
	}
}
