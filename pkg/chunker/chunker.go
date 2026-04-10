package chunker

import (
	"github.com/pkoukk/tiktoken-go"
)

// Chunk represents a piece of text with metadata.
type Chunk struct {
	Index   int
	Content string
}

var encoding *tiktoken.Tiktoken

func init() {
	enc, _ := tiktoken.GetEncoding("cl100k_base")
	encoding = enc
}

// Split breaks text into chunks of roughly chunkSize tokens with overlap.
func Split(text string, chunkSize, overlap int) []Chunk {
	if text == "" {
		return nil
	}

	if encoding == nil {
		// Fallback to simple line-based chunking if tiktoken is unavailable
		return simpleSplit(text, chunkSize*4) // Rough char estimate
	}

	tokens := encoding.Encode(text, nil, nil)
	var chunks []Chunk
	index := 0

	for start := 0; start < len(tokens); start += (chunkSize - overlap) {
		end := start + chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		chunkTokens := tokens[start:end]
		content := encoding.Decode(chunkTokens)

		chunks = append(chunks, Chunk{
			Index:   index,
			Content: content,
		})
		index++

		if end == len(tokens) {
			break
		}
	}

	return chunks
}

func simpleSplit(text string, chunkSizeChars int) []Chunk {
	var chunks []Chunk
	index := 0
	runes := []rune(text)

	for i := 0; i < len(runes); i += chunkSizeChars {
		end := i + chunkSizeChars
		if end > len(runes) {
			end = len(runes)
		}

		chunks = append(chunks, Chunk{
			Index:   index,
			Content: string(runes[i:end]),
		})
		index++
	}
	return chunks
}
