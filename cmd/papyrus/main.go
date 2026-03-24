package main

import (
	"fmt"
	"os"
	"strings"

	"papyrus/internal/config"
	"papyrus/pkg/conversation"
	"papyrus/pkg/llm"
	"papyrus/pkg/pdf"
	"papyrus/pkg/repl"
)

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

	ollamaURL := getEnv("OLLAMA_URL", config.DefaultOllamaURL)
	modelName := getEnv("OLLAMA_MODEL", config.DefaultModel)

	fmt.Printf("[PDF] Reading PDF: %s\n", pdfPath)
	text, err := pdf.ExtractText(pdfPath)
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

	// Create conversation with the PDF document
	conv := conversation.New(pdfPath, text)

	// Prepare initial prompt with document context
	userPrompt := "Please read the following document content and provide a clear, comprehensive explanation of its contents."
	if customPrompt != "" {
		userPrompt = customPrompt
	}

	fullUserMessage := fmt.Sprintf("%s\n\n<document>\n%s\n</document>", userPrompt, text)

	// Create LLM client
	client := llm.NewClient(ollamaURL, modelName, config.MaxTokens)

	// Send initial message (this also adds it to conversation history)
	explanation, err := client.SendMessage(conv.GetHistory(), fullUserMessage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add messages to conversation for multi-turn support
	conv.AddMessage("user", fullUserMessage)
	conv.AddMessage("assistant", explanation)

	fmt.Println(explanation)

	// Enter interactive REPL mode for follow-up questions
	r := repl.New(client, conv)
	if err := r.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
		os.Exit(1)
	}
}

// getEnv retrieves an environment variable with a fallback value.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// printUsage prints the usage information.
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
