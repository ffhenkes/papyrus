package conversation

import (
	"crypto/sha256"
	"fmt"
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

// generateSessionID creates a deterministic session ID from filename and timestamp.
// Format: filename-timestamp-hash for uniqueness.
func generateSessionID(fileName string) string {
	// Use SHA256 hash of filename + timestamp for collision avoidance
	hash := sha256.Sum256([]byte(fileName + time.Now().Format(time.RFC3339Nano)))
	return fmt.Sprintf("%s-%s",
		fileName[:len(fileName)-len(".pdf")], // Remove .pdf extension
		fmt.Sprintf("%x", hash)[:12])         // Use first 12 hex chars
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
