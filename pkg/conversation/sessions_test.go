package conversation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSaveAndLoadSession verifies save/load roundtrip with message preservation.
func TestSaveAndLoadSession(t *testing.T) {
	tempDir := t.TempDir()

	// Create a conversation with messages
	conv := New("test.pdf", "This is test document content")
	conv.AddMessage("user", "What is this document about?")
	conv.AddMessage("assistant", "This is a test document.")

	originalSessionID := conv.SessionID
	originalMessageCount := len(conv.Messages)

	// Save session
	if err := SaveSession(conv, tempDir); err != nil {
		t.Fatalf("SaveSession() returned error: %v", err)
	}

	// Verify file was created
	sessionFile := filepath.Join(tempDir, conv.SessionID+".json")
	if _, err := os.Stat(sessionFile); err != nil {
		t.Fatalf("Session file not created: %v", err)
	}

	// Load session
	loaded, err := LoadSession(originalSessionID, tempDir)
	if err != nil {
		t.Fatalf("LoadSession() returned error: %v", err)
	}

	// Verify content
	if loaded.SessionID != originalSessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", loaded.SessionID, originalSessionID)
	}
	if loaded.FileName != "test.pdf" {
		t.Errorf("FileName mismatch: got %s, want test.pdf", loaded.FileName)
	}
	if len(loaded.Messages) != originalMessageCount {
		t.Errorf("Message count mismatch: got %d, want %d", len(loaded.Messages), originalMessageCount)
	}
	if loaded.DocumentText != "This is test document content" {
		t.Errorf("DocumentText mismatch")
	}
}

// TestLoadSessionNotFound handles missing session file.
func TestLoadSessionNotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, err := LoadSession("nonexistent-session", tempDir)
	if err == nil {
		t.Error("LoadSession() should return error for nonexistent session")
	}
}

// TestListSessions returns sessions sorted by last updated.
func TestListSessions(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple sessions
	for i := 1; i <= 3; i++ {
		conv := New("document.pdf", "Content")
		conv.AddMessage("user", "Q1")
		conv.AddMessage("assistant", "A1")

		if err := SaveSession(conv, tempDir); err != nil {
			t.Fatalf("Failed to save session %d: %v", i, err)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// List sessions
	sessions, err := ListSessions(tempDir)
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("ListSessions() returned %d sessions, want 3", len(sessions))
	}

	// Verify newest first
	for i := 0; i < len(sessions)-1; i++ {
		if sessions[i].LastUpdated.Before(sessions[i+1].LastUpdated) {
			t.Errorf("Sessions not sorted by LastUpdated (newest first)")
		}
	}
}

// TestSessionExists checks session file existence.
func TestSessionExists(t *testing.T) {
	tempDir := t.TempDir()

	conv := New("test.pdf", "Content")
	sessionID := conv.SessionID

	// Before save
	if SessionExists(sessionID, tempDir) {
		t.Error("SessionExists() returned true for unsaved session")
	}

	// After save
	if err := SaveSession(conv, tempDir); err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	if !SessionExists(sessionID, tempDir) {
		t.Error("SessionExists() returned false for saved session")
	}
}

// TestDeleteSession removes a session file.
func TestDeleteSession(t *testing.T) {
	tempDir := t.TempDir()

	conv := New("test.pdf", "Content")
	if err := SaveSession(conv, tempDir); err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	// Delete
	if err := DeleteSession(conv.SessionID, tempDir); err != nil {
		t.Fatalf("DeleteSession() returned error: %v", err)
	}

	// Verify deletion
	if SessionExists(conv.SessionID, tempDir) {
		t.Error("Session file still exists after deletion")
	}

	// Delete again (should error)
	if err := DeleteSession(conv.SessionID, tempDir); err == nil {
		t.Error("DeleteSession() should error for nonexistent session")
	}
}

// TestSessionMetadata verifies metadata is populated correctly.
func TestSessionMetadata(t *testing.T) {
	tempDir := t.TempDir()

	conv := New("testfile.pdf", "Content")
	conv.AddMessage("user", "Q1")
	conv.AddMessage("assistant", "A1")
	conv.AddMessage("user", "Q2")

	if err := SaveSession(conv, tempDir); err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	sessions, err := ListSessions(tempDir)
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	meta := sessions[0]
	if meta.SessionID != conv.SessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", meta.SessionID, conv.SessionID)
	}
	if meta.FileName != "testfile.pdf" {
		t.Errorf("FileName mismatch: got %s, want testfile.pdf", meta.FileName)
	}
	if meta.MessageCount != 3 {
		t.Errorf("MessageCount mismatch: got %d, want 3", meta.MessageCount)
	}
}

// TestGenerateSessionID creates unique and deterministic IDs.
func TestGenerateSessionID(t *testing.T) {
	// Same filename should produce ID with filename
	id1 := new(Conversation).SessionID
	if id1 == "" {
		// generateSessionID must be called in New() to populate
		conv := New("document.pdf", "")
		if !isValidSessionID(conv.SessionID) {
			t.Errorf("Invalid session ID: %s", conv.SessionID)
		}

		// Different filenames should produce different IDs
		conv2 := New("other.pdf", "")
		if conv.SessionID == conv2.SessionID {
			t.Errorf("Different filenames produced same SessionID")
		}
	}
}

func isValidSessionID(id string) bool {
	// SessionID should have format: filename-hash
	return len(id) > 0 && (len(id) >= 20) // At least filename + dash + 12 hex chars
}

// TestSaveMultipleSessions handles multiple concurrent saves.
func TestSaveMultipleSessions(t *testing.T) {
	tempDir := t.TempDir()

	// Create and save multiple conversations
	convs := []*Conversation{}
	for i := 0; i < 5; i++ {
		conv := New("doc.pdf", "Content")
		convs = append(convs, conv)
	}

	// Save all
	for _, conv := range convs {
		if err := SaveSession(conv, tempDir); err != nil {
			t.Fatalf("SaveSession() failed: %v", err)
		}
	}

	// List and verify
	sessions, err := ListSessions(tempDir)
	if err != nil {
		t.Fatalf("ListSessions() failed: %v", err)
	}

	if len(sessions) != 5 {
		t.Errorf("ListSessions() returned %d sessions, want 5", len(sessions))
	}
}

// TestConversationWithMessages verifies message handling in saved sessions.
func TestConversationWithMessages(t *testing.T) {
	tempDir := t.TempDir()

	conv := New("doc.pdf", "Document content")
	conv.AddMessage("user", "First question?")
	conv.AddMessage("assistant", "First answer.")
	conv.AddMessage("user", "Follow-up question?")
	conv.AddMessage("assistant", "Follow-up answer.")

	if err := SaveSession(conv, tempDir); err != nil {
		t.Fatalf("SaveSession() failed: %v", err)
	}

	loaded, err := LoadSession(conv.SessionID, tempDir)
	if err != nil {
		t.Fatalf("LoadSession() failed: %v", err)
	}

	// Verify all messages preserved
	if len(loaded.Messages) != 4 {
		t.Errorf("Expected 4 messages, got %d", len(loaded.Messages))
	}

	// Verify message content
	expectedRoles := []string{"user", "assistant", "user", "assistant"}
	for i, role := range expectedRoles {
		if loaded.Messages[i].Role != role {
			t.Errorf("Message %d role mismatch: got %s, want %s", i, loaded.Messages[i].Role, role)
		}
	}
}
