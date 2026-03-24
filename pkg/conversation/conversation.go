package conversation

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"time"

	"papyrus/pkg/llm"
)

// Conversation represents a multi-turn conversation with a document.
type Conversation struct {
	DocumentText string            `json:"document_text"`
	FileName     string            `json:"file_name"`
	Messages     []llm.ChatMessage `json:"messages"`
	CreatedAt    time.Time         `json:"created_at"`
	LastUpdated  time.Time         `json:"last_updated"`
	SessionID    string            `json:"session_id,omitempty"`
}

// New creates a new conversation with a document.
func New(fileName, documentText string) *Conversation {
	now := time.Now()
	return &Conversation{
		DocumentText: documentText,
		FileName:     fileName,
		Messages:     []llm.ChatMessage{},
		CreatedAt:    now,
		LastUpdated:  now,
		SessionID:    generateSessionID(fileName),
	}
}

// generateSessionID creates a unique session ID from filename, timestamp, and random entropy.
// Format: base-hash. Safely strips any file extension.
func generateSessionID(fileName string) string {
	// Safely strip directory and extension from filename
	base := filepath.Base(fileName)
	if ext := filepath.Ext(base); ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" || base == "." {
		base = "session"
	}
	// Mix filename + nanosecond timestamp + random bytes for guaranteed uniqueness
	rndBytes := make([]byte, 8)
	_, _ = rand.Read(rndBytes)
	hash := sha256.Sum256(append([]byte(fileName+time.Now().Format(time.RFC3339Nano)), rndBytes...))
	return fmt.Sprintf("%s-%s", base, fmt.Sprintf("%x", hash)[:12])
}

// AddMessage adds a new message to the conversation.
func (c *Conversation) AddMessage(role, content string) {
	c.Messages = append(c.Messages, llm.ChatMessage{
		Role:    role,
		Content: content,
	})
}

// GetHistory returns a copy of the message history.
func (c *Conversation) GetHistory() []llm.ChatMessage {
	history := make([]llm.ChatMessage, len(c.Messages))
	copy(history, c.Messages)
	return history
}
