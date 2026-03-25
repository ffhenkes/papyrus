package llm

import (
	"fmt"
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

var encoding *tiktoken.Tiktoken

func init() {
	// Initialize the cl100k_base BPE encoder (A highly accurate proxy for Llama3/Mistral token ratios)
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err == nil {
		encoding = enc
	}
}

// TokenStats tracks the token usage for a single request or cumulatively.
type TokenStats struct {
	InputTokens  int     // Estimated input tokens
	OutputTokens int     // Estimated output tokens
	TotalTokens  int     // InputTokens + OutputTokens
	TokensPerSec float64 // Output tokens per second during generation
}

// EstimateTokens provides a highly accurate BPE token count.
// It gracefully falls back to the 1.33 heuristic if the encoder failed to initialize.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	if encoding != nil {
		// tiktoken Encode returns []int (the tokens)
		tokens := encoding.Encode(text, nil, nil)
		return len(tokens)
	}

	// Heuristic fallback: roughly 1.33 tokens per word
	words := len(strings.Fields(text))
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
	// Note: TokensPerSec is point-in-time calculation, ignored for cumulative stats.
}
