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
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
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
	Message   ChatMessage `json:"message"`
	Reasoning string      `json:"reasoning_content,omitempty"`
	Error     string      `json:"error,omitempty"`
	Done      bool        `json:"done"`
}

// Client represents an Ollama API client.
type Client struct {
	URL          string
	ModelName    string
	MaxTokens    int
	DocumentText string         // Document text stored once to avoid resending on each follow-up
	Cache        *ResponseCache // Optional response cache
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
func (c *Client) SendMessage(messages []ChatMessage, userMessage string, onToken func(string)) (string, TokenStats, error) {
	// Check cache
	if c.Cache != nil {
		cacheKey := NormalizeKey(userMessage)
		if cachedResponse, found := c.Cache.Get(cacheKey); found {
			fmt.Fprintf(os.Stderr, "\r[OK] Retrieved from cache (instant)              \n")
			if onToken != nil {
				onToken(cachedResponse)
			}
			stats := TokenStats{
				InputTokens:  EstimateTokens(userMessage),
				OutputTokens: EstimateTokens(cachedResponse),
				TotalTokens:  EstimateTokens(userMessage) + EstimateTokens(cachedResponse),
				TokensPerSec: 0,
			}
			return cachedResponse, stats, nil
		}
	}

	// Build request with full conversation history
	req := ChatRequest{
		Model:  c.ModelName,
		Stream: onToken != nil,
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

	fullResponse, duration, err := c.doRequestStream(req, onToken)
	if err != nil {
		return "", TokenStats{}, err
	}

	// Save to cache
	if c.Cache != nil {
		cacheKey := NormalizeKey(userMessage)
		c.Cache.Set(cacheKey, fullResponse)
		_ = c.Cache.Save()
	}

	historyText := ""
	for _, m := range messages {
		historyText += m.Content + " "
	}
	inputTokens := EstimateTokens(historyText + userMessage)
	outputTokens := EstimateTokens(fullResponse)

	stats := TokenStats{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		TokensPerSec: float64(outputTokens) / duration,
	}

	return fullResponse, stats, nil
}

// SendMessageWithDoc sends a message with document context without embedding document in history.
// This prevents re-sending the entire document with every follow-up question.
func (c *Client) SendMessageWithDoc(messages []ChatMessage, userMessage, documentContext string, onToken func(string)) (string, TokenStats, error) {
	// Check cache
	if c.Cache != nil {
		cacheKey := NormalizeKey(userMessage)
		if cachedResponse, found := c.Cache.Get(cacheKey); found {
			fmt.Fprintf(os.Stderr, "\r[OK] Retrieved from cache (instant)              \n")
			if onToken != nil {
				onToken(cachedResponse)
			}
			stats := TokenStats{
				InputTokens:  EstimateTokens(documentContext + " " + userMessage),
				OutputTokens: EstimateTokens(cachedResponse),
				TotalTokens:  EstimateTokens(documentContext+" "+userMessage) + EstimateTokens(cachedResponse),
				TokensPerSec: 0,
			}
			return cachedResponse, stats, nil
		}
	}

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
		Stream: onToken != nil,
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

	fullResponse, duration, err := c.doRequestStream(req, onToken)
	if err != nil {
		return "", TokenStats{}, err
	}

	// Save to cache
	if c.Cache != nil {
		cacheKey := NormalizeKey(userMessage)
		c.Cache.Set(cacheKey, fullResponse)
		_ = c.Cache.Save()
	}

	historyText := ""
	for _, m := range messages {
		historyText += m.Content + " "
	}
	inputTokens := EstimateTokens(documentContext + " " + historyText + userMessage)
	outputTokens := EstimateTokens(fullResponse)

	stats := TokenStats{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
		TokensPerSec: float64(outputTokens) / duration,
	}

	return fullResponse, stats, nil
}

func (c *Client) doRequestStream(req ChatRequest, onToken func(string)) (string, float64, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(c.URL, "/") + "/api/chat"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	spinnerDone := make(chan bool)
	startTime := time.Now()
	go spinner(spinnerDone, c.ModelName, startTime)

	client := &http.Client{}
	resp, err := client.Do(httpReq)

	if err != nil {
		close(spinnerDone)
		fmt.Fprintf(os.Stderr, "\r") // Clear spinner line
		return "", 0, fmt.Errorf("could not reach Ollama at %s — is it running? (%w)", c.URL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "\nError closing response body: %v\n", err)
		}
	}()

	var fullResponse strings.Builder
	firstToken := false

	if req.Stream {
		decoder := json.NewDecoder(resp.Body)
		lastType := "content" // current stream state: "content" or "reasoning"

		for {
			var chatResp ChatResponse
			err := decoder.Decode(&chatResp)
			if err == io.EOF {
				break
			}
			if err != nil {
				if !firstToken {
					close(spinnerDone)
				}
				return "", 0, fmt.Errorf("failed to decode stream: %w", err)
			}
			if chatResp.Error != "" {
				if !firstToken {
					close(spinnerDone)
				}
				return "", 0, fmt.Errorf("ollama error: %s", chatResp.Error)
			}

			if !firstToken {
				close(spinnerDone)
				time.Sleep(50 * time.Millisecond)  // Let spinner erase itself
				fmt.Fprintf(os.Stderr, "\r\033[K") // Clear line
				firstToken = true
			}

			// Handle Reasoning Content (native field)
			if chatResp.Reasoning != "" {
				if lastType != "reasoning" {
					if onToken != nil {
						onToken("<think>\n")
					}
					fullResponse.WriteString("<think>\n")
					lastType = "reasoning"
				}
				if onToken != nil {
					onToken(chatResp.Reasoning)
				}
				fullResponse.WriteString(chatResp.Reasoning)
			}

			// Handle Regular Content
			if chatResp.Message.Content != "" {
				if lastType == "reasoning" {
					if onToken != nil {
						onToken("\n</think>\n\n")
					}
					fullResponse.WriteString("\n</think>\n\n")
					lastType = "content"
				}
				if onToken != nil {
					onToken(chatResp.Message.Content)
				}
				fullResponse.WriteString(chatResp.Message.Content)
			}

			if chatResp.Done {
				// Ensure think block is closed if the response ended in reasoning
				if lastType == "reasoning" {
					if onToken != nil {
						onToken("\n</think>\n")
					}
					fullResponse.WriteString("\n</think>\n")
				}
				break
			}
		}
	} else {
		respBody, err := io.ReadAll(resp.Body)
		close(spinnerDone)
		time.Sleep(100 * time.Millisecond) // Allow spinner to finish printing

		if err != nil {
			return "", 0, fmt.Errorf("failed to read response: %w", err)
		}

		var chatResp ChatResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			return "", 0, fmt.Errorf("failed to parse response: %w\nraw: %s", err, string(respBody))
		}

		if chatResp.Error != "" {
			if strings.Contains(chatResp.Error, "model") && strings.Contains(chatResp.Error, "not found") {
				return "", 0, fmt.Errorf("ollama model '%s' not found. Please run 'ollama pull %s' (error: %s)", c.ModelName, c.ModelName, chatResp.Error)
			}
			return "", 0, fmt.Errorf("ollama error: %s", chatResp.Error)
		}
		fullResponse.WriteString(chatResp.Message.Content)
	}

	duration := time.Since(startTime).Seconds()
	return fullResponse.String(), duration, nil
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
