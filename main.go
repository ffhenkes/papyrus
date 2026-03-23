package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// --- Config ---

const (
	defaultOllamaURL = "http://host.docker.internal:11434"
	defaultModel     = "qwen3:8b"
	maxTokens        = 4096
)

// --- Ollama OpenAI-compatible types ---

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  ChatOptions   `json:"options"`
}

type ChatOptions struct {
	NumPredict int `json:"num_predict"`
}

type ChatResponse struct {
	Message ChatMessage `json:"message"`
	Error   string      `json:"error,omitempty"`
}

// --- Main ---

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	pdfPath := args[0]
	customPrompt := ""
	if len(args) > 1 {
		customPrompt = strings.Join(args[1:], " ")
	}

	ollamaURL := getEnv("OLLAMA_URL", defaultOllamaURL)
	modelName := getEnv("OLLAMA_MODEL", defaultModel)

	fmt.Printf("📄 Reading PDF: %s\n", pdfPath)
	text, err := extractPDFText(pdfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting PDF text: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "Error: could not extract any text from the PDF (scanned image PDF?)")
		os.Exit(1)
	}

	fmt.Printf("-> Extracted %d characters of text\n", len(text))
	fmt.Printf("# Papyrus → %s (%s)...\n", ollamaURL, modelName)
	fmt.Println(strings.Repeat("─", 60))

	explanation, err := explainText(ollamaURL, modelName, text, customPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(explanation)
}

// --- PDF Text Extraction ---

func extractPDFText(pdfPath string) (string, error) {
	conf := pdfcpu.NewDefaultConfiguration()

	ctx, err := api.ReadContextFile(pdfPath, conf)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= ctx.PageCount; i++ {
		page, err := ctx.ExtractPageText(i)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not extract text from page %d: %v\n", i, err)
			continue
		}
		if strings.TrimSpace(page) == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n--- Page %d ---\n", i))
		sb.WriteString(page)
	}

	return sb.String(), nil
}

// --- Ollama API Call ---

func explainText(ollamaURL, modelName, text, customPrompt string) (string, error) {
	userPrompt := "Please read the following document content and provide a clear, comprehensive explanation of its contents."
	if customPrompt != "" {
		userPrompt = customPrompt
	}

	fullUserMessage := fmt.Sprintf("%s\n\n<document>\n%s\n</document>", userPrompt, text)

	req := ChatRequest{
		Model:  modelName,
		Stream: false,
		Options: ChatOptions{
			NumPredict: maxTokens,
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
			{
				Role:    "user",
				Content: fullUserMessage,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(ollamaURL, "/") + "/api/chat"
	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (%w)", ollamaURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w\nraw: %s", err, string(respBody))
	}

	if chatResp.Error != "" {
		return "", fmt.Errorf("ollama error: %s", chatResp.Error)
	}

	return chatResp.Message.Content, nil
}

// --- Helpers ---

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: papyrus <path-to-pdf> [custom prompt]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment variables:")
	fmt.Fprintln(os.Stderr, "  OLLAMA_URL    Ollama base URL (default: http://host.docker.internal:11434)")
	fmt.Fprintln(os.Stderr, "  OLLAMA_MODEL  Model to use    (default: qwen3:8b)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  papyrus document.pdf")
	fmt.Fprintln(os.Stderr, "  papyrus document.pdf 'Focus on the technical details'")
	fmt.Fprintln(os.Stderr, "  OLLAMA_MODEL=deepseek-r1:14b papyrus document.pdf")
}
