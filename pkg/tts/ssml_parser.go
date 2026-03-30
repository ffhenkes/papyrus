package tts

import (
	"fmt"
	"regexp"
	"strings"
)

// Prosody represents speech prosody (rate, pitch, volume)
type Prosody struct {
	Rate   string // e.g., "1.0", "slow", "fast"
	Pitch  string // e.g., "1.0", "high", "low"
	Volume string // e.g., "100%", "loud", "soft"
}

// SSMLSegment represents a text segment with associated metadata
type SSMLSegment struct {
	Text    string
	Voice   string // e.g., "pt_BR-faber-medium"
	Prosody *Prosody
}

// SSMLBreak represents a time break
type SSMLBreak struct {
	DurationMs int // milliseconds
}

// SSMLElement is either a segment or a break
type SSMLElement interface {
	isSSMLElement()
}

func (s *SSMLSegment) isSSMLElement() {}
func (b *SSMLBreak) isSSMLElement()   {}

// ParseSSML parses SSML markup and returns a list of elements (segments and breaks)
// If the input is not SSML (no <speak> tag), returns a single segment with the text
func ParseSSML(input string) ([]SSMLElement, error) {
	input = strings.TrimSpace(input)

	// Check if this is SSML
	if !strings.HasPrefix(input, "<speak") {
		// Not SSML, return as plain text
		return []SSMLElement{&SSMLSegment{Text: input}}, nil
	}

	// Extract content between <speak> tags
	speakContent, err := extractTagContent(input, "speak")
	if err != nil {
		return nil, fmt.Errorf("invalid SSML structure: %w", err)
	}

	// Parse the content recursively
	elements, err := parseSSMLContent(speakContent, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSML content: %w", err)
	}

	return elements, nil
}

// parseSSMLContent recursively parses SSML content and returns elements
func parseSSMLContent(content string, defaultVoice string) ([]SSMLElement, error) {
	var elements []SSMLElement
	pos := 0

	for pos < len(content) {
		// Find next tag
		tagStart := strings.Index(content[pos:], "<")
		if tagStart == -1 {
			// No more tags, add remaining text
			remaining := strings.TrimSpace(content[pos:])
			if remaining != "" {
				elements = append(elements, &SSMLSegment{
					Text:  remaining,
					Voice: defaultVoice,
				})
			}
			break
		}

		tagStart += pos

		// Add text before tag
		textBefore := strings.TrimSpace(content[pos:tagStart])
		if textBefore != "" {
			elements = append(elements, &SSMLSegment{
				Text:  textBefore,
				Voice: defaultVoice,
			})
		}

		// Find tag end
		tagEnd := strings.Index(content[tagStart:], ">")
		if tagEnd == -1 {
			return nil, fmt.Errorf("unclosed tag at position %d", tagStart)
		}
		tagEnd += tagStart + 1

		// Parse the tag (includes < and >)
		tagContent := content[tagStart:tagEnd]

		switch {
		case isBreakTag(tagContent):
			// Parse break tag
			duration := parseBreakDuration(tagContent)
			elements = append(elements, &SSMLBreak{DurationMs: duration})
			pos = tagEnd

		case isVoiceTag(tagContent):
			// Parse voice tag - it's a container tag
			voiceName := extractAttribute(tagContent, "name")
			closingTag := "</" + extractTagName(tagContent) + ">"
			closingPos := strings.Index(content[tagEnd:], closingTag)
			if closingPos == -1 {
				return nil, fmt.Errorf("unclosed voice tag")
			}
			closingPos += tagEnd

			// Parse content inside voice tag
			innerContent := content[tagEnd:closingPos]
			innerElements, err := parseSSMLContent(innerContent, voiceName)
			if err != nil {
				return nil, err
			}

			// Override voice for all inner segments if voice name is set
			if voiceName != "" {
				for _, elem := range innerElements {
					if seg, ok := elem.(*SSMLSegment); ok {
						seg.Voice = voiceName
					}
				}
			}

			elements = append(elements, innerElements...)
			pos = closingPos + len(closingTag)

		case isProsodyTag(tagContent):
			// Parse prosody tag - it's a container tag
			prosody := parseProsodyTag(tagContent)
			closingTag := "</" + extractTagName(tagContent) + ">"
			closingPos := strings.Index(content[tagEnd:], closingTag)
			if closingPos == -1 {
				return nil, fmt.Errorf("unclosed prosody tag")
			}
			closingPos += tagEnd

			// Parse content inside prosody tag
			innerContent := content[tagEnd:closingPos]
			innerElements, err := parseSSMLContent(innerContent, defaultVoice)
			if err != nil {
				return nil, err
			}

			// Apply prosody to all inner segments
			for _, elem := range innerElements {
				if seg, ok := elem.(*SSMLSegment); ok {
					seg.Prosody = prosody
				}
			}

			elements = append(elements, innerElements...)
			pos = closingPos + len(closingTag)

		case isSentenceTag(tagContent):
			// Parse sentence tag (<s>) - container tag
			closingTag := "</" + extractTagName(tagContent) + ">"
			closingPos := strings.Index(content[tagEnd:], closingTag)
			if closingPos == -1 {
				return nil, fmt.Errorf("unclosed sentence tag")
			}
			closingPos += tagEnd

			// Parse content inside sentence tag
			innerContent := content[tagEnd:closingPos]
			innerElements, err := parseSSMLContent(innerContent, defaultVoice)
			if err != nil {
				return nil, err
			}

			elements = append(elements, innerElements...)
			pos = closingPos + len(closingTag)

		default:
			// Unknown tag, skip it
			pos = tagEnd
		}
	}

	return elements, nil
}

// extractTagContent extracts content between opening and closing tags
func extractTagContent(input string, tagName string) (string, error) {
	openTag := "<" + tagName
	closeTag := "</" + tagName + ">"

	// Find opening tag
	openPos := strings.Index(input, openTag)
	if openPos == -1 {
		return "", fmt.Errorf("tag <%s> not found", tagName)
	}

	// Find end of opening tag
	tagEndPos := strings.Index(input[openPos:], ">")
	if tagEndPos == -1 {
		return "", fmt.Errorf("malformed opening tag <%s>", tagName)
	}
	tagEndPos += openPos + 1

	// Find closing tag
	closePos := strings.Index(input[tagEndPos:], closeTag)
	if closePos == -1 {
		return "", fmt.Errorf("closing tag </%s> not found", tagName)
	}
	closePos += tagEndPos

	return input[tagEndPos:closePos], nil
}

// extractTagName extracts the tag name from a tag string
func extractTagName(tag string) string {
	// Remove < and attributes
	tag = strings.TrimPrefix(tag, "<")
	tag = strings.TrimSuffix(tag, ">")
	tag = strings.TrimSuffix(tag, "/")
	tag = strings.TrimSpace(tag)

	// Find space or closing char
	spaceIdx := strings.IndexAny(tag, " \t/>")
	if spaceIdx > 0 {
		return tag[:spaceIdx]
	}
	return tag
}

// isBreakTag checks if a tag is a break tag
func isBreakTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "<break") && (strings.HasSuffix(tag, "/>") || strings.HasSuffix(tag, ">"))
}

// isProsodyTag checks if a tag is a prosody tag
func isProsodyTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "<prosody") && strings.HasSuffix(tag, ">")
}

// isVoiceTag checks if a tag is a voice tag
func isVoiceTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "<voice") && strings.HasSuffix(tag, ">")
}

// isSentenceTag checks if a tag is a sentence tag (<s>)
func isSentenceTag(tag string) bool {
	tag = strings.TrimSpace(tag)
	return strings.HasPrefix(tag, "<s") && strings.HasSuffix(tag, ">") && !strings.HasPrefix(tag, "<speak")
}

// parseBreakDuration extracts break duration from break tag
func parseBreakDuration(tag string) int {
	// Look for time attribute: time="500ms" or time="0.5s"
	timeRe := regexp.MustCompile(`time\s*=\s*"([^"]+)"`)
	matches := timeRe.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return 200 // Default 200ms
	}

	timeStr := matches[1]
	return parseDuration(timeStr)
}

// parseDuration converts duration string (e.g., "500ms", "1s") to milliseconds
func parseDuration(durationStr string) int {
	durationStr = strings.TrimSpace(durationStr)

	if strings.HasSuffix(durationStr, "ms") {
		ms := strings.TrimSuffix(durationStr, "ms")
		var val int
		_, _ = fmt.Sscanf(ms, "%d", &val)
		return val
	}

	if strings.HasSuffix(durationStr, "s") {
		s := strings.TrimSuffix(durationStr, "s")
		var val float64
		_, _ = fmt.Sscanf(s, "%f", &val)
		return int(val * 1000)
	}

	// Try to parse as integer milliseconds
	var ms int
	_, _ = fmt.Sscanf(durationStr, "%d", &ms)
	return ms
}

// extractAttribute extracts an attribute value from a tag
func extractAttribute(tag string, attrName string) string {
	// Pattern: attrName="value" or attrName='value'
	pattern := regexp.MustCompile(attrName + `\s*=\s*["']([^"']*?)["']`)
	matches := pattern.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// parseProsodyTag extracts prosody attributes
func parseProsodyTag(tag string) *Prosody {
	return &Prosody{
		Rate:   extractAttribute(tag, "rate"),
		Pitch:  extractAttribute(tag, "pitch"),
		Volume: extractAttribute(tag, "volume"),
	}
}

// ConvertElementsToString converts SSML elements back to string for debugging
func ConvertElementsToString(elements []SSMLElement) string {
	var parts []string
	for _, elem := range elements {
		switch e := elem.(type) {
		case *SSMLSegment:
			if e.Voice != "" {
				parts = append(parts, fmt.Sprintf("[%s: %s]", e.Voice, e.Text))
			} else {
				parts = append(parts, e.Text)
			}
		case *SSMLBreak:
			parts = append(parts, fmt.Sprintf("[BREAK: %dms]", e.DurationMs))
		}
	}
	return strings.Join(parts, " ")
}
