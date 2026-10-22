// Run with: go run ./examples/async-wait
package main

import (
	"fmt"

	"github.com/tr1v3r/pkg/pools"
)

func main() {
	p := pools.NewPool(2)
	results := make([]int, 3)
	for i := range results {
		p.Wait()
		go func() {
			defer p.Done()
			results[i] = i + 10
		}()
	}
	// Start waiting only after every task has acquired its token.
	done := p.AsyncWaitAll()
	fmt.Println("doing other work while the batch finishes")
	<-done
	fmt.Println(results)
}
