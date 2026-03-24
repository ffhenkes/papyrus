package pdf

import (
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText extracts text content from a PDF file.
func ExtractText(pdfPath string) (string, error) {
	file, content, err := pdf.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("could not open PDF file '%s': make sure the file exists and is mapped to the /pdfs directory (error: %w)", pdfPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
		}
	}()

	var sb strings.Builder
	for i := 1; i <= content.NumPage(); i++ {
		page := content.Page(i)
		text, err := page.GetPlainText(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not extract text from page %d: %v\n", i, err)
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		fmt.Fprintf(&sb, "\n--- Page %d ---\n", i)
		sb.WriteString(text)
	}
	return sb.String(), nil
}
