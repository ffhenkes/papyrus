package pdf

import "testing"

// TestExtractTextFileNotFound tests extraction with missing file
func TestExtractTextFileNotFound(t *testing.T) {
	_, err := ExtractText("nonexistent_file.pdf")
	if err == nil {
		t.Error("ExtractText() should return error for nonexistent file")
	}
}
