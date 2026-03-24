package conversation

import (
	"strings"
	"testing"
)

// TestNewConversation creates conversation with correct initial state
func TestNew(t *testing.T) {
	fileName := "test.pdf"
	docText := "Sample document text"

	conv := New(fileName, docText)

	if conv.FileName != fileName {
		t.Errorf("New() FileName = %q, want %q", conv.FileName, fileName)
	}

	if conv.DocumentText != docText {
		t.Errorf("New() DocumentText = %q, want %q", conv.DocumentText, docText)
	}

	if len(conv.Messages) != 0 {
		t.Errorf("New() Messages length = %d, want 0", len(conv.Messages))
	}

	if conv.CreatedAt.IsZero() {
		t.Error("New() CreatedAt should not be zero")
	}

	if conv.SessionID == "" {
		t.Error("New() SessionID should not be empty")
	}
}

// TestAddMessage appends message to conversation
func TestAddMessage(t *testing.T) {
	conv := New("test.pdf", "test content")

	conv.AddMessage("user", "First question")
	if len(conv.Messages) != 1 {
		t.Errorf("After first AddMessage, length = %d, want 1", len(conv.Messages))
	}

	if conv.Messages[0].Role != "user" {
		t.Errorf("First message role = %q, want 'user'", conv.Messages[0].Role)
	}

	if conv.Messages[0].Content != "First question" {
		t.Errorf("First message content = %q, want 'First question'", conv.Messages[0].Content)
	}

	conv.AddMessage("assistant", "First answer")
	if len(conv.Messages) != 2 {
		t.Errorf("After second AddMessage, length = %d, want 2", len(conv.Messages))
	}

	if conv.Messages[1].Role != "assistant" {
		t.Errorf("Second message role = %q, want 'assistant'", conv.Messages[1].Role)
	}
}

// TestGetHistory returns copy of messages
func TestGetHistory(t *testing.T) {
	conv := New("test.pdf", "test content")

	conv.AddMessage("user", "Question 1")
	conv.AddMessage("assistant", "Answer 1")

	history := conv.GetHistory()

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

// TestGenerateSessionIDWithPDF handles standard PDF files.
func TestGenerateSessionIDWithPDF(t *testing.T) {
	conv := New("document.pdf", "")
	if !strings.HasPrefix(conv.SessionID, "document-") {
		t.Errorf("SessionID %q should start with 'document-'", conv.SessionID)
	}
}

// TestGenerateSessionIDNonPDF handles non-PDF extensions without panic.
func TestGenerateSessionIDNonPDF(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		prefix   string
	}{
		{"txt file", "notes.txt", "notes-"},
		{"md file", "README.md", "README-"},
		{"no extension", "myfile", "myfile-"},
		{"multiple dots", "archive.tar.gz", "archive.tar-"},
		{"path with dirs", "/home/user/docs/report.pdf", "report-"},
		{"windows path", "C:\\Users\\docs\\report.pdf", "report-"},
		{"dotfile", ".hidden", "session-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := New(tt.fileName, "")
			if !strings.HasPrefix(conv.SessionID, tt.prefix) {
				t.Errorf("SessionID %q should start with %q", conv.SessionID, tt.prefix)
			}
			// Verify hash suffix is present (13 chars: dash + 12 hex)
			parts := strings.SplitN(conv.SessionID, "-", 2)
			if len(parts) < 2 || len(parts[1]) < 12 {
				t.Errorf("SessionID %q missing valid hash suffix", conv.SessionID)
			}
		})
	}
}
