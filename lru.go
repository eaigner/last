package last

import (
	"container/list"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type Cache interface {
	//  SetMaxItems sets the maximum number of items the cache can hold to n
	SetMaxItems(n uint)

	// SetMinFreeMemory sets the minimum amount of free memory
	// in bytes before the cache starts evicting objects.
	SetMinFreeMemory(v uint)

	// Put stores and pushes the item to the front of the cache.
	Put(k string, v interface{})

	// Get gets the item from the cache and pushes it to the front.
	Get(k string) (interface{}, bool)

	// Del removes the item from the cache.
	Del(k string)

	// Len returns the number of items stored in the cache.
	Len() int

	// Evict removes the oldest n items from the cache.
	Evict(n uint)
}

type lru struct {
	mtx        sync.Mutex
	maxItems   uint64
	minFreeMem uint64
	lookup     map[string]*list.Element
	list       *list.List
}

type lruItem struct {
	key   string
	value interface{}
}

func New() Cache {
	return &lru{
		minFreeMem: 1024 * 1024 * 100, // 100MB
		lookup:     make(map[string]*list.Element),
		list:       list.New(),
	}
}

func (c *lru) SetMaxItems(n uint) {
	atomic.StoreUint64(&c.maxItems, uint64(n))
}

func (c *lru) SetMinFreeMemory(v uint) {
	atomic.StoreUint64(&c.minFreeMem, uint64(v))
}

func (c *lru) Put(k string, v interface{}) {
	if v == nil {
		return
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.del(k)
	c.lookup[k] = c.list.PushFront(&lruItem{
		key:   k,
		value: v,
	})
	c.evictIfNecessary()
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
	c.del(k)
}

func (c *lru) del(k string) {
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

func (c *lru) Evict(n uint) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.evict(uint64(n))
}

func (c *lru) evictIfNecessary() {
	err := refreshMemStats()
	if err != nil {
		panic(err)
	}
	min := uint64(atomic.LoadUint64(&c.minFreeMem))
	if memStats.Free < min {
		c.evict(uint64(c.list.Len() / 4))
		debug.FreeOSMemory()

		if evictFunc != nil {
			evictFunc()
		}

		// force a read reset, otherwise we might evict
		// the whole cache with subsequent calls.
		lastRead = time.Unix(0, 0)
	}
	n := uint64(c.list.Len())
	max := atomic.LoadUint64(&c.maxItems)
	if max > 1 && n > max {
		c.evict(n - max)
	}
}

var evictFunc func() = nil

func (c *lru) evict(n uint64) {
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
