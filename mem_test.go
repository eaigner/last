package last

import (
	"testing"
)

func TestReadSysMemStats(t *testing.T) {
	var s sysMemStats
	err := readSysMemStats(&s)
	if err != nil {
		t.Fatal(err)
	}
	// check if value is in a reasonable range (5MB - 64GB)
	var min uint64 = 5 * 1024 * 1024
	var max uint64 = 64 * 1024 * 1024 * 1024

	check := func(v uint64) {
		if v < min || v > max {
			t.Fatal(s)
		}
	}
	check(s.Used)
	check(s.Total)
	check(s.Free)
}
