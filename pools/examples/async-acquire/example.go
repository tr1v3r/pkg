// Run with: go run ./examples/async-acquire
package main

import (
	"fmt"

	"github.com/tr1v3r/pkg/pools"
)

func main() {
	p := pools.NewPool(1)
	p.Wait()
	ready := p.AsyncWait()
	// The existing token keeps the asynchronous acquisition pending.
	fmt.Println("doing other work while waiting")
	p.Done()

	// The signal closes after acquisition. It is not a cancellation handle.
	<-ready
	go func() {
		defer p.Done()
		fmt.Println("acquired token")
	}()
	p.WaitAll()
	fmt.Println("all tokens returned:", p.Num())
}
