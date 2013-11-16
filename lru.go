package last

import (
	"container/list"
	"sync"
	"sync/atomic"
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

	// Schedule adds the cache to the eviction scheduler that evicts it
	// automatically when the system memory is below the minimum threshold.
	//
	// You must unschedule the cache when you no longer need it, or it's memory won't be freed up.
	Schedule()

	// Unschedule removes the cache from the eviction scheduler.
	Unschedule()
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
	schedulerMtx.Lock()
	defer schedulerMtx.Unlock()
	caches[c] = 1
}

func (c *lru) Unschedule() {
	schedulerMtx.Lock()
	defer schedulerMtx.Unlock()
	delete(caches, c)
}
