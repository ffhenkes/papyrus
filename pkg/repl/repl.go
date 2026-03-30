package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"papyrus/pkg/conversation"
	"papyrus/pkg/llm"
	"papyrus/pkg/tts"
)

// REPL represents a read-eval-print loop for interactive conversations.
type REPL struct {
	client     *llm.Client
	conv       *conversation.Conversation
	reader     io.Reader
	writer     io.Writer
	done       chan bool
	sessionDir string
	stats      llm.TokenStats
	maxContext int
	ttsEngine  tts.TTSEngine
	isSSML     bool
}

// New creates a new REPL session.
func New(client *llm.Client, conv *conversation.Conversation, sessionDir string, maxContext int) *REPL {
	return &REPL{
		client:     client,
		conv:       conv,
		reader:     os.Stdin,
		writer:     os.Stdout,
		done:       make(chan bool),
		sessionDir: sessionDir,
		maxContext: maxContext,
	}
}

// WithTTS enables text-to-speech for the REPL.
func (r *REPL) WithTTS(engine tts.TTSEngine, isSSML bool) {
	r.ttsEngine = engine
	r.isSSML = isSSML
}

// Start begins the interactive REPL loop.
func (r *REPL) Start() error {
	scanner := bufio.NewScanner(r.reader)

	_, _ = fmt.Fprintln(r.writer, strings.Repeat("─", 60))
	_, _ = fmt.Fprintln(r.writer, "Interactive mode. Commands: 'quit', 'exit', 'history', 'save', 'session info'")
	_, _ = fmt.Fprintln(r.writer, strings.Repeat("─", 60))

	for {
		_, _ = fmt.Fprint(r.writer, "\n> ")
		if !scanner.Scan() {
			// EOF reached - auto-save on exit
			if err := r.saveSession(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: auto-save failed: %v\n", err)
			}
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle built-in commands
		if r.handleCommand(input) {
			break // quit/exit command
		}
	}

	return scanner.Err()
}

// handleCommand processes built-in commands like quit, exit, history, save.
// Returns true if the REPL should exit.
func (r *REPL) handleCommand(input string) bool {
	lowerInput := strings.ToLower(input)

	switch lowerInput {
	case "quit", "exit":
		// Save before exiting
		_ = r.saveSession()
		_, _ = fmt.Fprintln(r.writer, "Goodbye!")
		return true

	case "history":
		r.printHistory()
		return false

	case "save":
		_ = r.saveSession()
		return false

	case "session info":
		r.printSessionInfo()
		return false

	case "stats":
		r.printStats()
		return false

	case "export":
		md := conversation.ExportMarkdown(r.conv)
		filename := fmt.Sprintf("%s_export.md", r.conv.SessionID)
		if err := os.WriteFile(filename, []byte(md), 0600); err != nil {
			_, _ = fmt.Fprintf(r.writer, "Failed to export: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(r.writer, "Successfully exported conversation to %s\n", filename)
		}
		return false

	default:
		// Regular message - send to LLM
		return r.sendMessage(input)
	}
}

// sendMessage sends a user message to the LLM and appends the response to conversation.
func (r *REPL) sendMessage(userMessage string) bool {
	// Apply context pruning before sending
	var summary string
	r.conv.Messages, summary = conversation.PruneHistory(r.conv.Messages, r.maxContext)
	if summary != "" {
		_, _ = fmt.Fprintf(r.writer, "[Context] Pruned older messages to stay under %d tokens.\n", r.maxContext)
	}

	history := r.conv.GetHistory()
	if summary != "" {
		// Prepend summary as a system message so the LLM retains old context
		history = append([]llm.ChatMessage{{Role: "system", Content: summary}}, history...)
	}

	// Use SendMessageWithDoc to avoid re-sending the document with each follow-up
	_, _ = fmt.Fprintln(r.writer, "")
	response, stats, err := r.client.SendMessageWithDoc(history, userMessage, r.client.DocumentText, func(token string) {
		_, _ = fmt.Fprint(r.writer, token)
	})
	if err != nil {
		_, _ = fmt.Fprintf(r.writer, "Error: %v\n", err)
		return false
	}

	// Add both messages to conversation history
	r.conv.AddMessage("user", userMessage)
	r.conv.AddMessage("assistant", response)

	// Accumulate stats
	r.stats.Add(stats)

	// Display token stats (newline provided by writer)
	_, _ = fmt.Fprintln(r.writer, "\n\n"+llm.FormatTokenStats(stats))

	// Generate speech if TTS is enabled and response is not empty
	if r.ttsEngine != nil && strings.TrimSpace(tts.CleanMarkdown(response)) != "" {
		voiceFile := filepath.Join("voice", fmt.Sprintf("%s_%d.wav", r.conv.SessionID, len(r.conv.Messages)/2))
		_, _ = fmt.Fprintf(r.writer, "[TTS] Generating speech: %s... ", voiceFile)

		data, err := r.ttsEngine.Synthesize(context.Background(), response, r.isSSML)
		if err != nil {
			_, _ = fmt.Fprintf(r.writer, "Error: %v\n", err)
		} else {
			dir := filepath.Dir(voiceFile)
			if err := os.MkdirAll(dir, 0750); err != nil {
				_, _ = fmt.Fprintf(r.writer, "Error creating directory: %v\n", err)
			} else if err := os.WriteFile(voiceFile, data, 0600); err != nil {
				_, _ = fmt.Fprintf(r.writer, "Error saving audio: %v\n", err)
			} else {
				_, _ = fmt.Fprintln(r.writer, "Done.")
			}
		}
	}

	return false
}

// printHistory displays all messages in the conversation.
func (r *REPL) printHistory() {
	history := r.conv.GetHistory()

	if len(history) == 0 {
		_, _ = fmt.Fprintln(r.writer, "No messages yet.")
		return
	}

	_, _ = fmt.Fprintln(r.writer, "")
	_, _ = fmt.Fprintln(r.writer, "=== Conversation History ===")
	for i, msg := range history {
		_, _ = fmt.Fprintf(r.writer, "\n[%d] %s:\n", i+1, strings.ToUpper(msg.Role))
		_, _ = fmt.Fprintln(r.writer, msg.Content)
	}
	_, _ = fmt.Fprintln(r.writer, "\n=== End of History ===")
}

// saveSession saves the current conversation to disk.
func (r *REPL) saveSession() error {
	if err := conversation.SaveSession(r.conv, r.sessionDir); err != nil {
		_, _ = fmt.Fprintf(r.writer, "Error saving session: %v\n", err)
		return err
	}
	_, _ = fmt.Fprintf(r.writer, "[Session] Saved to '%s'\n", r.conv.SessionID)
	return nil
}

// printSessionInfo displays metadata about the current session.
func (r *REPL) printSessionInfo() {
	_, _ = fmt.Fprintln(r.writer, "")
	_, _ = fmt.Fprintln(r.writer, "=== Session Info ===")
	_, _ = fmt.Fprintf(r.writer, "Session ID:    %s\n", r.conv.SessionID)
	_, _ = fmt.Fprintf(r.writer, "File:          %s\n", r.conv.FileName)
	_, _ = fmt.Fprintf(r.writer, "Created:       %s\n", r.conv.CreatedAt.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(r.writer, "Last Updated:  %s\n", r.conv.LastUpdated.Format("2006-01-02 15:04:05"))
	_, _ = fmt.Fprintf(r.writer, "Messages:      %d\n", len(r.conv.Messages))
	_, _ = fmt.Fprintf(r.writer, "Doc Size:      %d bytes\n", len(r.conv.DocumentText))
	_, _ = fmt.Fprintln(r.writer, "=== End Info ===")
}

// printStats displays cumulative token usage for the session.
func (r *REPL) printStats() {
	_, _ = fmt.Fprintln(r.writer, "")
	_, _ = fmt.Fprintln(r.writer, "=== Session Token Stats ===")
	_, _ = fmt.Fprintf(r.writer, "Total Inputs:    %d tokens\n", r.stats.InputTokens)
	_, _ = fmt.Fprintf(r.writer, "Total Outputs:   %d tokens\n", r.stats.OutputTokens)
	_, _ = fmt.Fprintf(r.writer, "Total Usage:     %d tokens\n", r.stats.TotalTokens)
	_, _ = fmt.Fprintln(r.writer, "=== End Stats ===")
}
