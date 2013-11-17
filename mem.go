package last

import (
	"time"
)

type SysMemStats struct {
	Total uint64
	Used  uint64
	Free  uint64
}

var (
	lastRead time.Time
	memStats SysMemStats
)

func refreshMemStats() error {
	if lastRead.IsZero() || time.Now().Sub(lastRead) > time.Second {
		lastRead = time.Now()
		return ReadSysMemStats(&memStats)
	}
	return nil
}

// ReadSysMemStats reads the system memory statistics into s.
func ReadSysMemStats(s *SysMemStats) error {
	return readSysMemStats(s)
}
