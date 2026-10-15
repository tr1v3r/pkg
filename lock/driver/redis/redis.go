// Package redis implements owner-checked Redis lease operations for package lock.
//
// Acquiring uses a single SET NX PX command; renewing and releasing use atomic
// owner-checking Lua scripts. Positive TTLs are rounded up to milliseconds.
// Prefixes are prepended verbatim to keys; callers should include any desired
// separator and use a namespace reserved for leases. The lock core generates
// owner tokens and manages lease lifecycle; this driver does neither.
//
// Redis expiry depends on its clock. Clock jumps, pauses, partitions, and lost
// writes during asynchronous replication/failover can invalidate lease safety.
// This driver provides neither fencing tokens nor exactly-once execution, nor
// does it make Redis strongly consistent. Callers must account for those limits.
// The caller owns the client and its connection resources; the driver never
// closes the client or flushes the database.
//
// Operations check cancellation before calling Redis. They preserve a supplied
// context deadline, or add a one-second deadline when none is supplied. In-flight
// network I/O is bounded by that deadline, not necessarily interrupted immediately
// by cancellation. Custom clients must honor the supplied deadline. Known
// go-redis clients must enable ContextTimeoutEnabled and socket deadline updates.
package redis

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/tr1v3r/pkg/lock"
)

// Client is the minimal Redis command surface used by Driver. Both
// *redis.Client and *redis.ClusterClient implement it. Implementations must be
// safe for concurrent use and honor supplied context deadlines, including during
// network I/O. Cancellation need not immediately interrupt an in-flight command.
type Client interface {
	Do(context.Context, ...any) *goredis.Cmd
	Eval(context.Context, string, []string, ...any) *goredis.Cmd
}

// Driver implements atomic Redis lease operations. Construct it with New;
// its zero value is not usable. Driver is safe for concurrent use when Client is.
type Driver struct {
	client Client
	prefix string
}

var (
	_ lock.Driver = (*Driver)(nil)
	_ Client      = (*goredis.Client)(nil)
	_ Client      = (*goredis.ClusterClient)(nil)
)

const renewScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`

const releaseScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
    return redis.call('DEL', KEYS[1])
end
return 0
`

// New creates a driver without opening or taking ownership of client resources.
// A nil client, including an interface containing a typed nil, is rejected.
// Known go-redis clients must enable ContextTimeoutEnabled and must not disable
// socket deadline updates (ReadTimeout or WriteTimeout set to -2 before client
// construction). Client options must not be mutated after construction. Custom
// clients must honor supplied context deadlines. An empty prefix is allowed.
func New(client Client, prefix string) (*Driver, error) {
	if isNil(client) {
		return nil, errors.New("lock/redis: nil client")
	}
	var contextTimeoutEnabled bool
	var readTimeout, writeTimeout time.Duration
	switch concrete := client.(type) {
	case *goredis.Client:
		options := concrete.Options()
		contextTimeoutEnabled = options.ContextTimeoutEnabled
		readTimeout, writeTimeout = options.ReadTimeout, options.WriteTimeout
	case *goredis.ClusterClient:
		options := concrete.Options()
		contextTimeoutEnabled = options.ContextTimeoutEnabled
		readTimeout, writeTimeout = options.ReadTimeout, options.WriteTimeout
	default:
		return &Driver{client: client, prefix: prefix}, nil
	}
	if !contextTimeoutEnabled {
		return nil, errors.New("lock/redis: go-redis client must enable ContextTimeoutEnabled")
	}
	// go-redis normalizes -2 to -1 for ordinary clients, but cluster options
	// retain -2 until node creation. Both disable socket deadline updates.
	if readTimeout < 0 || writeTimeout < 0 {
		return nil, errors.New("lock/redis: go-redis client must not disable read or write socket deadlines")
	}
	return &Driver{client: client, prefix: prefix}, nil
}

// TryAcquire atomically creates an absent lease. False means the key is already
// held, even when owner matches; acquisition is not a renewal operation.
func (d *Driver) TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	ctx, cancel := operationContext(ctx)
	defer cancel()
	milliseconds, err := ttlMilliseconds(ttl)
	if err != nil {
		return false, err
	}
	cmd := d.client.Do(ctx, "SET", d.prefix+key, owner, "NX", "PX", milliseconds)
	if cmd == nil {
		return false, errors.New("lock/redis: acquire: nil command")
	}
	result, err := cmd.Result()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock/redis: acquire: %w", err)
	}
	if reply, ok := result.(string); !ok || reply != "OK" {
		return false, fmt.Errorf("lock/redis: acquire: unexpected reply %T (%v)", result, result)
	}
	return true, nil
}

// Renew atomically extends only an unexpired lease belonging to owner. It never
// creates a lease or revives an expired one. False means ownership was lost.
func (d *Driver) Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	ctx, cancel := operationContext(ctx)
	defer cancel()
	milliseconds, err := ttlMilliseconds(ttl)
	if err != nil {
		return false, err
	}
	return scriptResult(d.client.Eval(ctx, renewScript, []string{d.prefix + key}, owner, milliseconds), "renew")
}

// Release atomically deletes only an unexpired lease belonging to owner. False
// means ownership was lost; a successor's lease is never removed.
func (d *Driver) Release(ctx context.Context, key, owner string) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	ctx, cancel := operationContext(ctx)
	defer cancel()
	return scriptResult(d.client.Eval(ctx, releaseScript, []string{d.prefix + key}, owner), "release")
}

func (d *Driver) validate(ctx context.Context, key, owner string) error {
	if d == nil || isNil(d.client) {
		return errors.New("lock/redis: nil driver or client")
	}
	if ctx == nil {
		return errors.New("lock/redis: nil context")
	}
	if key == "" || owner == "" {
		return errors.New("lock/redis: key and owner must not be empty")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("lock/redis: context: %w", err)
	}
	return nil
}

func operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Second)
}

func ttlMilliseconds(ttl time.Duration) (int64, error) {
	if ttl <= 0 {
		return 0, errors.New("lock/redis: TTL must be positive")
	}
	// Divide before rounding: adding time.Millisecond-1 to ttl could overflow.
	milliseconds := int64(ttl / time.Millisecond)
	if ttl%time.Millisecond != 0 {
		milliseconds++
	}
	return milliseconds, nil
}

func scriptResult(cmd *goredis.Cmd, operation string) (bool, error) {
	if cmd == nil {
		return false, fmt.Errorf("lock/redis: %s: nil command", operation)
	}
	result, err := cmd.Result()
	if err != nil {
		return false, fmt.Errorf("lock/redis: %s: %w", operation, err)
	}
	if value, ok := result.(int64); ok && (value == 0 || value == 1) {
		return value == 1, nil
	}
	return false, fmt.Errorf("lock/redis: %s: unexpected reply %T (%v)", operation, result, result)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	switch reflect.ValueOf(value).Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflect.ValueOf(value).IsNil()
	default:
		return false
	}
}
