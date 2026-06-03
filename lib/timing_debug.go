//go:build debug

package fireblazer

import (
	"log"
	"time"
)

// TrackTime logs the elapsed time and unix timestamps for a given block.
func TrackTime(name string) func() {
	start := time.Now()
	log.Printf("[TIMING Start] %s at %d", name, start.UnixMilli())
	return func() {
		end := time.Now()
		log.Printf("[TIMING End] %s took %v (End: %d)", name, time.Since(start), end.UnixMilli())
	}
}

// TrackTimeSince logs the elapsed time since a given start time.
func TrackTimeSince(name string, start time.Time) {
	end := time.Now()
	log.Printf("[TIMING] %s took %v (End: %d)", name, time.Since(start), end.UnixMilli())
}

// TrackWorkerWaveSample tracks the first 3 calls in each batch size wave.
func TrackWorkerWaveSample(name string, index int, batchSize int) func() {
	if index%batchSize >= 3 {
		return func() {}
	}
	start := time.Now()
	log.Printf("[TIMING Worker Start] [%d] %s at %d", index, name, start.UnixMilli())
	return func() {
		end := time.Now()
		log.Printf("[TIMING Worker End] [%d] %s took %v (End: %d)", index, name, time.Since(start), end.UnixMilli())
	}
}
