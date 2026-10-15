//go:build cacheperf

package cache

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"
)

// TestExpiryContentionMeasurement is an opt-in measurement harness, not a latency
// assertion. Run without -race for timing. It calls the real nextDelay cleaner;
// no timer worker runs, so population and timer scheduling cannot skew the result.
func TestExpiryContentionMeasurement(t *testing.T) {
	t.Logf("Go=%s CPUs=%d GOMAXPROCS=%d", runtime.Version(), runtime.NumCPU(), runtime.GOMAXPROCS(0))
	const readers, waves = 32, 20
	for _, identical := range []bool{true, false} {
		shape := "identical"
		if !identical {
			shape = "staggered"
		}
		for _, count := range []int{1000, 10000, 100000} {
			t.Run(fmt.Sprintf("%s/N=%d", shape, count), func(t *testing.T) {
				latencies := make([]time.Duration, 0, readers*waves)
				cleanups := make([]time.Duration, 0, waves)
				ordinary := make([]time.Duration, 0, 100000)
				for wave := 0; wave < waves; wave++ {
					c := &Cache[int, int]{items: make(map[int]*entry[int, int], count+1)}
					c.expirations = make(expiryHeap[int, int], 0, count+1)
					expired := time.Now().Add(-time.Second)
					for key := 0; key < count; key++ {
						deadline := expired
						if !identical {
							// All entries are overdue, but preserve insertion-time
							// ordering as with a burst of equal-TTL Set calls.
							deadline = deadline.Add(time.Duration(key) * time.Nanosecond)
						}
						item := &entry[int, int]{key: key, value: key, deadline: deadline, index: key}
						c.items[key] = item
						c.expirations = append(c.expirations, item)
					}
					live := &entry[int, int]{key: -1, value: 42, deadline: time.Now().Add(time.Hour), index: count}
					c.items[-1] = live
					c.expirations = append(c.expirations, live)
					// Already a valid heap: deadlines are in ascending order.
					for i := 0; i < 5000; i++ {
						start := time.Now()
						value, ok := c.Get(-1)
						ordinary = append(ordinary, time.Since(start))
						if !ok || value != 42 {
							t.Fatal("ordinary read failed")
						}
					}

					// Hold one existing reader until the cleaner is queued. A
					// pending RWMutex writer blocks new readers. This deliberately
					// measures requests that overlap cleanup, not whole-workload P99.
					c.mapMu.RLock()
					cleaned := make(chan time.Time, 1)
					go func() {
						c.nextDelay()
						cleaned <- time.Now()
					}()
					limit := time.Now().Add(5 * time.Second)
					for c.mapMu.TryRLock() {
						c.mapMu.RUnlock()
						if time.Now().After(limit) {
							c.mapMu.RUnlock()
							t.Fatal("cleaner failed to queue")
						}
						runtime.Gosched()
					}
					ready := make(chan struct{}, readers)
					results := make(chan time.Duration, readers)
					failures := make(chan struct{}, readers)
					for i := 0; i < readers; i++ {
						go func() {
							start := time.Now()
							ready <- struct{}{}
							value, ok := c.Get(-1)
							elapsed := time.Since(start)
							if !ok || value != 42 {
								failures <- struct{}{}
							}
							results <- elapsed
						}()
					}
					for i := 0; i < readers; i++ {
						<-ready
					}
					released := time.Now()
					c.mapMu.RUnlock()
					cleanups = append(cleanups, (<-cleaned).Sub(released))
					for i := 0; i < readers; i++ {
						latencies = append(latencies, <-results)
					}
					if len(failures) != 0 {
						t.Fatal("live reads failed during cleanup")
					}
					if len(c.items) != 1 || len(c.expirations) != 1 {
						t.Fatal("cleanup did not remove exactly the expired entries")
					}
				}
				t.Logf("ordinary Get %s; overlapping Get %s; cleanup %s", latencySummary(ordinary), latencySummary(latencies), latencySummary(cleanups))
			})
		}
	}
}

func latencySummary(values []time.Duration) string {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return fmt.Sprintf("p50=%s p99=%s max=%s", values[len(values)/2], values[(len(values)*99-1)/100], values[len(values)-1])
}
