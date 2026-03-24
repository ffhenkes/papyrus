package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"papyrus/pkg/conversation"
	"papyrus/pkg/llm"
)

// REPL represents a read-eval-print loop for interactive conversations.
type REPL struct {
	client     *llm.Client
	conv       *conversation.Conversation
	reader     io.Reader
	writer     io.Writer
	done       chan bool
	sessionDir string
}

// New creates a new REPL session.
func New(client *llm.Client, conv *conversation.Conversation, sessionDir string) *REPL {
	return &REPL{
		client:     client,
		conv:       conv,
		reader:     os.Stdin,
		writer:     os.Stdout,
		done:       make(chan bool),
		sessionDir: sessionDir,
	}
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

	default:
		// Regular message - send to LLM
		return r.sendMessage(input)
	}
}

// sendMessage sends a user message to the LLM and appends the response to conversation.
func (r *REPL) sendMessage(userMessage string) bool {
	// Use SendMessageWithDoc to avoid re-sending the document with each follow-up
	response, err := r.client.SendMessageWithDoc(r.conv.GetHistory(), userMessage, r.client.DocumentText)
	if err != nil {
		_, _ = fmt.Fprintf(r.writer, "Error: %v\n", err)
		return false
	}

	// Add both messages to conversation history
	r.conv.AddMessage("user", userMessage)
	r.conv.AddMessage("assistant", response)

	// Display response
	_, _ = fmt.Fprintln(r.writer, "")
	_, _ = fmt.Fprintln(r.writer, response)

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
