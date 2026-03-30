package tts

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPiperSynthesize(t *testing.T) {
	// Create a mock Piper server that returns valid WAV data
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		text := r.URL.Query().Get("text")
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Return a minimal valid WAV file (44.1kHz, mono, 16-bit)
		// RIFF header + fmt chunk + data chunk
		w.Header().Set("Content-Type", "audio/wav")

		// Minimal WAV structure: RIFF header (12) + fmt chunk (24) + data chunk (8) + 100 bytes of zeros
		wav := []byte{
			// RIFF header
			'R', 'I', 'F', 'F', // "RIFF"
			136, 0, 0, 0, // File size - 8 (136 = 44 + 100 - 8)
			'W', 'A', 'V', 'E', // "WAVE"

			// fmt subchunk
			'f', 'm', 't', ' ', // "fmt "
			16, 0, 0, 0, // Subchunk1Size (16 for PCM)
			1, 0, // AudioFormat (1 = PCM)
			1, 0, // NumChannels (1 = mono)
			68, 172, 0, 0, // SampleRate (44100)
			16, 177, 2, 0, // ByteRate
			2, 0, // BlockAlign
			16, 0, // BitsPerSample (16)

			// data subchunk
			'd', 'a', 't', 'a', // "data"
			100, 0, 0, 0, // Subchunk2Size (100 bytes)
		}
		// Add 100 bytes of PCM data (silence)
		for i := 0; i < 100; i++ {
			wav = append(wav, 0)
		}

		_, _ = w.Write(wav)
	}))
	defer ts.Close()

	client := NewPiperClient(ts.URL)
	client.DefaultVoice = "pt_BR-faber-medium" // Set default voice

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
