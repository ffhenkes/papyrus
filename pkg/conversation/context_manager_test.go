package conversation

import (
	"strings"
	"testing"

	"papyrus/pkg/llm"
)

func TestPruneHistoryUnderLimit(t *testing.T) {
	messages := []llm.ChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
	}

	kept, summary := PruneHistory(messages, 1000)
	if len(kept) != 2 {
		t.Errorf("Expected 2 messages kept, got %d", len(kept))
	}
	if summary != "" {
		t.Errorf("Expected empty summary, got %s", summary)
	}
}

func TestPruneHistoryOverLimit(t *testing.T) {
	messages := []llm.ChatMessage{
		{Role: "user", Content: strings.Repeat("word ", 1000)}, // ~1333 tokens
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Short question"},
		{Role: "assistant", Content: "Short answer"},
	}

	kept, summary := PruneHistory(messages, 100) // Forces prune of the first pair

	if len(kept) != 2 {
		t.Errorf("Expected 2 messages kept, got %d", len(kept))
	}
	if summary == "" {
		t.Error("Expected non-empty summary for pruned messages")
	}
	if !strings.Contains(summary, "Previous context summary") {
		t.Errorf("Unexpected summary format: %s", summary)
	}
}

func TestSummarizeOldMessages(t *testing.T) {
	messages := []llm.ChatMessage{
		{Role: "user", Content: "Short query"},
		{Role: "assistant", Content: strings.Repeat("A", 200)}, // Should be truncated
	}

	summary := SummarizeOldMessages(messages)
	if !strings.Contains(summary, "user: Short query") {
		t.Errorf("Summary missing user query: %s", summary)
	}
	if !strings.Contains(summary, "...") {
		t.Errorf("Summary failed to truncate long message: %s", summary)
	}
}
