package last

import (
	"syscall"
)

func readSysMemStats(s *sysMemStats) error {
	if s == nil {
		return nil
	}
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		return err
	}

	s.Total = info.Totalram
	s.Free = info.Freeram
	s.Used = s.Total - s.Free

	return nil
}
