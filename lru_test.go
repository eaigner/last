package last

import (
	"strconv"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New()

	frontKey := func() string {
		return c.(*lru).list.Front().Value.(*lruItem).key
	}

	// Put
	for i := 10; i >= 1; i-- {
		k := strconv.Itoa(i)
		c.Put(k, i)
		if x := frontKey(); x != k {
			t.Fatal(k, x)
		}
	}
	if x := c.Len(); x != 10 {
		t.Fatal(x)
	}

	// Make sure Put overwrites items with the same key
	c.Put("1", 1)
	if x := c.Len(); x != 10 {
		t.Fatal(x)
	}
	if x := len(c.(*lru).lookup); x != 10 {
		t.Fatal(x)
	}

	// Delete
	c.Del("10")
	c.Del("9")

	if x := c.Len(); x != 8 {
		t.Fatal(x)
	}
	if v, ok := c.Get("10"); ok || v != nil {
		t.Fatal(ok, v)
	}
	if v, ok := c.Get("9"); ok || v != nil {
		t.Fatal(ok, v)
	}

	// Push to front on Get
	if v, ok := c.Get("2"); !ok || v.(int) != 2 {
		t.Fatal(ok, v)
	}
	if k := frontKey(); k != "2" {
		t.Fatal(k)
	}

	// Evict half
	c.Evict(uint(c.Len() / 2))

	if x := c.Len(); x != 4 {
		t.Fatal(x)
	}
	if x := len(c.(*lru).lookup); x != 4 {
		t.Fatal(x)
	}
	if _, ok := c.Get("4"); !ok {
		t.Fatal()
	}
	if _, ok := c.Get("5"); ok {
		t.Fatal()
	}

	// Evict all
	c.Evict(uint(c.Len()))

	if x := c.Len(); x != 0 {
		t.Fatal()
	}
	if x := len(c.(*lru).lookup); x != 0 {
		t.Fatal(x)
	}
}

func TestMaxItems(t *testing.T) {
	c := New()
	c.SetMaxItems(2)

	c.Put("a", 1)

	if x := c.Len(); x != 1 {
		t.Fatal(x)
	}

	c.Put("b", 2)

	if x := c.Len(); x != 2 {
		t.Fatal(x)
	}

	c.Put("c", 3)

	if x := c.Len(); x != 2 {
		t.Fatal(x)
	}
}

func TestMemory(t *testing.T) {
	c := New()

	// creates a buffer with alternating ones and zeros
	makeBuf := func(n int) []byte {
		var b = make([]byte, n)
		for i := 0; i < len(b); i++ {
			b[i] = byte(i % 2)
		}
		return b
	}

	var evictCount uint64
	evictFunc = func() {
		evictCount++
	}

	var i int
	var n = 1024 * 1024
	var a uint64

	// Evict if memory consumtion has increased 10MB
	var stats SysMemStats
	err := readSysMemStats(&stats)
	if err != nil {
		t.Fatal(err)
	}
	c.(*lru).minFreeMem = (stats.Free - 1024*1024*10)

	for {
		i++
		lastRead = time.Unix(0, 0) // force read
		c.Put(strconv.Itoa(i), makeBuf(n))
		a += uint64(n)
		if evictCount > 0 {
			break
		}
	}

	if i < 9 || i > 11 {
		t.Fatal(i)
	}
}
