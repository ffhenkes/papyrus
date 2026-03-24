package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ChatMessage represents a single message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request payload for the Ollama API.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  ChatOptions   `json:"options"`
}

// ChatOptions contains options for the chat request.
type ChatOptions struct {
	NumPredict int `json:"num_predict"`
}

// ChatResponse is the response payload from the Ollama API.
type ChatResponse struct {
	Message ChatMessage `json:"message"`
	Error   string      `json:"error,omitempty"`
}

// Client represents an Ollama API client.
type Client struct {
	URL          string
	ModelName    string
	MaxTokens    int
	DocumentText string // Document text stored once to avoid resending on each follow-up
}

// NewClient creates a new Ollama API client.
func NewClient(url, modelName string, maxTokens int) *Client {
	return &Client{
		URL:       url,
		ModelName: modelName,
		MaxTokens: maxTokens,
	}
}

// SendMessage sends a message to Ollama with conversation history.
func (c *Client) SendMessage(messages []ChatMessage, userMessage string) (string, error) {
	// Build request with full conversation history
	req := ChatRequest{
		Model:  c.ModelName,
		Stream: false,
		Options: ChatOptions{
			NumPredict: c.MaxTokens,
		},
		Messages: []ChatMessage{
			{
				Role: "system",
				Content: `You are an expert document analyst. When given document content, you:
1. Identify the document type and purpose
2. Summarize the key topics and main points clearly
3. Highlight important details, data, or findings
4. Explain any technical concepts in accessible language
5. Note the document structure and how it's organized

Be thorough but concise. Use bullet points and sections to organize your explanation.`,
			},
		},
	}

	// Append all previous messages (conversation history)
	req.Messages = append(req.Messages, messages...)

	// Append the new user message
	req.Messages = append(req.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(c.URL, "/") + "/api/chat"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Show progress spinner while waiting for LLM response
	done := make(chan bool)
	startTime := time.Now()
	go spinner(done, c.ModelName, startTime)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	close(done)
	time.Sleep(100 * time.Millisecond) // Allow spinner to finish printing

	if err != nil {
		fmt.Fprintf(os.Stderr, "\r") // Clear spinner line
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (%w)", c.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w\nraw: %s", err, string(respBody))
	}

	if chatResp.Error != "" {
		if strings.Contains(chatResp.Error, "model") && strings.Contains(chatResp.Error, "not found") {
			return "", fmt.Errorf("ollama model '%s' not found. Please run 'ollama pull %s' inside the Ollama container or on your host (error: %s)", c.ModelName, c.ModelName, chatResp.Error)
		}
		return "", fmt.Errorf("ollama error: %s", chatResp.Error)
	}

	return chatResp.Message.Content, nil
}

// SendMessageWithDoc sends a message with document context without embedding document in history.
// This prevents re-sending the entire document with every follow-up question.
func (c *Client) SendMessageWithDoc(messages []ChatMessage, userMessage, documentContext string) (string, error) {
	// Build request with full conversation history (without embedded document)
	systemPrompt := `You are an expert document analyst. When given document content, you:
1. Identify the document type and purpose
2. Summarize the key topics and main points clearly
3. Highlight important details, data, or findings
4. Explain any technical concepts in accessible language
5. Note the document structure and how it's organized

Be thorough but concise. Use bullet points and sections to organize your explanation.`

	// Include document context in system message for first message, reference for follow-ups
	if documentContext != "" {
		systemPrompt = fmt.Sprintf("%s\n\nDOCUMENT CONTENT:\n<document>\n%s\n</document>", systemPrompt, documentContext)
	}

	req := ChatRequest{
		Model:  c.ModelName,
		Stream: false,
		Options: ChatOptions{
			NumPredict: c.MaxTokens,
		},
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
		},
	}

	// Append all previous messages (conversation history WITHOUT document embedded)
	req.Messages = append(req.Messages, messages...)

	// Append the new user message
	req.Messages = append(req.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(c.URL, "/") + "/api/chat"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Show progress spinner while waiting for LLM response
	done := make(chan bool)
	startTime := time.Now()
	go spinner(done, c.ModelName, startTime)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	close(done)
	time.Sleep(100 * time.Millisecond) // Allow spinner to finish printing

	if err != nil {
		fmt.Fprintf(os.Stderr, "\r") // Clear spinner line
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (%w)", c.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing response body: %v\n", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w\nraw: %s", err, string(respBody))
	}

	if chatResp.Error != "" {
		if strings.Contains(chatResp.Error, "model") && strings.Contains(chatResp.Error, "not found") {
			return "", fmt.Errorf("ollama model '%s' not found. Please run 'ollama pull %s' inside the Ollama container or on your host (error: %s)", c.ModelName, c.ModelName, chatResp.Error)
		}
		return "", fmt.Errorf("ollama error: %s", chatResp.Error)
	}

	return chatResp.Message.Content, nil
}

// formatDuration returns a human-readable string of the elapsed time.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dμs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.1fm", d.Minutes())
	default:
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// spinner displays a progress spinner while waiting for a response.
func spinner(done chan bool, modelName string, startTime time.Time) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			elapsed := time.Since(startTime)
			fmt.Fprintf(os.Stderr, "\r[OK] Analysis complete! (%s)              \n", formatDuration(elapsed))
			return
		default:
			fmt.Fprintf(os.Stderr, "\r[..] %s Processing with %s...", frames[i%len(frames)], modelName)
			i++
			time.Sleep(100 * time.Millisecond)
		}
	}
}
