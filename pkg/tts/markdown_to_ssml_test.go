package tts

import (
	"strings"
	"testing"
)

// TestMarkdownToSSMLBasic tests basic markdown to SSML conversion
func TestMarkdownToSSMLBasic(t *testing.T) {
	config := DefaultMarkdownSSMLConfig()
	input := "This is **bold** text."
	output := MarkdownToSSML(input, config)

	// Just verify that the basic structure is there
	if !strings.Contains(output, "<prosody") && !strings.Contains(output, "<s>") {
		t.Errorf("Expected SSML elements, got: %s", output)
	}
	if !strings.Contains(output, "bold") {
		t.Errorf("Expected 'bold' text to be preserved, got: %s", output)
	}
}

// TestConvertHeadersToSSML tests header conversion
func TestConvertHeadersToSSML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "H1 header",
			input:    "# Main Title",
			contains: "<prosody",
		},
		{
			name:     "H2 header",
			input:    "## Subheading",
			contains: "<prosody",
		},
		{
			name:     "H3 header",
			input:    "### Sub-subheading",
			contains: "<prosody",
		},
		{
			name:     "Multiple headers",
			input:    "# Title\n## Subtitle",
			contains: "Title",
		},
	}

	config := DefaultMarkdownSSMLConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := convertHeadersToSSML(tt.input, config)

			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, output)
			}
		})
	}
}

// TestConvertListsToSSML tests list conversion
func TestConvertListsToSSML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "Unordered list with dash",
			input:    "- Item 1\n- Item 2",
			contains: "• Item 1",
		},
		{
			name:     "Unordered list with asterisk",
			input:    "* Item 1",
			contains: "• Item 1",
		},
		{
			name:     "Unordered list with plus",
			input:    "+ Item 1",
			contains: "• Item 1",
		},
		{
			name:     "Ordered list",
			input:    "1. First item\n2. Second item",
			contains: "First item",
		},
	}

	config := DefaultMarkdownSSMLConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := convertListsToSSML(tt.input, config)

			if !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, output)
			}
		})
	}
}

// TestConvertQuotesToSSML tests quote conversion
func TestConvertQuotesToSSML(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		shouldHave string
	}{
		{
			name:       "Simple quote",
			input:      "> This is a quote",
			shouldHave: "This is a quote",
		},
		{
			name:       "Multiple quotes",
			input:      "> First quote\n> Second quote",
			shouldHave: "First quote",
		},
	}

	config := DefaultMarkdownSSMLConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := convertQuotesToSSML(tt.input, config)

			if !strings.Contains(output, tt.shouldHave) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.shouldHave, output)
			}
		})
	}
}

// TestConvertBoldToSSML tests bold/italic conversion
func TestConvertBoldToSSML(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    string
		shouldNotHave string
	}{
		{
			name:          "Bold with asterisks",
			input:         "This is **bold** text",
			shouldHave:    "<prosody",
			shouldNotHave: "**",
		},
		{
			name:          "Bold with underscores",
			input:         "This is __bold__ text",
			shouldHave:    "<prosody",
			shouldNotHave: "__",
		},
		{
			name:          "Italic with asterisks",
			input:         "This is *italic* text",
			shouldHave:    "<prosody",
			shouldNotHave: "*italic*",
		},
		{
			name:          "Italic with underscores",
			input:         "This is _italic_ text",
			shouldHave:    "<prosody",
			shouldNotHave: "_italic_",
		},
	}

	config := DefaultMarkdownSSMLConfig()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := convertBoldToSSML(tt.input, config)

			if !strings.Contains(output, tt.shouldHave) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.shouldHave, output)
			}
			if tt.shouldNotHave != "" && strings.Contains(output, tt.shouldNotHave) {
				t.Errorf("Expected output to NOT contain '%s', got: %s", tt.shouldNotHave, output)
			}
		})
	}
}

// TestRemoveCodeBlocks tests code block removal
func TestRemoveCodeBlocks(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    string
		shouldNotHave string
	}{
		{
			name:          "Remove fenced code block",
			input:         "Text\n```go\nfunc test() {}\n```\nMore text",
			shouldHave:    "Text",
			shouldNotHave: "func test",
		},
		{
			name:          "Remove indented code block",
			input:         "Text\n    func test() {}\nMore text",
			shouldHave:    "Text",
			shouldNotHave: "func test",
		},
		{
			name:       "Keep text outside code",
			input:      "Keep this ```remove this``` keep this",
			shouldHave: "Keep this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := removeCodeBlocks(tt.input)

			if !strings.Contains(output, tt.shouldHave) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.shouldHave, output)
			}
			if tt.shouldNotHave != "" && strings.Contains(output, tt.shouldNotHave) {
				t.Errorf("Expected output to NOT contain '%s', got: %s", tt.shouldNotHave, output)
			}
		})
	}
}

// TestWrapSentences tests sentence wrapping
func TestWrapSentences(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain string
	}{
		{
			name:          "Single sentence",
			input:         "This is a sentence.",
			shouldContain: "<s>This is a sentence.</s>",
		},
		{
			name:          "Multiple sentences",
			input:         "First sentence. Second sentence.",
			shouldContain: "<s>",
		},
		{
			name:          "Question mark",
			input:         "Is this a question?",
			shouldContain: "<s>Is this a question?</s>",
		},
		{
			name:          "Exclamation mark",
			input:         "What an exclamation!",
			shouldContain: "<s>What an exclamation!</s>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := wrapSentences(tt.input)

			if !strings.Contains(output, tt.shouldContain) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.shouldContain, output)
			}
		})
	}
}

// TestDetectSentences tests sentence detection
func TestDetectSentences(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		shouldContain string
	}{
		{
			name:          "Single sentence",
			input:         "This is a sentence.",
			expectedCount: 1,
			shouldContain: "This is a sentence.",
		},
		{
			name:          "Two sentences",
			input:         "First sentence. Second sentence.",
			expectedCount: 2,
			shouldContain: "First sentence.",
		},
		{
			name:          "Question and statement",
			input:         "Is this a question? This is a statement.",
			expectedCount: 2,
			shouldContain: "Is this a question?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := DetectSentences(tt.input)

			if len(sentences) != tt.expectedCount {
				t.Errorf("Expected %d sentences, got %d", tt.expectedCount, len(sentences))
			}

			found := false
			for _, s := range sentences {
				if s.Text == tt.shouldContain {
					found = true
					break
				}
			}

			if !found && tt.shouldContain != "" {
				t.Errorf("Expected to find sentence '%s' in detected sentences", tt.shouldContain)
			}
		})
	}
}

// TestDefaultMarkdownSSMLConfig tests the default configuration
func TestDefaultMarkdownSSMLConfig(t *testing.T) {
	config := DefaultMarkdownSSMLConfig()

	if config.BoldRateMultiplier <= 0 {
		t.Errorf("BoldRateMultiplier should be positive, got: %f", config.BoldRateMultiplier)
	}
	if config.HeaderBreakBefore < 0 {
		t.Errorf("HeaderBreakBefore should be non-negative, got: %d", config.HeaderBreakBefore)
	}
	if config.WrapSentencesInTags {
		t.Errorf("WrapSentencesInTags should be false by default (LLM content has structure)")
	}
	if !config.RemoveCodeBlocks {
		t.Errorf("RemoveCodeBlocks should be true by default")
	}
}

// TestConversationalMinimalConfig tests the conversational config for REPL output
func TestConversationalMinimalConfig(t *testing.T) {
	config := ConversationalMinimalConfig()

	// All multipliers should be 1.0 (no prosody changes)
	if config.BoldRateMultiplier != 1.0 {
		t.Errorf("BoldRateMultiplier should be 1.0 for conversational, got: %f", config.BoldRateMultiplier)
	}
	if config.BoldPitchMultiplier != 1.0 {
		t.Errorf("BoldPitchMultiplier should be 1.0 for conversational, got: %f", config.BoldPitchMultiplier)
	}
	if config.HeaderRateMultiplier != 1.0 {
		t.Errorf("HeaderRateMultiplier should be 1.0 for conversational, got: %f", config.HeaderRateMultiplier)
	}

	// All breaks should be 0 (no breaks added)
	if config.HeaderBreakBefore != 0 {
		t.Errorf("HeaderBreakBefore should be 0 for conversational, got: %d", config.HeaderBreakBefore)
	}
	if config.ListItemBreak != 0 {
		t.Errorf("ListItemBreak should be 0 for conversational, got: %d", config.ListItemBreak)
	}

	// Code block removal should still be enabled
	if !config.RemoveCodeBlocks {
		t.Errorf("RemoveCodeBlocks should be true for conversational")
	}
}

// TestComplexMarkdown tests a complex markdown document
func TestComplexMarkdown(t *testing.T) {
	input := `# Main Title

This is the introduction with **bold text** and *italic text*.

## Section 1

Here's a list:
- First item
- Second item
- Third item

> This is an important quote

Some more text with code:
` + "```go\nfunc main() {}\n```" + `

## Section 2

1. Ordered item one
2. Ordered item two

Final paragraph with **emphasis**.`

	config := DefaultMarkdownSSMLConfig()
	output := MarkdownToSSML(input, config)

	// Verify code blocks are removed
	if strings.Contains(output, "func main") {
		t.Errorf("Code blocks should be removed: %s", output)
	}

	// Verify headers are converted
	if !strings.Contains(output, "Main Title") {
		t.Errorf("Header text should be preserved: %s", output)
	}

	// Verify lists are converted
	if !strings.Contains(output, "• First item") {
		t.Errorf("List items should be converted: %s", output)
	}

	// Verify quotes are converted
	if !strings.Contains(output, "This is an important quote") {
		t.Errorf("Quote text should be preserved: %s", output)
	}

	// Verify bold is converted
	if strings.Contains(output, "**bold text**") {
		t.Errorf("Bold markers should be converted to SSML: %s", output)
	}
}

// TestEmptyAndNilInputs tests edge cases
func TestEmptyAndNilInputs(t *testing.T) {
	config := DefaultMarkdownSSMLConfig()

	// Empty string
	output := MarkdownToSSML("", config)
	if output != "" {
		t.Errorf("Empty input should return empty output, got: %s", output)
	}

	// Whitespace only
	output = MarkdownToSSML("   \n\t  ", config)
	if len(strings.TrimSpace(output)) > 10 {
		t.Errorf("Whitespace-only input should produce minimal output, got: %s", output)
	}
}

// TestFloatToStr tests float to string conversion
func TestFloatToStr(t *testing.T) {
	tests := []struct {
		input    float32
		expected string
	}{
		{1.0, "1"},
		{1.05, "1.05"},
		{1.1, "1.1"},
		{0.95, "0.95"},
	}

	for _, tt := range tests {
		result := floatToStr(tt.input)
		if result != tt.expected {
			t.Errorf("floatToStr(%f) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// TestIntToMS tests integer to milliseconds conversion
func TestIntToMS(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{100, "100ms"},
		{50, "50ms"},
		{0, "0ms"},
		{1000, "1000ms"},
	}

	for _, tt := range tests {
		result := intToMS(tt.input)
		if result != tt.expected {
			t.Errorf("intToMS(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
