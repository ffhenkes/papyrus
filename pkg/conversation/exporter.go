package conversation

import (
	"fmt"
	"strings"
	"time"
)

// ExportMarkdown formats the entire conversation history into a clean Markdown document.
func ExportMarkdown(conv *Conversation) string {
	if conv == nil {
		return ""
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "# Papyrus Session Export: %s\n\n", conv.SessionID)
	fmt.Fprintf(&sb, "**Document**: `%s`\n", conv.FileName)
	fmt.Fprintf(&sb, "**Date**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("---\n\n")

	for _, msg := range conv.Messages {
		role := "\U0001F464 User" // User
		switch msg.Role {
		case "assistant":
			role = "\U0001F916 Assistant" // Assistant
		case "system":
			role = "\u2699\uFE0F System" // System
		}

		fmt.Fprintf(&sb, "### %s\n\n", role)
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}
