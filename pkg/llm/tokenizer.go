package llm

import (
	"fmt"
	"strings"
)

// TokenStats tracks the token usage for a single request or cumulatively.
type TokenStats struct {
	InputTokens  int     // Estimated input tokens
	OutputTokens int     // Estimated output tokens
	TotalTokens  int     // InputTokens + OutputTokens
	TokensPerSec float64 // Output tokens per second during generation
}

// EstimateTokens provides a simple, dependency-free heuristic for token counting.
// It assumes roughly 0.75 words per token (or ~1.33 tokens per word), which is
// standard for Latin-alphabet text using subword tokenization (like BPE/WordPiece).
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	// Integer math: (words * 4) / 3 gives ~1.33 multiplier
	return (words * 4) / 3
}

// FormatTokenStats creates a human-readable display string for token usage.
func FormatTokenStats(stats TokenStats) string {
	return fmt.Sprintf("[tokens] %d in \u2192 %d out (%.1f t/s)", stats.InputTokens, stats.OutputTokens, stats.TokensPerSec)
}

// Add combines two TokenStats together for cumulative tracking.
func (t *TokenStats) Add(other TokenStats) {
	t.InputTokens += other.InputTokens
	t.OutputTokens += other.OutputTokens
	t.TotalTokens += other.TotalTokens
	// Note: TokensPerSec doesn't make logical sense to sum up,
	// so it's left out of cumulative tracking.
}
