package last

import (
	"testing"
)

func TestReadSysMemStats(t *testing.T) {
	var s MemStats
	err := ReadSysMemStats(&s)
	if err != nil {
		t.Fatal(err)
	}
	if s.Used == 0 ||
		s.Free == 0 ||
		s.Total == 0 {
		t.Fatal(s)
	}
}
