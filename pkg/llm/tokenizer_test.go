package llm

import "testing"

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"single word", "hello", 1},
		{"two words", "hello world", 2},
		{"three words", "hello world testing", 3},
		{"long string", "the quick brown fox jumps over the lazy dog", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.input)
			if got != tt.expected {
				t.Errorf("EstimateTokens() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestFormatTokenStats(t *testing.T) {
	stats := TokenStats{
		InputTokens:  1000,
		OutputTokens: 250,
		TotalTokens:  1250,
		TokensPerSec: 25.5,
	}
	expected := "[tokens] 1000 in \u2192 250 out (25.5 t/s)"
	got := FormatTokenStats(stats)
	if got != expected {
		t.Errorf("FormatTokenStats() = %v, want %v", got, expected)
	}
}

func TestAddTokenStats(t *testing.T) {
	stats1 := TokenStats{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	stats2 := TokenStats{InputTokens: 20, OutputTokens: 15, TotalTokens: 35}

	stats1.Add(stats2)

	if stats1.InputTokens != 30 {
		t.Errorf("Expected InputTokens 30, got %d", stats1.InputTokens)
	}
	if stats1.OutputTokens != 20 {
		t.Errorf("Expected OutputTokens 20, got %d", stats1.OutputTokens)
	}
	if stats1.TotalTokens != 50 {
		t.Errorf("Expected TotalTokens 50, got %d", stats1.TotalTokens)
	}
}
