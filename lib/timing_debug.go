//go:build debug

package fireblazer

import (
	"log"
	"sync/atomic"
	"time"
)

// TrackTime logs the elapsed time for a given block.
func TrackTime(name string) func() {
	start := time.Now()
	return func() {
		log.Printf("[TIMING] %s took %v", name, time.Since(start))
	}
}

// TrackTimeSince logs the elapsed time since a given start time.
func TrackTimeSince(name string, start time.Time) {
	log.Printf("[TIMING] %s took %v", name, time.Since(start))
}

var workerMonitorCount int32

// TrackWorkerSampleTime tracks only the first 3 calls to avoid log spam.
func TrackWorkerSampleTime(name string) func() {
	if atomic.AddInt32(&workerMonitorCount, 1) > 3 {
		return func() {}
	}
	start := time.Now()
	return func() {
		log.Printf("[TIMING Worker Sample] %s took %v", name, time.Since(start))
	}
}
