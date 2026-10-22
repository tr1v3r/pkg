// Run with: go run ./examples/completion
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tr1v3r/pkg/pools"
)

func main() {
	p := pools.NewPool(2)
	var wg sync.WaitGroup
	results := make([]int, 3)
	for i := range results {
		if err := p.Acquire(context.Background()); err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer p.Release()
			results[i] = i + 10
		}()
	}
	// Completion belongs to the caller. Start waiting after all Add calls.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	fmt.Println("doing other work while the batch finishes")
	<-done
	fmt.Println(results)
}
