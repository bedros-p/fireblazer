//go:build !debug

package fireblazer

import "time"

// TrackTime is a no-op in production builds.
func TrackTime(name string) func() {
	return func() {}
}

// TrackTimeSince is a no-op in production builds.
func TrackTimeSince(name string, start time.Time) {}

// TrackWorkerWaveSample is a no-op in production builds.
func TrackWorkerWaveSample(name string, index int, batchSize int) func() {
	return func() {}
}
