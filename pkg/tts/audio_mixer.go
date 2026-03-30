package tts

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// WAVFormat represents the format of a WAV file
type WAVFormat struct {
	AudioFormat   uint16 // 1 for PCM
	NumChannels   uint16 // 1 for mono, 2 for stereo
	SampleRate    uint32 // e.g., 44100
	ByteRate      uint32 // SampleRate * NumChannels * BitsPerSample / 8
	BlockAlign    uint16 // NumChannels * BitsPerSample / 8
	BitsPerSample uint16 // 16 for 16-bit
}

// ExtractWAVFormat extracts format information from a WAV file
func ExtractWAVFormat(wavData []byte) (*WAVFormat, error) {
	if len(wavData) < 44 {
		return nil, fmt.Errorf("WAV file too small: %d bytes", len(wavData))
	}

	// Check RIFF header
	if string(wavData[0:4]) != "RIFF" {
		return nil, fmt.Errorf("invalid RIFF header")
	}

	// Check fmt chunk
	if string(wavData[8:12]) != "WAVE" {
		return nil, fmt.Errorf("invalid WAVE header")
	}

	// Find fmt chunk
	pos := 12
	for pos < len(wavData)-8 {
		chunkID := string(wavData[pos : pos+4])
		chunkSize := binary.LittleEndian.Uint32(wavData[pos+4 : pos+8])

		if chunkID == "fmt " {
			if pos+8+24 > len(wavData) {
				return nil, fmt.Errorf("fmt chunk too small")
			}

			format := &WAVFormat{
				AudioFormat:   binary.LittleEndian.Uint16(wavData[pos+8 : pos+10]),
				NumChannels:   binary.LittleEndian.Uint16(wavData[pos+10 : pos+12]),
				SampleRate:    binary.LittleEndian.Uint32(wavData[pos+12 : pos+16]),
				ByteRate:      binary.LittleEndian.Uint32(wavData[pos+16 : pos+20]),
				BlockAlign:    binary.LittleEndian.Uint16(wavData[pos+20 : pos+22]),
				BitsPerSample: binary.LittleEndian.Uint16(wavData[pos+22 : pos+24]),
			}
			return format, nil
		}

		pos += 8 + int(chunkSize)
	}

	return nil, fmt.Errorf("fmt chunk not found")
}

// ExtractPCMData extracts PCM audio data from a WAV file
func ExtractPCMData(wavData []byte) ([]byte, error) {
	if len(wavData) < 44 {
		return nil, fmt.Errorf("WAV file too small: %d bytes", len(wavData))
	}

	// Find data chunk
	pos := 12
	for pos < len(wavData)-8 {
		chunkID := string(wavData[pos : pos+4])
		chunkSize := binary.LittleEndian.Uint32(wavData[pos+4 : pos+8])

		if chunkID == "data" {
			dataStart := pos + 8
			dataEnd := dataStart + int(chunkSize)
			if dataEnd > len(wavData) {
				dataEnd = len(wavData)
			}
			return wavData[dataStart:dataEnd], nil
		}

		pos += 8 + int(chunkSize)
	}

	return nil, fmt.Errorf("data chunk not found")
}

// GenerateSilence generates PCM silence data
// format: audio format (typically 1 for PCM)
// numChannels: 1 for mono, 2 for stereo
// sampleRate: samples per second (e.g., 44100)
// durationMs: duration in milliseconds
// bitsPerSample: 16 for 16-bit audio
func GenerateSilence(numChannels uint16, sampleRate uint32, durationMs int, bitsPerSample uint16) []byte {
	// Calculate number of samples
	samples := (int64(sampleRate) * int64(durationMs)) / 1000

	// Calculate number of bytes
	bytesPerSample := int64(bitsPerSample) / 8
	numBytes := samples * int64(numChannels) * bytesPerSample

	// Create silence (all zeros)
	return make([]byte, numBytes)
}

// ConcatenatePCM concatenates multiple PCM audio data segments
func ConcatenatePCM(segments [][]byte) []byte {
	if len(segments) == 0 {
		return []byte{}
	}

	// Calculate total size
	totalSize := 0
	for _, seg := range segments {
		totalSize += len(seg)
	}

	// Concatenate
	result := make([]byte, 0, totalSize)
	for _, seg := range segments {
		result = append(result, seg...)
	}

	return result
}

// ConcatenateWAV concatenates multiple WAV files into one
// All WAV files must have the same format
func ConcatenateWAV(wavFiles [][]byte) ([]byte, error) {
	if len(wavFiles) == 0 {
		return nil, fmt.Errorf("no WAV files provided")
	}

	// Extract format from first file
	format, err := ExtractWAVFormat(wavFiles[0])
	if err != nil {
		return nil, fmt.Errorf("failed to extract format: %w", err)
	}

	// Extract PCM data from all files
	var pcmSegments [][]byte
	for _, wavData := range wavFiles {
		pcmData, err := ExtractPCMData(wavData)
		if err != nil {
			return nil, fmt.Errorf("failed to extract PCM data: %w", err)
		}
		pcmSegments = append(pcmSegments, pcmData)
	}

	// Concatenate PCM data
	concatenatedPCM := ConcatenatePCM(pcmSegments)

	// Create new WAV file with concatenated PCM data
	return CreateWAVFile(format, concatenatedPCM), nil
}

// CreateWAVFile creates a WAV file from format and PCM data
func CreateWAVFile(format *WAVFormat, pcmData []byte) []byte {
	buf := new(bytes.Buffer)

	// RIFF header
	riffSize := 36 + len(pcmData)
	buf.WriteString("RIFF")
	// #nosec G115 - riffSize is safe for uint32 (typical documents are much smaller than 4GB)
	_ = binary.Write(buf, binary.LittleEndian, uint32(riffSize))
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // fmt chunk size
	_ = binary.Write(buf, binary.LittleEndian, format.AudioFormat)
	_ = binary.Write(buf, binary.LittleEndian, format.NumChannels)
	_ = binary.Write(buf, binary.LittleEndian, format.SampleRate)
	_ = binary.Write(buf, binary.LittleEndian, format.ByteRate)
	_ = binary.Write(buf, binary.LittleEndian, format.BlockAlign)
	_ = binary.Write(buf, binary.LittleEndian, format.BitsPerSample)

	// data chunk
	buf.WriteString("data")
	// #nosec G115 - pcmData length is safe for uint32 (typical documents are much smaller than 4GB)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(pcmData)))
	buf.Write(pcmData)

	return buf.Bytes()
}
