package last

import (
	"container/list"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Cache interface {
	// SetMinFreeMemory sets the minimum amount of free ram
	// before the cache starts evicting objects.
	SetMinFreeMemory(v int64)

	// Put pushes the item to the front of the cache.
	Put(k string, v interface{})

	// Get get the item from the cache and pushes it to the front.
	Get(k string) (interface{}, bool)

	// Del removes the item from the cache
	Del(k string)

	// Len returns the number of items stored in the cache.
	Len() int

	// Evict evicts the last n items from the cache.
	Evict(n int)

	// Schedule starts monitoring the free system memory and evicts
	// items automatically when below the minimum memory threshold.
	Schedule()
}

type lru struct {
	mtx        sync.Mutex
	scheduled  int32
	minFreeMem int64
	lookup     map[string]*list.Element
	list       *list.List
}

type lruItem struct {
	key   string
	value interface{}
}

func New() Cache {
	return &lru{
		minFreeMem: 1024 * 1024 * 10, // 10MB
		lookup:     make(map[string]*list.Element),
		list:       list.New(),
	}
}

func (c *lru) SetMinFreeMemory(v int64) {
	atomic.StoreInt64(&c.minFreeMem, v)
}

func (c *lru) Put(k string, v interface{}) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if v == nil {
		return
	}
	c.lookup[k] = c.list.PushFront(&lruItem{
		key:   k,
		value: v,
	})
}

func (c *lru) Get(k string) (interface{}, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if e, ok := c.lookup[k]; ok {
		c.list.MoveToFront(e)
		return e.Value.(*lruItem).value, true
	}
	return nil, false
}

func (c *lru) Del(k string) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if e, ok := c.lookup[k]; ok {
		c.list.Remove(e)
		delete(c.lookup, k)
	}
}

func (c *lru) Len() int {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.list.Len()
}

func (c *lru) Evict(n int) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for {
		if n < 1 {
			break
		}
		e := c.list.Back()
		delete(c.lookup, e.Value.(*lruItem).key)
		c.list.Remove(e)
		n--
	}
}

func (c *lru) Schedule() {
	if atomic.LoadInt32(&c.scheduled) == 1 {
		return
	}
	atomic.StoreInt32(&c.scheduled, 1)
	go func() {
		for {
			time.Sleep(time.Second * 5)

			var stat SysMemStats
			err := ReadSysMemStats(&stat)
			if err != nil {
				log.Printf("error: %s", err)
				continue
			}

			if stat.Free < uint64(atomic.LoadInt64(&c.minFreeMem)) {
				c.Evict(c.Len() / 3) // evict one third
			}
		}
	}()
}
