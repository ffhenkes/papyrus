package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// SessionMetadata describes a saved session for listing/resuming.
type SessionMetadata struct {
	SessionID    string    `json:"session_id"`
	FileName     string    `json:"file_name"`
	CreatedAt    time.Time `json:"created_at"`
	LastUpdated  time.Time `json:"last_updated"`
	MessageCount int       `json:"message_count"`
}

// SaveSession persists a conversation to disk as JSON.
// Creates the session directory if it doesn't exist.
func SaveSession(conv *Conversation, sessionDir string) error {
	// Create sessions directory if it doesn't exist
	if err := os.MkdirAll(sessionDir, 0o750); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Update last modified timestamp
	conv.LastUpdated = time.Now()

	// Marshal conversation to JSON
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// Write to file: sessionDir/sessionID.json
	sessionFile := filepath.Join(sessionDir, conv.SessionID+".json")
	if err := os.WriteFile(sessionFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadSession restores a conversation from disk by session ID.
// Returns an error if the session file doesn't exist.
func LoadSession(sessionID, sessionDir string) (*Conversation, error) {
	sessionFile := filepath.Join(sessionDir, sessionID+".json")

	// Read session file
	// #nosec G304 - sessionFile is constructed from sessionID parameter which is user-provided but safe
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session '%s' not found", sessionID)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	// Unmarshal JSON to conversation
	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &conv, nil
}

// ListSessions returns all saved sessions sorted by last updated (newest first).
func ListSessions(sessionDir string) ([]SessionMetadata, error) {
	// Create directory if it doesn't exist (for first-time use)
	if err := os.MkdirAll(sessionDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to access session directory: %w", err)
	}

	// Read session directory
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []SessionMetadata

	for _, entry := range entries {
		// Skip directories and non-JSON files
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Load session to get metadata
		sessionID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		conv, err := LoadSession(sessionID, sessionDir)
		if err != nil {
			// Skip corrupted sessions, log and continue
			fmt.Fprintf(os.Stderr, "Warning: skipping corrupted session %s: %v\n", sessionID, err)
			continue
		}

		sessions = append(sessions, SessionMetadata{
			SessionID:    conv.SessionID,
			FileName:     conv.FileName,
			CreatedAt:    conv.CreatedAt,
			LastUpdated:  conv.LastUpdated,
			MessageCount: len(conv.Messages),
		})
	}

	// Sort by last updated (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastUpdated.After(sessions[j].LastUpdated)
	})

	return sessions, nil
}

// DeleteSession removes a session file.
func DeleteSession(sessionID, sessionDir string) error {
	sessionFile := filepath.Join(sessionDir, sessionID+".json")
	if err := os.Remove(sessionFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session '%s' not found", sessionID)
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// SessionExists checks if a session file exists.
func SessionExists(sessionID, sessionDir string) bool {
	sessionFile := filepath.Join(sessionDir, sessionID+".json")
	_, err := os.Stat(sessionFile)
	return err == nil
}
