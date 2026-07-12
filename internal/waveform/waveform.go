// Package waveform extracts downsampled peak data from an audio file for
// rendering a waveform in the browser (wavesurfer.js).
package waveform

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

const (
	sampleRate = 8000 // low rate is plenty for peak extraction
	numBuckets = 800  // waveform resolution
)

// Extract decodes the audio file at path via ffmpeg and returns normalized
// (0..1) peak values across numBuckets buckets spanning the whole track,
// along with the track's duration in seconds.
func Extract(path string) (peaks []float32, durationSeconds float64, err error) {
	cmd := exec.Command("ffmpeg",
		"-i", path,
		"-f", "f32le",
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-v", "error",
		"pipe:1",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, 0, fmt.Errorf("ffmpeg decode: %w: %s", err, stderr.String())
	}

	raw := stdout.Bytes()
	numSamples := len(raw) / 4
	if numSamples == 0 {
		return nil, 0, fmt.Errorf("ffmpeg produced no samples for %s", path)
	}

	samples := make([]float32, numSamples)
	for i := range samples {
		bits := binary.LittleEndian.Uint32(raw[i*4 : i*4+4])
		samples[i] = math.Float32frombits(bits)
	}

	durationSeconds = float64(numSamples) / float64(sampleRate)
	return downsample(samples, numBuckets), durationSeconds, nil
}

// Result bundles the peaks (JSON-marshaled) and duration for storage.
type Result struct {
	PeaksJSON       string
	DurationSeconds float64
}

// ExtractResult is a convenience wrapper returning peaks as a JSON array
// plus duration, ready to persist.
func ExtractResult(path string) (*Result, error) {
	peaks, duration, err := Extract(path)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(peaks)
	if err != nil {
		return nil, fmt.Errorf("marshal peaks: %w", err)
	}
	return &Result{PeaksJSON: string(b), DurationSeconds: duration}, nil
}

// downsample splits samples into n buckets and takes the max absolute value
// in each bucket, producing a compact peak envelope.
func downsample(samples []float32, n int) []float32 {
	if len(samples) <= n {
		return samples
	}

	out := make([]float32, n)
	bucketSize := float64(len(samples)) / float64(n)

	for i := 0; i < n; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(samples) {
			end = len(samples)
		}
		var peak float32
		for _, s := range samples[start:end] {
			if abs := float32(math.Abs(float64(s))); abs > peak {
				peak = abs
			}
		}
		out[i] = peak
	}
	return out
}
