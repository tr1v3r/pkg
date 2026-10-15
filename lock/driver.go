// Package lock provides backend-independent, owner-checked lease locks.
//
// Locks are time-bounded leases, not fencing tokens or exactly-once execution.
// Callers must stop protected work when a lease expires or its Done channel closes.
package lock

import (
	"context"
	"time"
)

// Driver implements atomic lease operations on a shared backend. Implementations
// must be safe for concurrent use, honor operation deadlines, check cancellation
// before dispatch, and must not invoke methods on the Lease whose operation they
// are handling. Cancellation need not interrupt in-flight I/O immediately, but
// every call must settle by its deadline (apart from scheduling delays).
//
// Every acquisition uses a fresh, unguessable owner. TryAcquire must atomically
// create an absent/expired lease with that owner and a positive TTL. Renew and
// Release must atomically check the owner and require an unexpired lease; an old
// owner must never extend or remove a successor's lease. False means contention
// or lost ownership, not a backend failure. Errors may have ambiguous outcomes.
//
// TTL is measured by the backend; it must not be rounded down. Clock jumps,
// backend failover, pauses, and partitions remain part of the backend's safety
// model. The core cannot turn a weak backend into a strongly consistent lock.
// Driver implementations own their connection resources; Locker never closes them.
type Driver interface {
	TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, owner string) (bool, error)
}
