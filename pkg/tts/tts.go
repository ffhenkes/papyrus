package tts

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Client handles communication with the Piper TTS service.
type Client struct {
	BaseURL string
}

// NewClient creates a new TTS client.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

// Synthesize sends text to Piper and saves the resulting audio to a WAV file.
func (c *Client) Synthesize(text string, outputPath string) error {
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}

	cleanText := CleanMarkdown(text)

	// Prepare the request URL
	params := url.Values{}
	params.Add("text", cleanText)
	fullURL := fmt.Sprintf("%s/?%s", c.BaseURL, params.Encode())

	// Make the HTTP request
	// #nosec G107
	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Piper: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("piper returned error (status %d): %s", resp.StatusCode, string(body))
	}

	// Ensure the output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the output file
	// #nosec G304
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Stream the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save audio file: %w", err)
	}

	return nil
}

var (
	markdownBoldItalicRegex = regexp.MustCompile(`(\*\*|\*|__|_)`)
	markdownLinkRegex       = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	markdownHeaderRegex     = regexp.MustCompile(`(?m)^#{1,6}\s`)
	markdownListRegex       = regexp.MustCompile(`(?m)^[\t ]*[-*+]\s`)
	markdownCodeRegex       = regexp.MustCompile("`")
	markdownRuleRegex       = regexp.MustCompile(`(?m)^---[\t ]*$`)
	markdownMiscRegex       = regexp.MustCompile(`[~=|>|$]+`)
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
