// Run with: go run ./examples/batch
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tr1v3r/pkg/pools"
)

func squares(ctx context.Context, inputs []int) ([]int, error) {
	p := pools.NewPool(2)
	results := make([]int, len(inputs))
	var wg sync.WaitGroup
	// Also join already-started workers if a later acquisition is cancelled.
	defer wg.Wait()
	for i, value := range inputs {
		if err := p.Acquire(ctx); err != nil {
			return nil, err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer p.Release()
			results[i] = value * value
		}()
	}
	return results, nil
}

func main() {
	results, err := squares(context.Background(), []int{1, 2, 3, 4, 5})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(results)
}
