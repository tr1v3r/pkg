package cache_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/tr1v3r/pkg/cache"
)

// Keep entries alive throughout ordinary benchmark runs so these measure the
// named operations rather than expiration. Each benchmark closes its cache.
const benchmarkTTL = 24 * time.Hour

func newBenchmarkCache(b *testing.B, entries int) *cache.Cache[int, int] {
	b.Helper()
	c := cache.New[int, int]()
	b.Cleanup(c.Close)
	for key := 0; key < entries; key++ {
		if err := c.Set(key, key, benchmarkTTL); err != nil {
			b.Fatal(err)
		}
	}
	return c
}

func BenchmarkCacheGetHit(b *testing.B) {
	const entries = 1024
	c := newBenchmarkCache(b, entries)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := i % entries
		if value, ok := c.Get(key); !ok || value != key {
			b.Fatalf("Get(%d) = (%d, %t), want (%d, true)", key, value, ok, key)
		}
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	c := newBenchmarkCache(b, 1024)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, ok := c.Get(-1); ok {
			b.Fatal("unexpected cache hit")
		}
	}
}

func BenchmarkCacheGetParallel(b *testing.B) {
	const entries = 1024
	c := newBenchmarkCache(b, entries)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		// This counter belongs to this worker; no shared benchmark sink is used.
		key := 0
		for pb.Next() {
			if value, ok := c.Get(key); !ok || value != key {
				b.Errorf("Get(%d) = (%d, %t), want (%d, true)", key, value, ok, key)
				return
			}
			key = (key + 1) % entries
		}
	})
}

func BenchmarkCacheOverwrite(b *testing.B) {
	for _, entries := range []int{1, 1024, 65536} {
		b.Run(fmt.Sprintf("Entries=%d", entries), func(b *testing.B) {
			c := newBenchmarkCache(b, entries)
			b.ReportAllocs()
			b.ResetTimer()

			// Refresh only existing keys; storage stays bounded by entries.
			for i := 0; i < b.N; i++ {
				if err := c.Set(i%entries, i, benchmarkTTL); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCacheSetDelete(b *testing.B) {
	const entries = 1024
	c := newBenchmarkCache(b, entries)
	b.ReportAllocs()
	b.ResetTimer()

	// Use one extra key and remove it every iteration. At most entries+1 keys
	// exist, independent of b.N; one iteration measures a Set/Delete pair.
	for i := 0; i < b.N; i++ {
		if err := c.Set(entries, i, benchmarkTTL); err != nil {
			b.Fatal(err)
		}
		c.Delete(entries)
	}
}
