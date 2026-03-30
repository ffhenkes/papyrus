package tts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// TTSEngine defines the interface for text-to-speech engines.
type TTSEngine interface {
	Synthesize(ctx context.Context, text string, isSSML bool) ([]byte, error)
}

// PiperClient handles communication with the Piper TTS service.
type PiperClient struct {
	BaseURL      string
	DefaultVoice string // Default voice for SSML synthesis
}

// NewPiperClient creates a new Piper TTS client.
func NewPiperClient(baseURL string) *PiperClient {
	return &PiperClient{BaseURL: baseURL}
}

// Synthesize sends text to Piper and returns the resulting audio bytes.
// If isSSML is true, it parses SSML and synthesizes each segment separately,
// then concatenates them with silence for breaks.
// Otherwise, converts markdown to SSML for enhanced prosody and structure.
func (c *PiperClient) Synthesize(ctx context.Context, text string, isSSML bool) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// First, strip any nested <speak> and </speak> tags that might come from LLM
	text = stripSpeakTags(text)

	// Check if input is already in SSML or HTML-like format
	// If so, skip markdown processing and route directly to SSML synthesis
	hasSSMLTags := strings.Contains(text, "<speak") ||
		strings.Contains(text, "<break") ||
		strings.Contains(text, "<voice") ||
		strings.Contains(text, "<prosody")

	hasHTMLTags := strings.Contains(text, "<b>") ||
		strings.Contains(text, "<ul>") ||
		strings.Contains(text, "<li>") ||
		strings.Contains(text, "<br>")

	if isSSML || hasSSMLTags || hasHTMLTags {
		// Content is already SSML or HTML-formatted
		// Wrap in a single <speak> tag
		cleanText := text
		if !strings.Contains(cleanText, "<speak") {
			cleanText = "<speak>" + cleanText + "</speak>"
		}
		return c.synthesizeSSML(ctx, cleanText)
	}

	// Convert plain markdown to SSML for enhanced prosody and structure
	// Use minimal config for short conversational text (REPL), full config for documents
	var config MarkdownSSMLConfig

	// Heuristic: if text is short and lacks document structure, treat as conversational
	isConversational := len(text) < 300 &&
		!strings.Contains(text, "# ") &&
		!strings.Contains(text, "## ") &&
		!strings.Contains(text, "### ") &&
		!strings.ContainsAny(text, "\n\n") // Multiple paragraphs suggest document

	if isConversational {
		config = ConversationalMinimalConfig()
	} else {
		config = DefaultMarkdownSSMLConfig()
	}

	ssmlText := MarkdownToSSML(text, config)

	// Clean any remaining markdown
	cleanText := CleanMarkdown(ssmlText)

	// Wrap in SSML speak tags if not already wrapped
	if !strings.Contains(cleanText, "<speak") {
		cleanText = "<speak>" + cleanText + "</speak>"
	}

	// Route to SSML synthesis
	return c.synthesizeSSML(ctx, cleanText)
}

// stripSpeakTags removes all nested <speak> and </speak> tags from text
// This handles cases where LLM output has multiple nested speak tags
func stripSpeakTags(text string) string {
	// Use regex to handle variations and nested tags
	speakRegex := regexp.MustCompile(`</?speak\s*>`)
	text = speakRegex.ReplaceAllString(text, "")
	// Clean up any extra whitespace that might have been created
	return strings.TrimSpace(text)
}

// synthesizeSSML synthesizes SSML content by parsing it and synthesizing each segment
func (c *PiperClient) synthesizeSSML(ctx context.Context, ssmlText string) ([]byte, error) {
	// Parse SSML
	elements, err := ParseSSML(ssmlText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSML: %w", err)
	}

	// Extract format info by synthesizing a dummy segment to get sample rate info
	// We'll assume Piper generates 44.1kHz 16-bit mono (standard for Piper)
	sampleRate := uint32(44100)
	numChannels := uint16(1)
	bitsPerSample := uint16(16)

	// Synthesize each element
	var wavSegments [][]byte

	for _, elem := range elements {
		switch e := elem.(type) {
		case *SSMLSegment:
			// Determine which voice to use
			voice := e.Voice
			if voice == "" {
				voice = c.DefaultVoice
			}

			// Clean the text
			cleanText := CleanMarkdown(e.Text)
			if cleanText == "" {
				continue
			}

			// Synthesize the segment
			wav, err := c.synthesizeSegmentWithVoice(ctx, cleanText, voice)
			if err != nil {
				// Log error but continue with next segment
				fmt.Printf("Failed to synthesize segment with voice %q: %v\n", voice, err)
				continue
			}

			wavSegments = append(wavSegments, wav)

		case *SSMLBreak:
			// Generate silence for the break duration
			silence := GenerateSilence(numChannels, sampleRate, e.DurationMs, bitsPerSample)

			// Create a WAV segment with silence
			silenceWAV := CreateWAVFile(&WAVFormat{
				AudioFormat:   1,
				NumChannels:   numChannels,
				SampleRate:    sampleRate,
				ByteRate:      uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample) / 8,
				BlockAlign:    numChannels * bitsPerSample / 8,
				BitsPerSample: bitsPerSample,
			}, silence)

			wavSegments = append(wavSegments, silenceWAV)
		}
	}

	if len(wavSegments) == 0 {
		return nil, fmt.Errorf("no audio segments generated from SSML")
	}

	// Concatenate all segments
	combined, err := ConcatenateWAV(wavSegments)
	if err != nil {
		return nil, fmt.Errorf("failed to concatenate audio segments: %w", err)
	}

	return combined, nil
}

// synthesizeSegmentWithVoice synthesizes a text segment with a specific voice
func (c *PiperClient) synthesizeSegmentWithVoice(ctx context.Context, text string, voice string) ([]byte, error) {
	// Prepare the request URL with optional voice parameter
	params := url.Values{}
	params.Add("text", text)

	// If voice is specified, include it in the URL
	// Note: This depends on how the Piper HTTP server is configured
	// Standard piper-http might use voice as a query parameter or might not support it
	if voice != "" {
		params.Add("voice", voice)
	}

	fullURL := fmt.Sprintf("%s/?%s", c.BaseURL, params.Encode())

	// Make the HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Piper: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("piper returned error (status %d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

var (
	markdownBoldItalicRegex = regexp.MustCompile(`(\*\*|\*|__|_)`)
	markdownLinkRegex       = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	markdownHeaderRegex     = regexp.MustCompile(`(?m)^#{1,6}\s`)
	markdownListRegex       = regexp.MustCompile(`(?m)^[\t ]*[-*+]\s`)
	markdownCodeRegex       = regexp.MustCompile("`")
	markdownRuleRegex       = regexp.MustCompile(`(?m)^---[\t ]*$`)
	markdownMiscRegex       = regexp.MustCompile(`[~=|$]+`)
	markdownThinkRegex      = regexp.MustCompile(`(?s)<think>.*?</think>`)
)

// CleanMarkdown removes Markdown formatting characters for natural speech synthesis.
func CleanMarkdown(text string) string {
	// 0. Remove reasoning blocks
	text = markdownThinkRegex.ReplaceAllString(text, "")

	// 1. Remove horizontal rules
	text = markdownRuleRegex.ReplaceAllString(text, "")

	// 2. Convert links [text](url) -> text
	text = markdownLinkRegex.ReplaceAllString(text, "$1")

	// 3. Remove headers ###
	text = markdownHeaderRegex.ReplaceAllString(text, "")

	// 4. Remove list markers (at start of line)
	text = markdownListRegex.ReplaceAllString(text, "")

	// 5. Remove bold/italic markers
	text = markdownBoldItalicRegex.ReplaceAllString(text, "")

	// 6. Remove backticks
	text = markdownCodeRegex.ReplaceAllString(text, "")

	// 7. Remove miscellaneous markers (blockquotes, strike, highlight, math)
	text = markdownMiscRegex.ReplaceAllString(text, "")

	return strings.TrimSpace(text)
}
