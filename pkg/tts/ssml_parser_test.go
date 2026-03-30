package tts

import (
	"strings"
	"testing"
)

func TestParseSSML_PlainText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "simple text",
			input:   "Hello world",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "text with whitespace",
			input:   "  Hello world  ",
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elements, err := ParseSSML(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(elements) != tt.wantLen {
				t.Errorf("ParseSSML() len = %d, want %d", len(elements), tt.wantLen)
				return
			}

			if len(elements) > 0 {
				if seg, ok := elements[0].(*SSMLSegment); !ok {
					t.Errorf("ParseSSML() expected SSMLSegment, got %T", elements[0])
				} else {
					if tt.input != "" && seg.Text != "Hello world" {
						t.Errorf("ParseSSML() text = %q, want %q", seg.Text, "Hello world")
					}
				}
			}
		})
	}
}

func TestParseSSML_BasicSSML(t *testing.T) {
	input := `<speak>Hello world</speak>`
	elements, err := ParseSSML(input)

	if err != nil {
		t.Fatalf("ParseSSML() unexpected error: %v", err)
	}

	if len(elements) != 1 {
		t.Fatalf("ParseSSML() len = %d, want 1", len(elements))
	}

	seg, ok := elements[0].(*SSMLSegment)
	if !ok {
		t.Fatalf("ParseSSML() expected SSMLSegment, got %T", elements[0])
	}

	if seg.Text != "Hello world" {
		t.Errorf("ParseSSML() text = %q, want %q", seg.Text, "Hello world")
	}
}

func TestParseSSML_BreakTag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantDur int
		wantErr bool
	}{
		{
			name:    "break with milliseconds",
			input:   `<speak>Hello<break time="500ms"/>world</speak>`,
			wantLen: 3,
			wantDur: 500,
			wantErr: false,
		},
		{
			name:    "break with seconds",
			input:   `<speak>Hello<break time="1s"/>world</speak>`,
			wantLen: 3,
			wantDur: 1000,
			wantErr: false,
		},
		{
			name:    "multiple breaks",
			input:   `<speak>A<break time="100ms"/>B<break time="200ms"/>C</speak>`,
			wantLen: 5,
			wantDur: 100, // first break should be 100ms
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elements, err := ParseSSML(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(elements) != tt.wantLen {
				t.Errorf("ParseSSML() len = %d, want %d", len(elements), tt.wantLen)
				return
			}

			// Find first break
			for _, elem := range elements {
				if brk, ok := elem.(*SSMLBreak); ok {
					if brk.DurationMs != tt.wantDur {
						t.Errorf("ParseSSML() break duration = %d, want %d", brk.DurationMs, tt.wantDur)
					}
					break
				}
			}
		})
	}
}

func TestParseSSML_VoiceTag(t *testing.T) {
	input := `<speak>Default <voice name="pt_BR-faber-medium">Portuguese voice</voice> back to default</speak>`
	elements, err := ParseSSML(input)

	if err != nil {
		t.Fatalf("ParseSSML() unexpected error: %v", err)
	}

	if len(elements) != 3 {
		t.Fatalf("ParseSSML() len = %d, want 3", len(elements))
	}

	// Check first segment (default voice)
	seg1, ok := elements[0].(*SSMLSegment)
	if !ok {
		t.Fatalf("elements[0] expected SSMLSegment, got %T", elements[0])
	}
	if seg1.Voice != "" {
		t.Errorf("elements[0].Voice = %q, want empty", seg1.Voice)
	}

	// Check second segment (Portuguese voice)
	seg2, ok := elements[1].(*SSMLSegment)
	if !ok {
		t.Fatalf("elements[1] expected SSMLSegment, got %T", elements[1])
	}
	if seg2.Voice != "pt_BR-faber-medium" {
		t.Errorf("elements[1].Voice = %q, want %q", seg2.Voice, "pt_BR-faber-medium")
	}

	// Check third segment (default voice again)
	seg3, ok := elements[2].(*SSMLSegment)
	if !ok {
		t.Fatalf("elements[2] expected SSMLSegment, got %T", elements[2])
	}
	if seg3.Voice != "" {
		t.Errorf("elements[2].Voice = %q, want empty", seg3.Voice)
	}
}

func TestParseSSML_ProsodyTag(t *testing.T) {
	input := `<speak>Normal <prosody rate="1.5" pitch="high" volume="loud">Fast high loud</prosody> normal</speak>`
	elements, err := ParseSSML(input)

	if err != nil {
		t.Fatalf("ParseSSML() unexpected error: %v", err)
	}

	// Find segment with prosody
	var prosodySegment *SSMLSegment
	for _, elem := range elements {
		if seg, ok := elem.(*SSMLSegment); ok && seg.Prosody != nil {
			prosodySegment = seg
			break
		}
	}

	if prosodySegment == nil {
		t.Fatal("ParseSSML() no segment with prosody found")
	}

	if prosodySegment.Prosody.Rate != "1.5" {
		t.Errorf("Prosody.Rate = %q, want %q", prosodySegment.Prosody.Rate, "1.5")
	}
	if prosodySegment.Prosody.Pitch != "high" {
		t.Errorf("Prosody.Pitch = %q, want %q", prosodySegment.Prosody.Pitch, "high")
	}
	if prosodySegment.Prosody.Volume != "loud" {
		t.Errorf("Prosody.Volume = %q, want %q", prosodySegment.Prosody.Volume, "loud")
	}
}

func TestParseSSML_ComplexNesting(t *testing.T) {
	input := `<speak>
		Hello 
		<voice name="voice1">
			First voice 
			<break time="300ms"/>
			still first voice
		</voice>
		Back to default
		<voice name="voice2">
			Second voice
			<prosody rate="1.2">
				Fast second voice
			</prosody>
		</voice>
	</speak>`

	elements, err := ParseSSML(input)

	if err != nil {
		t.Fatalf("ParseSSML() unexpected error: %v", err)
	}

	if len(elements) == 0 {
		t.Fatal("ParseSSML() returned empty elements")
	}

	// Verify we have breaks
	hasBreak := false
	for _, elem := range elements {
		if _, ok := elem.(*SSMLBreak); ok {
			hasBreak = true
			break
		}
	}

	if !hasBreak {
		t.Error("ParseSSML() no breaks found in complex SSML")
	}

	// Verify we have voice tags
	voiceCount := 0
	for _, elem := range elements {
		if seg, ok := elem.(*SSMLSegment); ok && seg.Voice != "" {
			voiceCount++
		}
	}

	if voiceCount == 0 {
		t.Error("ParseSSML() no voice-tagged segments found")
	}
}

func TestParseSSML_SentenceTag(t *testing.T) {
	input := `<speak><s>First sentence.</s> <s>Second sentence.</s></speak>`
	elements, err := ParseSSML(input)

	if err != nil {
		t.Fatalf("ParseSSML() unexpected error: %v", err)
	}

	if len(elements) < 2 {
		t.Errorf("ParseSSML() len = %d, want >= 2", len(elements))
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"500ms", 500},
		{"100ms", 100},
		{"1s", 1000},
		{"1.5s", 1500},
		{"0.5s", 500},
		{"2s", 2000},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractAttribute(t *testing.T) {
	tests := []struct {
		tag      string
		attrName string
		expected string
	}{
		{`<voice name="pt_BR-faber-medium">`, "name", "pt_BR-faber-medium"},
		{`<break time="500ms"/>`, "time", "500ms"},
		{`<prosody rate="1.5" pitch="high">`, "rate", "1.5"},
		{`<prosody rate="1.5" pitch="high">`, "pitch", "high"},
		{`<tag attr="value" other="foo">`, "other", "foo"},
		{`<tag missing="value">`, "notfound", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := extractAttribute(tt.tag, tt.attrName)
			if result != tt.expected {
				t.Errorf("extractAttribute(%q, %q) = %q, want %q", tt.tag, tt.attrName, result, tt.expected)
			}
		})
	}
}

func TestConvertElementsToString(t *testing.T) {
	elements := []SSMLElement{
		&SSMLSegment{Text: "Hello", Voice: ""},
		&SSMLBreak{DurationMs: 500},
		&SSMLSegment{Text: "world", Voice: "pt_BR-faber-medium"},
	}

	result := ConvertElementsToString(elements)

	// Just verify it returns a non-empty string with expected parts
	if len(result) == 0 {
		t.Error("ConvertElementsToString() returned empty string")
	}

	if !strings.Contains(result, "Hello") {
		t.Error("ConvertElementsToString() missing 'Hello'")
	}

	if !strings.Contains(result, "BREAK") {
		t.Error("ConvertElementsToString() missing 'BREAK'")
	}

	if !strings.Contains(result, "world") {
		t.Error("ConvertElementsToString() missing 'world'")
	}
}
