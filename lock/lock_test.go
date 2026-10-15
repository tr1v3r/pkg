package lock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type memoryRecord struct {
	owner    string
	deadline time.Time
}

type memoryDriver struct {
	mu      sync.Mutex
	records map[string]memoryRecord
}

func newMemoryDriver() *memoryDriver { return &memoryDriver{records: make(map[string]memoryRecord)} }

func (d *memoryDriver) TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if record, ok := d.records[key]; ok && time.Now().Before(record.deadline) {
		return false, nil
	}
	d.records[key] = memoryRecord{owner, time.Now().Add(ttl)}
	return true, nil
}

func (d *memoryDriver) Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	record, ok := d.records[key]
	if !ok || record.owner != owner || !time.Now().Before(record.deadline) {
		return false, nil
	}
	d.records[key] = memoryRecord{owner, time.Now().Add(ttl)}
	return true, nil
}

func (d *memoryDriver) Release(ctx context.Context, key, owner string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	record, ok := d.records[key]
	if !ok || record.owner != owner || !time.Now().Before(record.deadline) {
		return false, nil
	}
	delete(d.records, key)
	return true, nil
}

type hookedDriver struct {
	Driver
	acquire func(context.Context, string, string, time.Duration) (bool, error)
	renew   func(context.Context, string, string, time.Duration) (bool, error)
	release func(context.Context, string, string) (bool, error)
}

func (d hookedDriver) TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if d.acquire != nil {
		return d.acquire(ctx, key, owner, ttl)
	}
	return d.Driver.TryAcquire(ctx, key, owner, ttl)
}

func (d hookedDriver) Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if d.renew != nil {
		return d.renew(ctx, key, owner, ttl)
	}
	return d.Driver.Renew(ctx, key, owner, ttl)
}

func (d hookedDriver) Release(ctx context.Context, key, owner string) (bool, error) {
	if d.release != nil {
		return d.release(ctx, key, owner)
	}
	return d.Driver.Release(ctx, key, owner)
}

func testLocker(t *testing.T, driver Driver, options ...Option) *Locker {
	t.Helper()
	locker, err := New(driver, options...)
	if err != nil {
		t.Fatal(err)
	}
	return locker
}

func testLease(t *testing.T, locker *Locker, key string) *Lease {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	lease, err := locker.TryLock(ctx, key, time.Hour)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = lease.Release(context.Background())
	})
	return lease
}

func waitClosed(t *testing.T, channel <-chan struct{}) {
	t.Helper()
	select {
	case <-channel:
	case <-time.After(3 * time.Second):
		t.Fatal("channel did not close")
	}
}

func TestNewValidation(t *testing.T) {
	var nilDriver *memoryDriver
	for _, driver := range []Driver{nil, nilDriver} {
		if _, err := New(driver); err == nil {
			t.Fatal("nil driver accepted")
		}
	}
	for _, option := range []Option{
		nil, WithRetryInterval(0), WithRetryInterval(-time.Second),
		WithOperationTimeout(0), WithOperationTimeout(-time.Second),
	} {
		if _, err := New(newMemoryDriver(), option); err == nil {
			t.Fatal("invalid option accepted")
		}
	}
	if nilValue(1) {
		t.Fatal("value driver marked nil")
	}
}

func TestAcquisitionValidationAndBackendErrors(t *testing.T) {
	failure := errors.New("backend unavailable")
	locker := testLocker(t, hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
		return false, failure
	}})
	for _, tc := range []struct {
		key string
		ttl time.Duration
		err error
	}{
		{"", time.Hour, ErrInvalidKey},
		{"k", 0, ErrInvalidTTL},
		{"k", -time.Second, ErrInvalidTTL},
		{"k", time.Hour, failure},
	} {
		lease, err := locker.TryLock(context.Background(), tc.key, tc.ttl)
		if lease != nil || !errors.Is(err, tc.err) {
			t.Fatalf("TryLock(%q, %s) = %v, %v", tc.key, tc.ttl, lease, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := locker.TryLock(ctx, "k", time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := locker.Lock(context.Background(), "k", time.Hour); !errors.Is(err, failure) {
		t.Fatal("Lock should not retry backend errors", err)
	}
}

func TestContentionIndependentKeysAndFreshOwners(t *testing.T) {
	driver := newMemoryDriver()
	locker := testLocker(t, driver)
	first := testLease(t, locker, "same")
	if first.Key() != "same" || first.Err() != nil || first.Deadline().Before(time.Now()) {
		t.Fatal("invalid initial lease")
	}
	select {
	case <-first.Done():
		t.Fatal("fresh lease closed")
	default:
	}
	other := testLocker(t, driver)
	if _, err := other.TryLock(context.Background(), "same", time.Hour); !errors.Is(err, ErrNotAcquired) {
		t.Fatal(err)
	}
	_ = testLease(t, other, "different")
	if err := first.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := first.Release(context.Background()); err != nil {
		t.Fatal("Release should be idempotent", err)
	}
	second := testLease(t, other, "same")
	if first.owner == second.owner {
		t.Fatal("owner reused across acquisitions")
	}
	if err := first.Renew(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal("closed lease revived", err)
	}
	if !errors.Is(first.Err(), ErrClosed) || second.Err() != nil {
		t.Fatal("wrong terminal state")
	}
}

func TestLockWaitAndCancellation(t *testing.T) {
	locker := testLocker(t, newMemoryDriver(), WithRetryInterval(time.Millisecond))
	first := testLease(t, locker, "k")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := locker.Lock(ctx, "k", time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	started := make(chan struct{})
	acquired := make(chan *Lease)
	go func() {
		close(started)
		lease, err := locker.Lock(context.Background(), "k", time.Hour)
		if err != nil {
			t.Error(err)
		}
		acquired <- lease
	}()
	<-started
	if err := first.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case lease := <-acquired:
		if lease == nil {
			t.Fatal("no lease")
		}
		if err := lease.Release(context.Background()); err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Lock did not acquire after release")
	}
}

func TestManualRenewAndCancellation(t *testing.T) {
	locker := testLocker(t, newMemoryDriver())
	lease := testLease(t, locker, "k")
	before := lease.Deadline()
	if err := lease.Renew(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !lease.Deadline().After(before) {
		t.Fatal("deadline did not advance")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lease.Renew(canceled); !errors.Is(err, context.Canceled) || lease.Err() != nil {
		t.Fatal("canceled wait should not abandon ownership", err, lease.Err())
	}
	ctx, cancelScope := context.WithCancel(context.Background())
	other, err := locker.TryLock(ctx, "other", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cancelScope()
	waitClosed(t, other.done)
	waitClosed(t, other.stopped)
	if !errors.Is(other.Err(), context.Canceled) || !errors.Is(other.Err(), ErrLeaseLost) {
		t.Fatal(other.Err())
	}
	if err := other.Release(context.Background()); err != nil {
		t.Fatal("canceled lease should permit owner cleanup", err)
	}
}

func TestRenewFailurePermanentlyInvalidates(t *testing.T) {
	failure := errors.New("ambiguous renewal")
	for _, result := range []struct {
		owned bool
		err   error
	}{{false, nil}, {false, failure}, {true, failure}} {
		driver := hookedDriver{Driver: newMemoryDriver(), renew: func(context.Context, string, string, time.Duration) (bool, error) {
			return result.owned, result.err
		}}
		lease := testLease(t, testLocker(t, driver), "k")
		err := lease.Renew(context.Background())
		if !errors.Is(err, ErrLeaseLost) || (result.err != nil && !errors.Is(err, failure)) {
			t.Fatal(err)
		}
		waitClosed(t, lease.done)
		waitClosed(t, lease.stopped)
		if err := lease.Renew(context.Background()); !errors.Is(err, ErrLeaseLost) {
			t.Fatal("lost lease revived", err)
		}
	}
}

func TestReleaseErrorsAreRetryableButRenewIsNot(t *testing.T) {
	failure := errors.New("ambiguous release")
	calls := 0
	memory := newMemoryDriver()
	driver := hookedDriver{Driver: memory, release: func(ctx context.Context, key, owner string) (bool, error) {
		calls++
		if calls == 1 {
			return false, failure
		}
		return memory.Release(ctx, key, owner)
	}}
	lease := testLease(t, testLocker(t, driver), "k")
	if err := lease.Release(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	waitClosed(t, lease.stopped)
	if err := lease.Renew(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := lease.Release(context.Background()); err != nil || calls != 2 {
		t.Fatal("repeated successful release called driver", err, calls)
	}
}

func TestLocalExpiryAndStaleHandleCannotReleaseSuccessor(t *testing.T) {
	driver := newMemoryDriver()
	locker := testLocker(t, driver)
	first := testLease(t, locker, "k")
	first.mu.Lock()
	first.deadline = time.Now().Add(-time.Second)
	first.mu.Unlock()
	if !errors.Is(first.Err(), ErrLeaseLost) {
		t.Fatal(first.Err())
	}
	waitClosed(t, first.stopped)
	driver.mu.Lock()
	delete(driver.records, "k")
	driver.mu.Unlock()
	second := testLease(t, locker, "k")
	for range 2 {
		if err := first.Release(context.Background()); !errors.Is(err, ErrLeaseLost) {
			t.Fatal(err)
		}
	}
	if second.Err() != nil {
		t.Fatal("successor invalidated")
	}
	if err := second.Renew(context.Background()); err != nil {
		t.Fatal("stale release deleted successor", err)
	}
}

func TestReleaseCancelsInflightRenewal(t *testing.T) {
	started := make(chan struct{})
	driver := hookedDriver{Driver: newMemoryDriver(), renew: func(ctx context.Context, _, _ string, _ time.Duration) (bool, error) {
		close(started)
		<-ctx.Done()
		return false, ctx.Err()
	}}
	lease := testLease(t, testLocker(t, driver), "k")
	result := make(chan error, 1)
	go func() { result <- lease.Renew(context.Background()) }()
	<-started
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	waitClosed(t, lease.stopped)
}

func TestContextAwareOperationGate(t *testing.T) {
	lease := testLease(t, testLocker(t, newMemoryDriver()), "k")
	<-lease.gate
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- lease.Renew(ctx) }()
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if lease.Err() != nil {
		t.Fatal("canceled gate wait invalidated lease")
	}
	lease.leave()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lease.Release(canceled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	waitClosed(t, lease.stopped)
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal("retry release with cleanup context", err)
	}
}

func TestSlowAndCanceledSuccessfulAcquisitionAreCleanedUp(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		driver := hookedDriver{
			acquire: func(context.Context, string, string, time.Duration) (bool, error) {
				if canceled {
					cancel()
				} else {
					time.Sleep(2 * time.Millisecond)
				}
				return true, nil
			},
			release: func(ctx context.Context, _, _ string) (bool, error) {
				calls++
				return true, ctx.Err()
			},
		}
		lease, err := testLocker(t, driver).TryLock(ctx, "k", time.Millisecond)
		cancel()
		if lease != nil || !errors.Is(err, ErrLeaseLost) || calls != 1 {
			t.Fatal("invalid lease returned or cleanup missing", lease, err, calls)
		}
	}
}

func TestConcurrentOwnershipAndLeaseOperations(t *testing.T) {
	locker := testLocker(t, newMemoryDriver(), WithRetryInterval(time.Millisecond))
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			lease, err := locker.Lock(ctx, "shared", time.Hour)
			if err != nil {
				t.Error(err)
				return
			}
			if err := lease.Renew(ctx); err != nil {
				t.Error(err)
			}
			if err := lease.Release(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	lease := testLease(t, locker, "readers")
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_ = lease.Key()
				_ = lease.Deadline()
				_ = lease.Err()
				_ = lease.Done()
				_ = lease.Renew(context.Background())
			}
		}()
	}
	wg.Add(1)
	go func() { defer wg.Done(); _ = lease.Release(context.Background()) }()
	wg.Wait()
}
