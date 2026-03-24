package conversation

import (
	"strings"

	"papyrus/pkg/llm"
)

// SummarizeOldMessages extracts meaningful text from removed messages to keep context.
// It uses a simple truncation approach to save tokens.
func SummarizeOldMessages(messages []llm.ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Previous context summary:\n")
	for _, msg := range messages {
		sb.WriteString(msg.Role + ": " + summarizeText(msg.Content) + "\n")
	}
	return sb.String()
}

func summarizeText(content string) string {
	// Basic shortening if it's very long
	if len(content) > 150 {
		return content[:147] + "..."
	}
	return content
}

// PruneHistory keeps the most recent messages while staying under maxTokens.
// It returns the kept messages and a summary string of the pruned messages.
func PruneHistory(messages []llm.ChatMessage, maxTokens int) ([]llm.ChatMessage, string) {
	if len(messages) <= 1 {
		return messages, ""
	}

	totalTokens := 0
	keepIndex := len(messages)

	// Work backwards from most recent message
	for i := len(messages) - 1; i >= 0; i-- {
		tokens := llm.EstimateTokens(messages[i].Content)
		if totalTokens+tokens > maxTokens {
			break
		}
		totalTokens += tokens
		keepIndex = i
	}

	if keepIndex == 0 {
		return messages, "" // Everything fits
	}

	// Try to keep user-assistant pairs together if possible
	// Typically evens are user, odds are assistant
	if keepIndex%2 != 0 && keepIndex < len(messages) {
		keepIndex++ // Prune the orphaned assistant reply too
	}

	if keepIndex >= len(messages) {
		// Pruned everything
		return []llm.ChatMessage{}, SummarizeOldMessages(messages)
	}

	pruned := messages[:keepIndex]
	summary := SummarizeOldMessages(pruned)

	kept := make([]llm.ChatMessage, len(messages[keepIndex:]))
	copy(kept, messages[keepIndex:])

	return kept, summary
}
