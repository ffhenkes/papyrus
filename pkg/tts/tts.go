package tts

import (
	"bytes"
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
	BaseURL string
}

// NewPiperClient creates a new Piper TTS client.
func NewPiperClient(baseURL string) *PiperClient {
	return &PiperClient{BaseURL: baseURL}
}

// Synthesize sends text to Piper and returns the resulting audio bytes.
func (c *PiperClient) Synthesize(ctx context.Context, text string, isSSML bool) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Piper doesn't support SSML, so we always clean the text
	cleanText := CleanMarkdown(text)

	// Prepare the request URL
	params := url.Values{}
	params.Add("text", cleanText)
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

// OpenTTSClient handles communication with the OpenTTS service.
type OpenTTSClient struct {
	BaseURL string
	Voice   string // Default voice ID (e.g., "espeak:en")
}

// NewOpenTTSClient creates a new OpenTTS client.
func NewOpenTTSClient(baseURL string) *OpenTTSClient {
	return &OpenTTSClient{BaseURL: baseURL}
}

// Synthesize sends text to OpenTTS and returns the resulting audio bytes.
func (c *OpenTTSClient) Synthesize(ctx context.Context, text string, isSSML bool) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	params := url.Values{}
	if c.Voice != "" {
		params.Add("voice", c.Voice)
	}

	// Clean markdown to avoid issues with specialized symbols in TTS engines
	text = CleanMarkdown(text)

	endpoint := fmt.Sprintf("%s/api/tts?%s", c.BaseURL, params.Encode())
	contentType := "text/plain"
	if isSSML {
		contentType = "application/ssml+xml"
		// Ensure text is wrapped in <speak>
		trimmed := strings.TrimSpace(text)
		if !strings.HasPrefix(trimmed, "<speak") {
			text = fmt.Sprintf("<speak>\n%s\n</speak>", text)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(text))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OpenTTS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opentts returned error (status %d): %s", resp.StatusCode, string(body))
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
