package last

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var (
	schedulerMtx sync.Mutex
	started      int32
	caches       map[Cache]int = make(map[Cache]int)
)

// StartScheduler starts the eviction scheduler.
func StartScheduler() {
	if atomic.LoadInt32(&started) == 1 {
		return
	}
	atomic.StoreInt32(&started, 1)
	go schedule()
}

func schedule() {
	for {
		time.Sleep(time.Second * 5)
		run()
	}
}

func run() {
	schedulerMtx.Lock()
	defer schedulerMtx.Unlock()

	var stat SysMemStats
	err := ReadSysMemStats(&stat)
	if err != nil {
		log.Printf("error: %s", err)
		return
	}

	for c, _ := range caches {
		if stat.Free < uint64(atomic.LoadInt64(&c.(*lru).minFreeMem)) {
			c.Evict(c.Len() / 3) // evict one third
		}
	}
}
