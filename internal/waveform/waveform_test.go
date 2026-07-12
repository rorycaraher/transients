package waveform

import "testing"

func TestExtract(t *testing.T) {
	peaks, duration, err := Extract("testdata/sample.mp3")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if len(peaks) != numBuckets {
		t.Fatalf("got %d buckets, want %d", len(peaks), numBuckets)
	}

	if duration < 1.9 || duration > 2.1 {
		t.Fatalf("expected duration ~2s for the fixture, got %v", duration)
	}

	var maxPeak float32
	for _, p := range peaks {
		if p < 0 || p > 1.01 {
			t.Fatalf("peak %v out of expected [0,1] range", p)
		}
		if p > maxPeak {
			maxPeak = p
		}
	}

	// A steady 440Hz sine tone should produce peaks well above zero
	// throughout, not silence.
	if maxPeak < 0.5 {
		t.Fatalf("expected a strong peak for a sine tone, got max %v", maxPeak)
	}
}

func TestExtractResult(t *testing.T) {
	r, err := ExtractResult("testdata/sample.mp3")
	if err != nil {
		t.Fatalf("ExtractResult: %v", err)
	}
	if len(r.PeaksJSON) == 0 || r.PeaksJSON[0] != '[' {
		t.Fatalf("expected JSON array, got %q", r.PeaksJSON[:min(20, len(r.PeaksJSON))])
	}
	if r.DurationSeconds < 1.9 || r.DurationSeconds > 2.1 {
		t.Fatalf("expected duration ~2s for the fixture, got %v", r.DurationSeconds)
	}
}
