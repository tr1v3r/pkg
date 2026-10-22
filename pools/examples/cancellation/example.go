// Run with: go run ./examples/cancellation
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tr1v3r/pkg/pools"
)

func main() {
	p := pools.NewPool(1)
	if err := p.Acquire(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer p.Release()

	// Capacity is held above, so this acquisition waits until its deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := p.Acquire(ctx); err != nil {
		fmt.Println(err)                           // context deadline exceeded
		fmt.Println("tokens still held:", p.Num()) // 1; failure did not take a token.
		return                                     // Do not release a token for the failed acquisition.
	}
	p.Release()
}
