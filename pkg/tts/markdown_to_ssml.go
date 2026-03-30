package tts

import (
	"fmt"
	"regexp"
	"strings"
)

// MarkdownSSMLConfig holds configuration for markdown to SSML conversion
type MarkdownSSMLConfig struct {
	// Rate multipliers (0.5-2.0)
	BoldRateMultiplier    float32
	BoldPitchMultiplier   float32
	HeaderRateMultiplier  float32
	HeaderPitchMultiplier float32
	ListRateMultiplier    float32
	QuoteRateMultiplier   float32
	QuotePitchMultiplier  float32

	// Break durations (in milliseconds)
	HeaderBreakBefore int
	HeaderBreakAfter  int
	ListItemBreak     int
	QuoteBreak        int

	// Behavior flags
	WrapSentencesInTags bool
	RemoveCodeBlocks    bool
}

// DefaultMarkdownSSMLConfig returns sensible defaults for markdown-to-SSML conversion
func DefaultMarkdownSSMLConfig() MarkdownSSMLConfig {
	return MarkdownSSMLConfig{
		BoldRateMultiplier:    0.95,  // Slightly slower for emphasis
		BoldPitchMultiplier:   1.05,  // Slightly higher pitch
		HeaderRateMultiplier:  1.05,  // Slightly faster
		HeaderPitchMultiplier: 1.02,  // Slightly higher pitch
		ListRateMultiplier:    0.98,  // Very slightly slower for clarity
		QuoteRateMultiplier:   0.95,  // Slightly slower for reading
		QuotePitchMultiplier:  0.95,  // Slightly lower pitch for quotes
		HeaderBreakBefore:     50,    // 50ms before header (subtle)
		HeaderBreakAfter:      50,    // 50ms after header (subtle)
		ListItemBreak:         25,    // 25ms between list items (minimal)
		QuoteBreak:            50,    // 50ms around quotes (subtle)
		WrapSentencesInTags:   false, // Don't wrap by default - LLM content often has structure
		RemoveCodeBlocks:      true,
	}
}

// ConversationalMinimalConfig returns minimal config for conversational output (REPL interactions)
// Only cleans markdown, doesn't add prosody or breaks
func ConversationalMinimalConfig() MarkdownSSMLConfig {
	return MarkdownSSMLConfig{
		BoldRateMultiplier:    1.0,   // No rate change for bold
		BoldPitchMultiplier:   1.0,   // No pitch change for bold
		HeaderRateMultiplier:  1.0,   // No rate change for headers
		HeaderPitchMultiplier: 1.0,   // No pitch change for headers
		ListRateMultiplier:    1.0,   // No rate change for lists
		QuoteRateMultiplier:   1.0,   // No rate change for quotes
		QuotePitchMultiplier:  1.0,   // No pitch change for quotes
		HeaderBreakBefore:     0,     // No break
		HeaderBreakAfter:      0,     // No break
		ListItemBreak:         0,     // No break
		QuoteBreak:            0,     // No break
		WrapSentencesInTags:   false, // Don't wrap sentences
		RemoveCodeBlocks:      true,  // Still remove code blocks
	}
}

// Sentence represents a parsed sentence with its content and type
type Sentence struct {
	Text     string // The actual sentence text
	StartIdx int    // Start index in original text
	EndIdx   int    // End index in original text
	IsSpeech bool   // Whether this is meant for speech (vs code blocks)
}

// MarkdownToSSML converts markdown text to SSML with prosody and breaks
func MarkdownToSSML(text string, config MarkdownSSMLConfig) string {
	if text == "" {
		return text
	}

	// Skip processing if content already contains SSML or markup
	if strings.Contains(text, "<speak") ||
		strings.Contains(text, "<break") ||
		strings.Contains(text, "<voice") ||
		strings.Contains(text, "<b>") ||
		strings.Contains(text, "<ul>") ||
		strings.Contains(text, "<li>") {
		// Already formatted, return as-is
		return text
	}

	result := text

	// Remove code blocks first (they should never be spoken)
	if config.RemoveCodeBlocks {
		result = removeCodeBlocks(result)
	}

	// Order matters: headers, lists, quotes, then bold/italic
	result = convertHeadersToSSML(result, config)
	result = convertListsToSSML(result, config)
	result = convertQuotesToSSML(result, config)
	result = convertBoldToSSML(result, config)

	// Wrap sentences in tags if configured
	if config.WrapSentencesInTags {
		result = wrapSentences(result)
	}

	return result
}

// convertHeadersToSSML converts markdown headers to SSML with prosody and breaks
// # Header1, ## Header2, etc.
func convertHeadersToSSML(text string, config MarkdownSSMLConfig) string {
	// Match headers: # Header, ## Header, ### Header, etc.
	headerRegex := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+?)$`)

	return headerRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := headerRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		level := len(parts[1]) // Number of # symbols
		headerText := strings.TrimSpace(parts[2])

		// Adjust prosody based on header level
		rate := 1.0 + (float32(6-level) * 0.05)
		if rate > 1.2 {
			rate = 1.2
		}

		pitch := 1.0 + (float32(6-level) * 0.05)
		if pitch > 1.2 {
			pitch = 1.2
		}

		beforeBreak := config.HeaderBreakBefore
		afterBreak := config.HeaderBreakAfter

		return strings.Join([]string{
			"<break time=\"" + intToMS(beforeBreak) + "\"/>",
			"<s><prosody rate=\"" + floatToStr(rate) + "\" pitch=\"" + floatToStr(pitch) + "\">" + headerText + "</prosody></s>",
			"<break time=\"" + intToMS(afterBreak) + "\"/>",
		}, "")
	})
}

// convertListsToSSML converts markdown lists to SSML with prosody and breaks
// - item
// * item
// + item
// 1. item
func convertListsToSSML(text string, config MarkdownSSMLConfig) string {
	// Match unordered list items: -, *, +
	unorderedRegex := regexp.MustCompile(`(?m)^[\s]*[-*+]\s+(.+?)$`)

	text = unorderedRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := unorderedRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		itemText := strings.TrimSpace(parts[1])

		return strings.Join([]string{
			"<s><prosody rate=\"" + floatToStr(config.ListRateMultiplier) + "\">• " + itemText + "</prosody></s>",
			"<break time=\"" + intToMS(config.ListItemBreak) + "\"/>",
		}, "")
	})

	// Match ordered list items: 1. 2. etc.
	orderedRegex := regexp.MustCompile(`(?m)^[\s]*\d+\.\s+(.+?)$`)

	text = orderedRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := orderedRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		itemText := strings.TrimSpace(parts[1])

		return strings.Join([]string{
			"<s><prosody rate=\"" + floatToStr(config.ListRateMultiplier) + "\">" + itemText + "</prosody></s>",
			"<break time=\"" + intToMS(config.ListItemBreak) + "\"/>",
		}, "")
	})

	return text
}

// convertQuotesToSSML converts markdown block quotes to SSML with prosody and breaks
// > quote
func convertQuotesToSSML(text string, config MarkdownSSMLConfig) string {
	// Match quote lines: > text
	// Can be multi-line if lines start with >
	quoteRegex := regexp.MustCompile(`(?m)^>\s+(.+?)$`)

	return quoteRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := quoteRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		quoteText := strings.TrimSpace(parts[1])

		return strings.Join([]string{
			"<break time=\"" + intToMS(config.QuoteBreak) + "\"/>",
			"<s><prosody rate=\"" + floatToStr(config.QuoteRateMultiplier) + "\" pitch=\"" + floatToStr(config.QuotePitchMultiplier) + "\">\"" + quoteText + "\"</prosody></s>",
			"<break time=\"" + intToMS(config.QuoteBreak) + "\"/>",
		}, "")
	})
}

// convertBoldToSSML converts markdown bold and italic to SSML with prosody
// **bold** or __bold__ or *italic* or _italic_
func convertBoldToSSML(text string, config MarkdownSSMLConfig) string {
	// Convert **bold** to <prosody>
	boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*|__(.+?)__`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := boldRegex.FindStringSubmatch(match)
		var boldText string
		if parts[1] != "" {
			boldText = parts[1]
		} else {
			boldText = parts[2]
		}

		return "<prosody rate=\"" + floatToStr(config.BoldRateMultiplier) + "\" pitch=\"" + floatToStr(config.BoldPitchMultiplier) + "\">" + boldText + "</prosody>"
	})

	// Convert *italic* and _italic_ to subtle prosody (use bold settings for now, could be customized)
	italicRegex := regexp.MustCompile(`\*(.+?)\*|_(.+?)_`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := italicRegex.FindStringSubmatch(match)
		var italicText string
		if parts[1] != "" {
			italicText = parts[1]
		} else {
			italicText = parts[2]
		}

		// Italic: slightly higher pitch, same rate as normal
		return "<prosody pitch=\"1.05\">" + italicText + "</prosody>"
	})

	return text
}

// removeCodeBlocks removes code blocks from text (both fenced and inline)
// ```code``` and `code`
func removeCodeBlocks(text string) string {
	// Remove fenced code blocks (```...```)
	fencedRegex := regexp.MustCompile("(?s)```.*?```")
	text = fencedRegex.ReplaceAllString(text, "")

	// Remove indented code blocks (4+ spaces at start of line)
	indentedRegex := regexp.MustCompile(`(?m)^    .*$`)
	text = indentedRegex.ReplaceAllString(text, "")

	// Note: We keep inline code with backticks because they might be important
	// Remove them only if you want to skip inline code too:
	// inlineRegex := regexp.MustCompile("`([^`]+)`")
	// text = inlineRegex.ReplaceAllString(text, "$1")

	return text
}

// wrapSentences wraps sentences in <s> tags
// This is conservative and only wraps complete sentences ending with . ! or ?
func wrapSentences(text string) string {
	// Skip if already contains SSML
	if strings.Contains(text, "</s>") {
		return text
	}

	// Match sentences ending with . ! or ?
	// This is intentionally simple to avoid breaking on edge cases
	sentenceRegex := regexp.MustCompile(`([^.!?]*[.!?])(?:\s+|$)`)

	var result strings.Builder
	matches := sentenceRegex.FindAllStringIndex(text, -1)

	if len(matches) == 0 {
		// No sentences found with ending punctuation, return as-is
		return text
	}

	lastEnd := 0
	for _, match := range matches {
		start := match[0]
		end := match[1]

		// Preserve any whitespace before the sentence
		sentence := text[start:end]
		trimmed := strings.TrimSpace(sentence)

		if trimmed != "" {
			result.WriteString("<s>")
			result.WriteString(trimmed)
			result.WriteString("</s>")
		}

		// Add whitespace between sentences
		if end < len(text) && text[end] == ' ' {
			result.WriteString(" ")
		}

		lastEnd = end
	}

	// Add any remaining text
	if lastEnd < len(text) {
		remaining := strings.TrimSpace(text[lastEnd:])
		if remaining != "" {
			result.WriteString("<s>")
			result.WriteString(remaining)
			result.WriteString("</s>")
		}
	}

	return result.String()
}

// DetectSentences parses text into individual sentences
func DetectSentences(text string) []Sentence {
	var sentences []Sentence

	// Match sentences ending with . ! or ?
	sentenceRegex := regexp.MustCompile(`([^.!?]*[.!?])`)
	matches := sentenceRegex.FindAllStringSubmatchIndex(text, -1)

	for _, match := range matches {
		start := match[0]
		end := match[1]

		sentence := strings.TrimSpace(text[start:end])
		if sentence != "" {
			sentences = append(sentences, Sentence{
				Text:     sentence,
				StartIdx: start,
				EndIdx:   end,
				IsSpeech: true,
			})
		}
	}

	return sentences
}

// Helper functions for SSML generation

// floatToStr converts a float32 to a string with 2 decimal places
func floatToStr(f float32) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

// intToMS converts milliseconds integer to SSML duration string
func intToMS(ms int) string {
	return fmt.Sprintf("%dms", ms)
}
