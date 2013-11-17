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
		return readSysMemStats(&memStats)
	}
	return nil
}
