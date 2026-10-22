// Run with: go run ./examples/batch
package main

import (
	"fmt"

	"github.com/tr1v3r/pkg/pools"
)

func main() {
	p := pools.NewPool(2)
	inputs := []int{1, 2, 3, 4, 5}
	results := make([]int, len(inputs))
	for i, value := range inputs {
		// Acquire before spawning to bound the number of worker goroutines.
		p.Wait()
		go func() {
			defer p.Done()
			// Each worker owns one slot; no shared append or map writes.
			results[i] = value * value
		}()
	}
	// All acquisitions have completed before waiting for this batch.
	p.WaitAll()
	fmt.Println(results)
}
