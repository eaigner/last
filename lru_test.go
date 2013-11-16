package last

import (
	"strconv"
	"testing"
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
	c.Evict(c.Len() / 2)

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
	c.Evict(c.Len())

	if x := c.Len(); x != 0 {
		t.Fatal()
	}
	if x := len(c.(*lru).lookup); x != 0 {
		t.Fatal(x)
	}
}
