package watcher

import (
	"context"
	"os"
	"time"
)

// StabilityChecker verifies that a file has finished transferring from camera/SD card.
type StabilityChecker struct {
	checkInterval time.Duration
	requiredCheck int
}

// NewStabilityChecker creates a file stability verifier.
func NewStabilityChecker(intervalMs, requiredChecks int) *StabilityChecker {
	if intervalMs < 500 {
		intervalMs = 500
	}
	if requiredChecks < 1 {
		requiredChecks = 3
	}
	return &StabilityChecker{
		checkInterval: time.Duration(intervalMs) * time.Millisecond,
		requiredCheck: requiredChecks,
	}
}

// WaitUntilStable blocks until the file size and modification time remain constant for required checks.
func (s *StabilityChecker) WaitUntilStable(ctx context.Context, filePath string) bool {
	var lastSize int64 = -1
	var lastModTime time.Time
	stableCount := 0

	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			fi, err := os.Stat(filePath)
			if err != nil {
				// File might have been removed or renamed mid-transfer
				return false
			}

			currSize := fi.Size()
			currModTime := fi.ModTime()

			// Zero byte files are still transferring
			if currSize == 0 {
				stableCount = 0
				continue
			}

			if currSize == lastSize && currModTime.Equal(lastModTime) {
				stableCount++
				if stableCount >= s.requiredCheck {
					// Verify we can open the file for reading (ensures transfer handle is closed)
					if f, err := os.Open(filePath); err == nil {
						_ = f.Close()
						return true
					}
				}
			} else {
				stableCount = 0
				lastSize = currSize
				lastModTime = currModTime
			}
		}
	}
}
