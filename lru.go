package last

import (
	"container/list"
	"runtime/debug"
	"sync"
)

type Cache interface {
	// SetMinFreeMemory sets the minimum amount of free memory
	// in bytes before the cache starts evicting objects.
	SetMinFreeMemory(v uint64)

	// Put stores pushes the item to the front of the cache.
	Put(k string, v interface{})

	// Get gets the item from the cache and pushes it to the front.
	Get(k string) (interface{}, bool)

	// Del removes the item from the cache.
	Del(k string)

	// Len returns the number of items stored in the cache.
	Len() int

	// Evict removes the oldest n items from the cache.
	Evict(n int)
}

type lru struct {
	mtx        sync.Mutex
	scheduled  int32
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
		minFreeMem: 1024 * 1024 * 10, // 10MB
		lookup:     make(map[string]*list.Element),
		list:       list.New(),
	}
}

func (c *lru) SetMinFreeMemory(v uint64) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.minFreeMem = v
}

func (c *lru) Put(k string, v interface{}) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if v == nil {
		return
	}
	c.evictIfNecessary()
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
	c.evict(n)
}

func (c *lru) evictIfNecessary() {
	err := refreshMemStats()
	if err != nil {
		panic(err)
	}
	if memStats.Free < c.minFreeMem {
		c.evict(1000)
		debug.FreeOSMemory()
	}
}

func (c *lru) evict(n int) {
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
