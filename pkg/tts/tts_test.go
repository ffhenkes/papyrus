package tts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPiperSynthesize(t *testing.T) {
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

	client := NewPiperClient(ts.URL)

	// Test synthesis
	data, err := client.Synthesize(context.Background(), "Hello World", false)
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if !bytes.Contains(data, []byte("WAVE")) {
		t.Errorf("Expected WAVE data, got %q", string(data))
	}

	// Test empty text error
	_, err = client.Synthesize(context.Background(), "", false)
	if err == nil {
		t.Errorf("Expected error for empty text, got nil")
	}
}

func TestOpenTTSSynthesize(t *testing.T) {
	// Create a mock OpenTTS server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType == "application/ssml+xml" {
			if !bytes.Contains(body, []byte("<speak>")) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		w.Header().Set("Content-Type", "audio/wav")
		_, _ = fmt.Fprint(w, "OPENTTS_AUDIO_DATA")
	}))
	defer ts.Close()

	client := NewOpenTTSClient(ts.URL)

	// Test SSML synthesis
	data, err := client.Synthesize(context.Background(), "<speak>Hello</speak>", true)
	if err != nil {
		t.Fatalf("SSML Synthesize failed: %v", err)
	}
	if string(data) != "OPENTTS_AUDIO_DATA" {
		t.Errorf("Got %q, want OPENTTS_AUDIO_DATA", string(data))
	}

	// Test plain text synthesis
	data, err = client.Synthesize(context.Background(), "Hello", false)
	if err != nil {
		t.Fatalf("Plain Synthesize failed: %v", err)
	}
	if string(data) != "OPENTTS_AUDIO_DATA" {
		t.Errorf("Got %q, want OPENTTS_AUDIO_DATA", string(data))
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
