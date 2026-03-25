package conversation

import (
	"strings"
	"testing"
)

func TestExportMarkdown(t *testing.T) {
	conv := New("sample_document.pdf", "Some sample text.")
	conv.SessionID = "TEST-SESSION-123"
	conv.AddMessage("user", "What is this document about?")
	conv.AddMessage("assistant", "This document is a test.")

	md := ExportMarkdown(conv)

	// Validate title and metadata
	if !strings.Contains(md, "# Papyrus Session Export: TEST-SESSION-123") {
		t.Errorf("Markdown export missing title/session ID: \n%s", md)
	}
	if !strings.Contains(md, "**Document**: `sample_document.pdf`") {
		t.Errorf("Markdown export missing filename metadata")
	}

	// Validate avatars/roles
	if !strings.Contains(md, "### \U0001F464 User") {
		t.Errorf("Markdown missing user avater")
	}
	if !strings.Contains(md, "### \U0001F916 Assistant") {
		t.Errorf("Markdown missing assistant avater")
	}

	// Validate message content
	if !strings.Contains(md, "What is this document about?") {
		t.Errorf("Markdown missing user message content")
	}
	if !strings.Contains(md, "This document is a test.") {
		t.Errorf("Markdown missing assistant message content")
	}
}
