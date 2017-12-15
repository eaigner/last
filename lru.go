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
	SetMinFreeMemory(v uint64)

	// SetTimeout sets a default timeout duration for cache items in milliseconds.
	SetTimeout(ms uint)

	// Put stores and pushes the item to the front of the cache.
	Put(k string, v interface{})

	// Get gets the item from the cache and pushes it to the front.
	Get(k string) (interface{}, bool)

	// GetOrPut returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value. The bool
	// result is true if the value was loaded, false if stored.
	GetOrPut(k string, v interface{}) (interface{}, bool)

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
	timeout    uint64
	lookup     map[string]*list.Element
	list       *list.List
}

type lruItem struct {
	key     string
	timeout uint64 // absolute ms timestamp
	value   interface{}
}

func New() Cache {
	return &lru{
		lookup: make(map[string]*list.Element),
		list:   list.New(),
	}
}

func (c *lru) SetMaxItems(n uint) {
	atomic.StoreUint64(&c.maxItems, uint64(n))
}

func (c *lru) SetMinFreeMemory(v uint64) {
	atomic.StoreUint64(&c.minFreeMem, v)
}

func (c *lru) SetTimeout(ms uint) {
	atomic.StoreUint64(&c.timeout, uint64(ms))
}

func (c *lru) Put(k string, v interface{}) {
	if v == nil {
		return
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.put(k, v)
}

func (c *lru) put(k string, v interface{}) {
	timeout := atomic.LoadUint64(&c.timeout)

	e, ok := c.lookup[k]
	if ok {
		i := e.Value.(*lruItem)
		if timeout > 0 {
			i.timeout = nowMs() + timeout
		}
		i.value = v
		c.list.MoveToFront(e)
	} else {
		i := &lruItem{
			key:   k,
			value: v,
		}
		if timeout > 0 {
			i.timeout = nowMs() + timeout
		}
		c.lookup[k] = c.list.PushFront(i)
	}
	c.evictIfNecessary()
}

func (c *lru) Get(k string) (interface{}, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if e, ok := c.lookup[k]; ok {
		timeout := e.Value.(*lruItem).timeout
		if timeout > 0 && nowMs() > timeout {
			c.list.Remove(e)
			delete(c.lookup, k)
		} else {
			c.list.MoveToFront(e)
			return e.Value.(*lruItem).value, true
		}
	}
	return nil, false
}

func (c *lru) GetOrPut(k string, v interface{}) (interface{}, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if e, ok := c.lookup[k]; ok {
		timeout := e.Value.(*lruItem).timeout
		if timeout > 0 && nowMs() > timeout {
			c.list.Remove(e)
			delete(c.lookup, k)
		} else {
			c.list.MoveToFront(e)
			return e.Value.(*lruItem).value, true
		}
	}
	if v == nil {
		return nil, false
	}
	c.put(k, v)
	return v, false
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
	// remove items when memory gets low
	minFreeMem := atomic.LoadUint64(&c.minFreeMem)
	if minFreeMem > 0 {
		err := refreshMemStats()
		if err != nil {
			panic(err)
		}
		if memStats.Free < uint64(minFreeMem) {
			c.evict(uint64(c.list.Len() / 4))
			debug.FreeOSMemory()

			if evictFunc != nil {
				evictFunc()
			}

			// force a read reset, otherwise we might evict
			// the whole cache with subsequent calls.
			lastRead = time.Unix(0, 0)
		}
	}

	// remove items that exceed max number of items
	maxItems := atomic.LoadUint64(&c.maxItems)
	if maxItems > 1 {
		n := uint64(c.list.Len())
		if n > maxItems {
			c.evict(n - maxItems)
		}
	}

	// remove timed-out items
	timeout := atomic.LoadUint64(&c.timeout)
	if timeout > 0 {
		now := nowMs()
		for {
			e := c.list.Back()
			if e == nil {
				break
			}
			itm := e.Value.(*lruItem)
			if itm.timeout < now {
				delete(c.lookup, itm.key)
				c.list.Remove(e)
			} else {
				break
			}
		}
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

func nowMs() uint64 {
	return uint64(time.Now().UnixNano() / 1e6)
}
