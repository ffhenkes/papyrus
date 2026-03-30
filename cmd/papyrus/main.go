package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"papyrus/internal/config"
	"papyrus/pkg/conversation"
	"papyrus/pkg/llm"
	"papyrus/pkg/pdf"
	"papyrus/pkg/repl"
	"papyrus/pkg/tts"
)

func main() {
	// Define flags
	fs := flag.NewFlagSet("papyrus", flag.ExitOnError)
	sessionID := fs.String("session", "", "Resume an existing session by ID")
	listSessions := fs.Bool("list", false, "List all saved sessions and exit")
	listSessions2 := fs.Bool("sessions", false, "List all saved sessions and exit (alias for --list)")
	deleteSession := fs.String("delete", "", "Delete a saved session by ID")
	noCache := fs.Bool("no-cache", false, "Disable semantic caching for LLM responses")
	maxContext := fs.Int("max-context", 8192, "Maximum tokens to keep in conversation history before pruning")
	exportFlag := fs.Bool("export", false, "Analyze document, export conversation to Markdown, and exit instantly")
	ttsFlag := fs.Bool("tts", false, "Enable text-to-speech for model responses")

	// Parse flags (allowing positional args to remain)
	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Fallback for flags passed after positional args
	for _, arg := range os.Args {
		if arg == "--tts" {
			*ttsFlag = true
		}
		if arg == "--no-cache" {
			*noCache = true
		}
	}

	args := fs.Args()

	sessionDir := getSessionDir()

	// Handle --list or --sessions flag
	if *listSessions || *listSessions2 {
		handleListSessions(sessionDir)
		return
	}

	// Handle --delete flag
	if *deleteSession != "" {
		handleDeleteSession(*deleteSession, sessionDir)
		return
	}

	// Initialize TTS if enabled (uses Piper engine with SSML support)
	var ttsEngine tts.TTSEngine
	isSSML := false
	if *ttsFlag {
		piperURL := getEnv("PIPER_URL", "http://localhost:5000")
		piperClient := tts.NewPiperClient(piperURL)
		piperClient.DefaultVoice = strings.Trim(os.Getenv("PIPER_VOICE"), "\"' ")
		ttsEngine = piperClient
		isSSML = true // Piper supports SSML parsing and synthesis
		fmt.Printf("[TTS] Using Piper at %s (SSML enabled)\n", piperURL)
	}

	// Handle session resumption via --session flag
	if *sessionID != "" {
		handleResumeSession(*sessionID, sessionDir, *noCache, *maxContext, ttsEngine, isSSML)
		return
	}

	// Normal flow: new PDF analysis
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	pdfPath := args[0]
	customPrompt := ""
	if len(args) > 1 {
		customPrompt = strings.Join(args[1:], " ")
	}

	// Extract and analyze PDF
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

	// Check if session already exists and prompt
	if conversation.SessionExists(conv.SessionID, sessionDir) {
		fmt.Printf("\nSession '%s' already exists. Overwrite? (y/n): ", conv.SessionID)
		var response string
		_, _ = fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Cancelled. Use --session <ID> to resume an existing session, or --list to see all sessions.")
			os.Exit(0)
		}
	}

	// Prepare initial prompt with document context
	userPrompt := "Please read the following document content and provide a clear, comprehensive explanation of its contents."
	if customPrompt != "" {
		userPrompt = customPrompt
	}

	fullUserMessage := fmt.Sprintf("%s\n\n<document>\n%s\n</document>", userPrompt, text)

	// Create LLM client
	client := llm.NewClient(ollamaURL, modelName, config.MaxTokens)
	if !*noCache {
		homeDir, _ := os.UserHomeDir()
		client.Cache = llm.NewResponseCache(filepath.Join(homeDir, ".papyrus", "cache", conv.SessionID+".cache.json"))
	}
	client.DocumentText = text
	client.IsSSML = isSSML

	// Handle case where flags were passed after the filename
	if !*ttsFlag && len(args) > 0 {
		// Simple check for --tts in the remaining args if not already set
		for _, arg := range os.Args {
			if arg == "--tts" {
				*ttsFlag = true
				break
			}
		}
	}

	// Send initial message with document context
	fmt.Println("\n=== Explanation ===")
	explanation, stats, err := client.SendMessageWithDoc([]llm.ChatMessage{}, fullUserMessage, text, func(token string) {
		fmt.Print(token)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add messages to conversation for multi-turn support
	conv.AddMessage("user", userPrompt)
	conv.AddMessage("assistant", explanation)

	// Generate speech if enabled and text is not empty/just symbols
	if ttsEngine != nil && strings.TrimSpace(tts.CleanMarkdown(explanation)) != "" {
		voiceFile := filepath.Join("voice", fmt.Sprintf("%s_initial.wav", conv.SessionID))
		fmt.Printf("\n[TTS] Generating speech: %s... ", voiceFile)
		if err := synthesizeToFile(context.Background(), ttsEngine, explanation, isSSML, voiceFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			fmt.Println("Done.")
		}
	}

	fmt.Println() // ensure newline after stream finishes
	fmt.Println(llm.FormatTokenStats(stats))

	// Save session before entering REPL
	if err := conversation.SaveSession(conv, sessionDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save session: %v\n", err)
	} else {
		fmt.Printf("\n[Session] Saved as '%s'. Use --session %s to resume.\n", conv.SessionID, conv.SessionID)
	}

	if *exportFlag {
		md := conversation.ExportMarkdown(conv)
		exportFile := fmt.Sprintf("%s_export.md", conv.SessionID)
		if err := os.WriteFile(exportFile, []byte(md), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write export file: %v\n", err)
		} else {
			fmt.Printf("\n[Export] Saved conversation to %s\n", exportFile)
		}
		return // Exit without entering REPL
	}

	// Enter interactive REPL mode for follow-up questions
	r := repl.New(client, conv, sessionDir, *maxContext)
	if ttsEngine != nil {
		r.WithTTS(ttsEngine, isSSML)
	}
	if err := r.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
		os.Exit(1)
	}
}

// getSessionDir returns the directory where sessions are stored.
func getSessionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir unavailable
		return ".papyrus/sessions"
	}
	return filepath.Join(home, ".papyrus", "sessions")
}

// handleListSessions displays all saved sessions and exits.
func handleListSessions(sessionDir string) {
	sessions, err := conversation.ListSessions(sessionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading sessions: %v\n", err)
		os.Exit(1)
	}

	if len(sessions) == 0 {
		fmt.Println("No saved sessions found.")
		return
	}

	fmt.Println("\n=== Saved Sessions ===")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("%-30s | %-20s | %s\n", "Session ID", "File", "Questions")
	fmt.Println(strings.Repeat("─", 80))

	for _, session := range sessions {
		shortID := session.SessionID
		if len(shortID) > 28 {
			shortID = shortID[:25] + "..."
		}
		fmt.Printf("%-30s | %-20s | %d\n", shortID, filepath.Base(session.FileName), session.MessageCount/2)
	}
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("\nTo resume a session: papyrus --session <SESSION_ID>\n")
}

// handleResumeSession loads an existing session and enters REPL mode.
func handleResumeSession(sessionID, sessionDir string, noCache bool, maxContext int, ttsEngine tts.TTSEngine, isSSML bool) {
	conv, err := conversation.LoadSession(sessionID, sessionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ollamaURL := getEnv("OLLAMA_URL", config.DefaultOllamaURL)
	modelName := getEnv("OLLAMA_MODEL", config.DefaultModel)

	fmt.Printf("[Session] Resuming '%s' (%s)\n", sessionID, conv.FileName)
	fmt.Printf("-> %d messages in conversation\n", len(conv.Messages))
	fmt.Printf("# Papyrus → %s (%s)\n", ollamaURL, modelName)
	fmt.Println(strings.Repeat("─", 60))

	// Normal flow will use ttsFlag from global flags
	// But in resume mode, we need to check if --tts was passed
	// (Actually fs.Parse was called at the start of main)

	// Recreate LLM client with document context
	client := llm.NewClient(ollamaURL, modelName, config.MaxTokens)
	if !noCache {
		homeDir, _ := os.UserHomeDir()
		client.Cache = llm.NewResponseCache(filepath.Join(homeDir, ".papyrus", "cache", sessionID+".cache.json"))
	}
	client.DocumentText = conv.DocumentText
	client.IsSSML = isSSML

	// Display last few messages as context
	fmt.Println("\n--- Conversation so far ---")
	if len(conv.Messages) > 4 {
		fmt.Println("...")
		for _, msg := range conv.Messages[len(conv.Messages)-4:] {
			role := strings.ToUpper(msg.Role)
			content := msg.Content
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			fmt.Printf("\n[%s]:\n%s\n", role, content)
		}
	} else {
		for _, msg := range conv.Messages {
			role := strings.ToUpper(msg.Role)
			fmt.Printf("\n[%s]:\n%s\n", role, msg.Content)
		}
	}
	fmt.Println("\n" + strings.Repeat("─", 60))

	// Enter REPL with existing conversation
	r := repl.New(client, conv, sessionDir, maxContext)
	if ttsEngine != nil {
		r.WithTTS(ttsEngine, isSSML)
	}
	if err := r.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
		os.Exit(1)
	}
}

// synthesizeToFile is a helper to run synthesis and write to disk
func synthesizeToFile(ctx context.Context, engine tts.TTSEngine, text string, isSSML bool, outputPath string) error {
	data, err := engine.Synthesize(ctx, text, isSSML)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(outputPath, data, 0600)
}

// getEnv retrieves an environment variable with a fallback value.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// handleDeleteSession deletes a saved session by ID.
func handleDeleteSession(sessionID, sessionDir string) {
	if err := conversation.DeleteSession(sessionID, sessionDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Session '%s' deleted.\n", sessionID)
}

// printUsage prints the usage information.
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: papyrus <path-to-pdf> [custom prompt]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --session ID    Resume an existing session")
	fmt.Fprintln(os.Stderr, "  --list          List all saved sessions")
	fmt.Fprintln(os.Stderr, "  --sessions      List all saved sessions (alias)")
	fmt.Fprintln(os.Stderr, "  --delete ID     Delete a saved session")
	fmt.Fprintln(os.Stderr, "  --no-cache      Disable semantic caching for LLM responses")
	fmt.Fprintln(os.Stderr, "  --max-context N Max tokens in conversation history before pruning (default: 8192)")
	fmt.Fprintln(os.Stderr, "  --export        Export session to Markdown and exit immediately")
	fmt.Fprintln(os.Stderr, "  --tts           Enable text-to-speech for model responses")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Environment variables:")
	fmt.Fprintln(os.Stderr, "  OLLAMA_URL    Ollama base URL (default: http://host.docker.internal:11434)")
	fmt.Fprintln(os.Stderr, "  OLLAMA_MODEL  Model to use    (default: qwen3:8b)")
	fmt.Fprintln(os.Stderr, "  PIPER_URL     Piper HTTP URL  (default: http://localhost:5000)")
	fmt.Fprintln(os.Stderr, "  PIPER_VOICE   Piper voice ID  (default: en_US-hfc_female-medium)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  papyrus document.pdf")
	fmt.Fprintln(os.Stderr, "  papyrus document.pdf 'Focus on the technical details'")
	fmt.Fprintln(os.Stderr, "  papyrus --list")
	fmt.Fprintln(os.Stderr, "  papyrus --session my-doc-abc123def456")
	fmt.Fprintln(os.Stderr, "  papyrus --delete my-doc-abc123def456")
	fmt.Fprintln(os.Stderr, "  OLLAMA_MODEL=deepseek-r1:14b papyrus document.pdf")
}
