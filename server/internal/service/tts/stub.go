// Package tts provides server-side text-to-speech fallback (plan 12.8 phase 1 stub).
package tts

import (
	"encoding/binary"
	"math"
	"strings"
)

const (
	sampleRate = 22050
)

// StubWAV synthesizes a short WAV tone whose duration scales with text length and speed.
// Phase 2 replaces this with Google Cloud TTS / Polly.
func StubWAV(text string, speed float64) []byte {
	if speed <= 0 {
		speed = 1
	}
	wordCount := len(strings.Fields(strings.TrimSpace(text)))
	if wordCount < 1 {
		wordCount = 1
	}
	durationSec := float64(wordCount) * 0.35 / speed
	if durationSec < 0.25 {
		durationSec = 0.25
	}
	if durationSec > 30 {
		durationSec = 30
	}
	numSamples := int(durationSec * sampleRate)
	data := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / sampleRate
		// Gentle alternating tone pattern — audible placeholder, not speech.
		freq := 220.0 + 40*math.Sin(t*2)
		sample := int16(8000 * math.Sin(2*math.Pi*freq*t))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(sample))
	}
	return wrapWAV(data, sampleRate)
}

func wrapWAV(pcm []byte, rate int) []byte {
	dataSize := len(pcm)
	fileSize := 36 + dataSize
	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(fileSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1)
	binary.LittleEndian.PutUint16(buf[22:24], 1)
	binary.LittleEndian.PutUint32(buf[24:28], uint32(rate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(rate*2))
	binary.LittleEndian.PutUint16(buf[32:34], 2)
	binary.LittleEndian.PutUint16(buf[34:36], 16)
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	copy(buf[44:], pcm)
	return buf
}
