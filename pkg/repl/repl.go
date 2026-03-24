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
	client *llm.Client
	conv   *conversation.Conversation
	reader io.Reader
	writer io.Writer
	done   chan bool
}

// New creates a new REPL session.
func New(client *llm.Client, conv *conversation.Conversation) *REPL {
	return &REPL{
		client: client,
		conv:   conv,
		reader: os.Stdin,
		writer: os.Stdout,
		done:   make(chan bool),
	}
}

// Start begins the interactive REPL loop.
func (r *REPL) Start() error {
	scanner := bufio.NewScanner(r.reader)

	_, _ = fmt.Fprintln(r.writer, strings.Repeat("─", 60))
	_, _ = fmt.Fprintln(r.writer, "Interactive mode. Type 'quit' or 'exit' to end, 'history' to see all messages.")
	_, _ = fmt.Fprintln(r.writer, strings.Repeat("─", 60))

	for {
		_, _ = fmt.Fprint(r.writer, "\n> ")
		if !scanner.Scan() {
			// EOF reached
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

// handleCommand processes built-in commands like quit, exit, and history.
// Returns true if the REPL should exit.
func (r *REPL) handleCommand(input string) bool {
	switch strings.ToLower(input) {
	case "quit", "exit":
		_, _ = fmt.Fprintln(r.writer, "Goodbye!")
		return true

	case "history":
		r.printHistory()
		return false

	default:
		// Regular message - send to LLM
		return r.sendMessage(input)
	}
}

// sendMessage sends a user message to the LLM and appends the response to conversation.
func (r *REPL) sendMessage(userMessage string) bool {
	response, err := r.client.SendMessage(r.conv.GetHistory(), userMessage)
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
