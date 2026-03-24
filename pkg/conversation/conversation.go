package conversation

import (
	"time"

	"papyrus/pkg/llm"
)

// Conversation represents a multi-turn conversation with a document.
type Conversation struct {
	DocumentText string            `json:"document_text"`
	FileName     string            `json:"file_name"`
	Messages     []llm.ChatMessage `json:"messages"`
	CreatedAt    time.Time         `json:"created_at"`
	SessionID    string            `json:"session_id,omitempty"`
}

// New creates a new conversation with a document.
func New(fileName, documentText string) *Conversation {
	return &Conversation{
		DocumentText: documentText,
		FileName:     fileName,
		Messages:     []llm.ChatMessage{},
		CreatedAt:    time.Now(),
		SessionID:    "",
	}
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
