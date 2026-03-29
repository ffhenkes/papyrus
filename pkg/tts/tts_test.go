package tts

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSynthesize(t *testing.T) {
	// Create a mock Piper server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := r.URL.Query().Get("text")
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Return dummy WAV data
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = fmt.Fprint(w, "RIFF....WAVEfmt ....data....")
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	// Create temp directory for output
	tmpDir, err := os.MkdirTemp("", "papyrus-tts-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	outputPath := filepath.Join(tmpDir, "test.wav")

	// Test synthesis
	err = client.Synthesize("Hello World", outputPath)
	if err != nil {
		t.Errorf("Synthesize failed: %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Errorf("Output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("Output file is empty")
	}

	// Test empty text error
	err = client.Synthesize("", outputPath)
	if err == nil {
		t.Errorf("Expected error for empty text, got nil")
	}
}
func TestCleanMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Headers and Rules",
			input:    "### Title\n---\nContent",
			expected: "Title\n\nContent",
		},
		{
			name:     "Bold and Italic",
			input:    "This is **bold** and *italic* and __underscore__.",
			expected: "This is bold and italic and underscore.",
		},
		{
			name:     "Links",
			input:    "Check this [Google](https://google.com) link.",
			expected: "Check this Google link.",
		},
		{
			name:     "Lists",
			input:    "- Item 1\n* Item 2\n  + Item 3",
			expected: "Item 1\nItem 2\nItem 3",
		},
		{
			name:     "Code",
			input:    "Use `kubectl` to deploy.",
			expected: "Use kubectl to deploy.",
		},
		{
			name:     "Mixed Markdown",
			input:    "### Summary\n\n- **Safety**: important\n- [Details](url)",
			expected: "Summary\n\nSafety: important\nDetails",
		},
		{
			name:     "Reasoning Blocks",
			input:    "<think>\nThis is internal thinking\nmulti-line\n</think>Actual response",
			expected: "Actual response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("CleanMarkdown() = %q, want %q", got, tt.expected)
			}
		})
	}
}
