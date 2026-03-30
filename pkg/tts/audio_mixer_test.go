package tts

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// createTestWAV creates a simple test WAV file with the given PCM data
func createTestWAV(format *WAVFormat, pcmData []byte) []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	riffSize := 36 + len(pcmData)
	buf.WriteString("RIFF")
	// #nosec G115 - riffSize is safe for uint32
	_ = binary.Write(buf, binary.LittleEndian, uint32(riffSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, format.AudioFormat)
	_ = binary.Write(buf, binary.LittleEndian, format.NumChannels)
	_ = binary.Write(buf, binary.LittleEndian, format.SampleRate)
	_ = binary.Write(buf, binary.LittleEndian, format.ByteRate)
	_ = binary.Write(buf, binary.LittleEndian, format.BlockAlign)
	_ = binary.Write(buf, binary.LittleEndian, format.BitsPerSample)

	// data chunk
	buf.WriteString("data")
	// #nosec G115 - pcmData length is safe for uint32
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(pcmData)))
	buf.Write(pcmData)

	return buf.Bytes()
}

func TestExtractWAVFormat(t *testing.T) {
	format := &WAVFormat{
		AudioFormat:   1, // PCM
		NumChannels:   1, // Mono
		SampleRate:    44100,
		ByteRate:      88200,
		BlockAlign:    2,
		BitsPerSample: 16,
	}

	// Create test PCM data (440 Hz sine wave, 1 sample)
	pcm := make([]byte, 2)
	binary.LittleEndian.PutUint16(pcm, 0x7FFF)

	wavData := createTestWAV(format, pcm)

	// Extract format
	extracted, err := ExtractWAVFormat(wavData)
	if err != nil {
		t.Fatalf("ExtractWAVFormat() error = %v", err)
	}

	if extracted.AudioFormat != format.AudioFormat {
		t.Errorf("AudioFormat = %d, want %d", extracted.AudioFormat, format.AudioFormat)
	}
	if extracted.NumChannels != format.NumChannels {
		t.Errorf("NumChannels = %d, want %d", extracted.NumChannels, format.NumChannels)
	}
	if extracted.SampleRate != format.SampleRate {
		t.Errorf("SampleRate = %d, want %d", extracted.SampleRate, format.SampleRate)
	}
	if extracted.BitsPerSample != format.BitsPerSample {
		t.Errorf("BitsPerSample = %d, want %d", extracted.BitsPerSample, format.BitsPerSample)
	}
}

func TestExtractPCMData(t *testing.T) {
	format := &WAVFormat{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    44100,
		ByteRate:      88200,
		BlockAlign:    2,
		BitsPerSample: 16,
	}

	// Create predictable test data
	originalPCM := []byte{0xFF, 0x7F, 0x00, 0x80} // Two 16-bit samples

	wavData := createTestWAV(format, originalPCM)

	// Extract PCM
	extracted, err := ExtractPCMData(wavData)
	if err != nil {
		t.Fatalf("ExtractPCMData() error = %v", err)
	}

	if !bytes.Equal(extracted, originalPCM) {
		t.Errorf("extracted PCM = %v, want %v", extracted, originalPCM)
	}
}

func TestGenerateSilence(t *testing.T) {
	tests := []struct {
		name          string
		numChannels   uint16
		sampleRate    uint32
		durationMs    int
		bitsPerSample uint16
		expectedBytes int64
	}{
		{
			name:          "Mono 44.1kHz 16-bit 100ms",
			numChannels:   1,
			sampleRate:    44100,
			durationMs:    100,
			bitsPerSample: 16,
			expectedBytes: 8820, // 44100 * 100 / 1000 * 1 * 2
		},
		{
			name:          "Stereo 44.1kHz 16-bit 100ms",
			numChannels:   2,
			sampleRate:    44100,
			durationMs:    100,
			bitsPerSample: 16,
			expectedBytes: 17640, // 44100 * 100 / 1000 * 2 * 2
		},
		{
			name:          "Mono 44.1kHz 16-bit 1s",
			numChannels:   1,
			sampleRate:    44100,
			durationMs:    1000,
			bitsPerSample: 16,
			expectedBytes: 88200, // 44100 * 1000 / 1000 * 1 * 2
		},
		{
			name:          "Mono 48kHz 16-bit 500ms",
			numChannels:   1,
			sampleRate:    48000,
			durationMs:    500,
			bitsPerSample: 16,
			expectedBytes: 48000, // 48000 * 500 / 1000 * 1 * 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			silence := GenerateSilence(tt.numChannels, tt.sampleRate, tt.durationMs, tt.bitsPerSample)

			if int64(len(silence)) != tt.expectedBytes {
				t.Errorf("GenerateSilence() len = %d, want %d", len(silence), tt.expectedBytes)
			}

			// Verify it's all zeros
			for i, b := range silence {
				if b != 0 {
					t.Errorf("Byte %d is %v, want 0", i, b)
					break
				}
			}
		})
	}
}

func TestConcatenatePCM(t *testing.T) {
	seg1 := []byte{0x01, 0x02}
	seg2 := []byte{0x03, 0x04}
	seg3 := []byte{0x05, 0x06}

	result := ConcatenatePCM([][]byte{seg1, seg2, seg3})

	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	if !bytes.Equal(result, expected) {
		t.Errorf("ConcatenatePCM() = %v, want %v", result, expected)
	}
}

func TestConcatenateWAV(t *testing.T) {
	format := &WAVFormat{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    44100,
		ByteRate:      88200,
		BlockAlign:    2,
		BitsPerSample: 16,
	}

	// Create three test WAV files with different PCM data
	pcm1 := []byte{0x01, 0x02}
	pcm2 := []byte{0x03, 0x04}
	pcm3 := []byte{0x05, 0x06}

	wav1 := createTestWAV(format, pcm1)
	wav2 := createTestWAV(format, pcm2)
	wav3 := createTestWAV(format, pcm3)

	// Concatenate
	result, err := ConcatenateWAV([][]byte{wav1, wav2, wav3})
	if err != nil {
		t.Fatalf("ConcatenateWAV() error = %v", err)
	}

	// Extract PCM from result
	extractedPCM, err := ExtractPCMData(result)
	if err != nil {
		t.Fatalf("ExtractPCMData() error = %v", err)
	}

	expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	if !bytes.Equal(extractedPCM, expected) {
		t.Errorf("ConcatenateWAV() PCM = %v, want %v", extractedPCM, expected)
	}

	// Verify format is preserved
	extractedFormat, err := ExtractWAVFormat(result)
	if err != nil {
		t.Fatalf("ExtractWAVFormat() error = %v", err)
	}

	if extractedFormat.NumChannels != format.NumChannels {
		t.Errorf("NumChannels = %d, want %d", extractedFormat.NumChannels, format.NumChannels)
	}
	if extractedFormat.SampleRate != format.SampleRate {
		t.Errorf("SampleRate = %d, want %d", extractedFormat.SampleRate, format.SampleRate)
	}
}

func TestCreateWAVFile(t *testing.T) {
	format := &WAVFormat{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    44100,
		ByteRate:      88200,
		BlockAlign:    2,
		BitsPerSample: 16,
	}

	pcmData := []byte{0x01, 0x02, 0x03, 0x04}

	wavData := CreateWAVFile(format, pcmData)

	// Verify it's a valid WAV file
	if string(wavData[0:4]) != "RIFF" {
		t.Errorf("RIFF header invalid")
	}

	if string(wavData[8:12]) != "WAVE" {
		t.Errorf("WAVE header invalid")
	}

	// Extract and verify PCM
	extracted, err := ExtractPCMData(wavData)
	if err != nil {
		t.Fatalf("ExtractPCMData() error = %v", err)
	}

	if !bytes.Equal(extracted, pcmData) {
		t.Errorf("extracted PCM = %v, want %v", extracted, pcmData)
	}
}

func TestCreateWAVFileWithStereo(t *testing.T) {
	format := &WAVFormat{
		AudioFormat:   1,
		NumChannels:   2,
		SampleRate:    48000,
		ByteRate:      192000,
		BlockAlign:    4,
		BitsPerSample: 16,
	}

	pcmData := make([]byte, 100)
	wavData := CreateWAVFile(format, pcmData)

	// Extract and verify format
	extractedFormat, err := ExtractWAVFormat(wavData)
	if err != nil {
		t.Fatalf("ExtractWAVFormat() error = %v", err)
	}

	if extractedFormat.NumChannels != 2 {
		t.Errorf("NumChannels = %d, want 2", extractedFormat.NumChannels)
	}

	if extractedFormat.SampleRate != 48000 {
		t.Errorf("SampleRate = %d, want 48000", extractedFormat.SampleRate)
	}
}
