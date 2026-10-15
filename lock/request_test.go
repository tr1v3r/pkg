package lock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func lateRequestContext(t *testing.T, kind string) (context.Context, context.CancelFunc, error) {
	t.Helper()
	if kind == "caller deadline" {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		t.Cleanup(cancel)
		return ctx, cancel, context.DeadlineExceeded
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if kind == "caller cancellation" {
		return ctx, cancel, context.Canceled
	}
	return ctx, cancel, context.DeadlineExceeded
}

func awaitLateReply(request context.Context, cancel context.CancelFunc, kind string) {
	if kind == "caller cancellation" {
		cancel()
	}
	<-request.Done()
}

func TestAcquireRejectsLateReplies(t *testing.T) {
	for _, api := range []string{"TryLock", "Lock"} {
		for _, kind := range []string{"operation timeout", "caller deadline", "caller cancellation"} {
			for _, acquired := range []bool{true, false} {
				name := api + "/" + kind + "/busy"
				if acquired {
					name = api + "/" + kind + "/acquired"
				}
				t.Run(name, func(t *testing.T) {
					memory := newMemoryDriver()
					ctx, cancel, want := lateRequestContext(t, kind)
					releases, attempts := 0, 0
					driver := hookedDriver{
						Driver: memory,
						acquire: func(request context.Context, key, owner string, ttl time.Duration) (bool, error) {
							attempts++
							if attempts > 1 {
								t.Fatal("late contention reply was retried")
							}
							if acquired {
								if ok, err := memory.TryAcquire(request, key, owner, ttl); !ok || err != nil {
									t.Fatal("failed to install remote lease", ok, err)
								}
							}
							// Model a successful/busy reply observed only after cancellation
							// or a deadline, including scheduling delays after backend work.
							awaitLateReply(request, cancel, kind)
							return acquired, nil
						},
						release: func(cleanup context.Context, key, owner string) (bool, error) {
							releases++
							if cleanup.Err() != nil {
								t.Error("cleanup reused expired request context", cleanup.Err())
							}
							return memory.Release(cleanup, key, owner)
						},
					}
					timeout := time.Second
					if kind == "operation timeout" {
						timeout = 20 * time.Millisecond
					}
					locker := testLocker(t, driver, WithOperationTimeout(timeout))
					entry := locker.TryLock
					if api == "Lock" {
						entry = locker.Lock
					}
					lease, err := entry(ctx, "k", time.Hour)
					if lease != nil {
						_ = lease.Release(context.Background())
						t.Fatal("late reply returned a lease")
					}
					if !errors.Is(err, want) {
						t.Fatalf("error = %v, want %v", err, want)
					}
					if acquired && !errors.Is(err, ErrLeaseLost) {
						t.Fatalf("late acquisition did not report lease loss: %v", err)
					}
					wantReleases := 0
					if acquired {
						wantReleases = 1
					}
					if attempts != 1 || releases != wantReleases {
						t.Fatalf("attempts/releases = %d/%d, want 1/%d", attempts, releases, wantReleases)
					}
				})
			}
		}
	}
}

func TestRenewRejectsLateReplies(t *testing.T) {
	for _, kind := range []string{"operation timeout", "caller deadline", "caller cancellation"} {
		for _, owned := range []bool{true, false} {
			name := kind + "/lost"
			if owned {
				name = kind + "/owned"
			}
			t.Run(name, func(t *testing.T) {
				ctx, cancel, want := lateRequestContext(t, kind)
				calls := 0
				memory := newMemoryDriver()
				driver := hookedDriver{Driver: memory,
					renew: func(request context.Context, key, owner string, ttl time.Duration) (bool, error) {
						calls++
						if owned {
							if ok, err := memory.Renew(request, key, owner, ttl); !ok || err != nil {
								t.Fatal("failed to renew remote lease", ok, err)
							}
						}
						awaitLateReply(request, cancel, kind)
						return owned, nil
					},
				}
				timeout := time.Second
				if kind == "operation timeout" {
					timeout = 20 * time.Millisecond
				}
				lease := testLease(t, testLocker(t, driver, WithOperationTimeout(timeout)), "k")
				deadline := lease.Deadline()
				err := lease.Renew(ctx)
				if !errors.Is(err, want) || !errors.Is(err, ErrLeaseLost) {
					t.Fatalf("Renew = %v, want lease loss and %v", err, want)
				}
				if !errors.Is(lease.Err(), want) || lease.Context().Err() == nil {
					t.Fatal("late renewal left a valid local lease", lease.Err())
				}
				select {
				case <-lease.Done():
				default:
					t.Fatal("late renewal did not close Done")
				}
				if !lease.Deadline().Equal(deadline) {
					t.Fatal("late renewal extended the local deadline")
				}
				if err := lease.Renew(context.Background()); !errors.Is(err, ErrLeaseLost) || calls != 1 {
					t.Fatal("lost lease attempted another renewal", err, calls)
				}
			})
		}
	}
}

func TestLateAcquirePreservesCleanupFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cleanupFailure := errors.New("cleanup failed")
	driver := hookedDriver{
		acquire: func(context.Context, string, string, time.Duration) (bool, error) {
			cancel()
			return true, nil
		},
		release: func(context.Context, string, string) (bool, error) {
			return false, cleanupFailure
		},
	}
	lease, err := testLocker(t, driver).TryLock(ctx, "k", time.Hour)
	if lease != nil || !errors.Is(err, ErrLeaseLost) ||
		!errors.Is(err, context.Canceled) || !errors.Is(err, cleanupFailure) {
		t.Fatal("late acquisition lost error identity", lease, err)
	}
}

type pendingDeadlineContext struct {
	context.Context
	deadline time.Time
}

func (c pendingDeadlineContext) Deadline() (time.Time, bool) { return c.deadline, true }

func TestRequestErrorBeforeCancellationIsRecorded(t *testing.T) {
	// The clock has passed the deadline, but cancellation is not yet visible.
	ctx := pendingDeadlineContext{Context: context.Background(), deadline: time.Now().Add(-time.Second)}
	if ctx.Err() != nil || !errors.Is(requestError(ctx), context.DeadlineExceeded) {
		t.Fatal("elapsed deadline was accepted while cancellation was pending")
	}
	ctx.deadline = time.Now().Add(time.Hour)
	if err := requestError(ctx); err != nil {
		t.Fatal("future deadline was rejected", err)
	}
	if err := requestError(context.Background()); err != nil {
		t.Fatal("context without a deadline was rejected", err)
	}
}
