package last

type SysMemStats struct {
	Total uint64
	Used  uint64
	Free  uint64
}

// ReadSysMemStats reads the system memory stats and writes them to s.
func ReadSysMemStats(s *SysMemStats) error {
	return readSysMemStats(s)
}
