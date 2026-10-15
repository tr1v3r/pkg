package lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type runEntry func(context.Context, string, time.Duration, func(context.Context) error) (bool, error)

func runEntries(locker *Locker) map[string]runEntry {
	return map[string]runEntry{"TryRun": locker.TryRun, "Run": locker.Run}
}

func TestRunValidationAndPreCanceledContext(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			var acquisitions, callbacks int
			driver := hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
				acquisitions++
				return true, nil
			}}
			entry := runEntries(testLocker(t, driver))[name]
			fn := func(context.Context) error { callbacks++; return nil }
			for _, tc := range []struct {
				name string
				key  string
				ttl  time.Duration
				fn   func(context.Context) error
				want error
			}{
				{"empty key", "", time.Hour, fn, ErrInvalidKey},
				{"zero TTL", "k", 0, fn, ErrInvalidTTL},
				{"negative TTL", "k", -time.Second, fn, ErrInvalidTTL},
				{"nil callback", "k", time.Hour, nil, ErrNilFunc},
			} {
				t.Run(tc.name, func(t *testing.T) {
					executed, err := entry(context.Background(), tc.key, tc.ttl, tc.fn)
					if executed || !errors.Is(err, tc.want) {
						t.Fatalf("executed=%v err=%v; want false, %v", executed, err, tc.want)
					}
				})
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if executed, err := entry(ctx, "k", time.Hour, fn); executed || !errors.Is(err, context.Canceled) {
				t.Fatalf("pre-canceled call: executed=%v err=%v", executed, err)
			}
			if acquisitions != 0 || callbacks != 0 {
				t.Fatal("invalid call reached backend or callback", acquisitions, callbacks)
			}
		})
	}
}

func TestTryRunBusyMakesSingleAttempt(t *testing.T) {
	calls := 0
	driver := hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
		calls++
		return false, nil
	}}
	executed, err := testLocker(t, driver).TryRun(context.Background(), "k", time.Hour, func(context.Context) error {
		t.Error("busy TryRun invoked callback")
		return nil
	})
	if executed || err != nil || calls != 1 {
		t.Fatal("contention is exactly one (false, nil) attempt", executed, err, calls)
	}
}

func TestRunConcurrentLockersShareWindowAndIndependentKeys(t *testing.T) {
	memory := newMemoryDriver()
	var callbacks, acquisitions atomic.Int32
	driver := hookedDriver{Driver: memory, acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
		acquisitions.Add(1)
		return memory.TryAcquire(ctx, key, owner, ttl)
	}}
	const contenders = 32
	lockers := make([]*Locker, contenders)
	for i := range lockers {
		lockers[i] = testLocker(t, driver)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, locker := range lockers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			executed, err := locker.TryRun(context.Background(), "shared", time.Hour, func(context.Context) error {
				callbacks.Add(1)
				return nil
			})
			if err != nil {
				t.Error(err)
			}
			if executed && callbacks.Load() != 1 {
				t.Error("more than one callback admitted")
			}
		}()
	}
	close(start)
	workers.Wait()
	if callbacks.Load() != 1 || acquisitions.Load() != contenders {
		t.Fatal("shared backend did not enforce one callback", callbacks.Load(), acquisitions.Load())
	}
	for _, key := range []string{"other-a", "other-b"} {
		executed, err := lockers[0].TryRun(context.Background(), key, time.Hour, func(context.Context) error { return nil })
		if !executed || err != nil {
			t.Fatal("different key blocked", key, executed, err)
		}
	}
}

func TestRunOnlyRetriesPlainContentionAndUsesFreshOwners(t *testing.T) {
	owners := make(map[string]bool)
	attempts, callbacks := 0, 0
	driver := hookedDriver{acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
		attempts++
		if key != "k" || ttl != time.Hour || owner == "" || owners[owner] {
			t.Fatal("incorrect acquisition arguments or reused owner", key, owner, ttl)
		}
		owners[owner] = true
		if _, bounded := ctx.Deadline(); !bounded {
			t.Fatal("acquisition has no operation deadline")
		}
		return attempts == 3, nil
	}}
	locker := testLocker(t, driver, WithRetryInterval(time.Millisecond))
	executed, err := locker.Run(context.Background(), "k", time.Hour, func(context.Context) error {
		callbacks++
		return nil
	})
	if !executed || err != nil || attempts != 3 || callbacks != 1 {
		t.Fatal("Run should retry busy attempts, then invoke once", executed, err, attempts, callbacks)
	}
	// Even unsuccessful TryRun calls must generate a new owner, across calls and lockers.
	other := testLocker(t, driver)
	for _, contender := range []*Locker{locker, other, locker} {
		if executed, err = contender.TryRun(context.Background(), "k", time.Hour, func(context.Context) error {
			t.Error("busy attempt invoked callback")
			return nil
		}); executed || err != nil {
			t.Fatal(executed, err)
		}
	}
	if len(owners) != 6 {
		t.Fatal("owner was reused", len(owners))
	}
}

func TestRunBackendErrorsNeverRetry(t *testing.T) {
	failure := errors.New("backend unavailable")
	for _, name := range []string{"TryRun", "Run"} {
		for _, backendErr := range []error{failure, ErrNotAcquired, fmt.Errorf("backend wrapped: %w", ErrNotAcquired)} {
			for _, acquired := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%v/acquired=%v", name, backendErr, acquired), func(t *testing.T) {
					attempts := 0
					driver := hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
						attempts++
						// Fail quickly rather than hang if Run mistakes an error for contention.
						if attempts > 1 {
							return false, failure
						}
						return acquired, backendErr
					}}
					entry := runEntries(testLocker(t, driver))[name]
					executed, err := entry(context.Background(), "k", time.Hour, func(context.Context) error {
						t.Error("errored acquisition invoked callback")
						return nil
					})
					if executed || !errors.Is(err, backendErr) || attempts != 1 {
						t.Fatal("backend error was retried or lost", executed, err, attempts)
					}
				})
			}
		}
	}
}

func TestRunReturnsOriginalCallbackErrorAndRetainsMarker(t *testing.T) {
	failure := errors.New("callback failed")
	for _, name := range []string{"TryRun", "Run"} {
		for _, callbackErr := range []error{nil, failure, ErrNotAcquired, fmt.Errorf("callback wrapped: %w", ErrNotAcquired)} {
			t.Run(fmt.Sprintf("%s/%v", name, callbackErr), func(t *testing.T) {
				memory := newMemoryDriver()
				attempts, callbacks := 0, 0
				driver := hookedDriver{
					Driver: memory,
					acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
						attempts++
						if attempts > 1 {
							return false, failure
						}
						return memory.TryAcquire(ctx, key, owner, ttl)
					},
					renew: func(context.Context, string, string, time.Duration) (bool, error) {
						t.Error("execution entry renewed marker")
						return false, failure
					},
					release: func(context.Context, string, string) (bool, error) {
						t.Error("execution entry released marker")
						return false, failure
					},
				}
				entry := runEntries(testLocker(t, driver, WithAutoRenew()))[name]
				executed, err := entry(context.Background(), "k", time.Hour, func(context.Context) error {
					callbacks++
					return callbackErr
				})
				if !executed || err != callbackErr || callbacks != 1 || attempts != 1 {
					t.Fatal("callback result was wrapped, retried, or lost", executed, err, attempts, callbacks)
				}
				assertRunMarker(t, memory, "k")
				assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
			})
		}
	}
}

func TestRunPanicPropagatesAndRetainsMarker(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			memory := newMemoryDriver()
			entry := runEntries(testLocker(t, memory))[name]
			panicValue := &struct{ message string }{"callback panic"}
			callbacks := 0
			func() {
				defer func() {
					if recovered := recover(); recovered != panicValue {
						t.Errorf("panic changed or swallowed: %v", recovered)
					}
				}()
				_, _ = entry(context.Background(), "k", time.Hour, func(context.Context) error {
					callbacks++
					panic(panicValue)
				})
				t.Error("callback panic was swallowed")
			}()
			if callbacks != 1 {
				t.Fatal("callback invoked more than once", callbacks)
			}
			assertRunMarker(t, memory, "k")
			assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
		})
	}
}

func TestRunCallbackGetsOriginalCallerContext(t *testing.T) {
	type contextKey struct{}
	for _, name := range []string{"TryRun", "Run"} {
		for _, callerDeadline := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deadline=%v", name, callerDeadline), func(t *testing.T) {
				ctx := context.WithValue(context.Background(), contextKey{}, "caller value")
				if callerDeadline {
					withDeadline, cancel := context.WithTimeout(ctx, time.Minute)
					defer cancel()
					ctx = withDeadline
				}
				var request context.Context
				driver := hookedDriver{acquire: func(acquisition context.Context, _ string, _ string, _ time.Duration) (bool, error) {
					request = acquisition
					deadline, bounded := acquisition.Deadline()
					if !bounded || deadline.After(time.Now().Add(time.Second)) {
						t.Fatal("backend request missing operation timeout", deadline, bounded)
					}
					if acquisition.Value(contextKey{}) != "caller value" {
						t.Fatal("backend request lost context values")
					}
					return true, nil
				}}
				entry := runEntries(testLocker(t, driver, WithOperationTimeout(time.Second)))[name]
				executed, err := entry(ctx, "k", time.Hour, func(callbackCtx context.Context) error {
					if callbackCtx != ctx || callbackCtx.Value(contextKey{}) != "caller value" {
						t.Error("callback did not receive original caller context")
					}
					return nil
				})
				if !executed || err != nil || ctx.Err() != nil {
					t.Fatal(executed, err, ctx.Err())
				}
				if request.Err() != context.Canceled {
					t.Fatal("completed backend request context not canceled", request.Err())
				}
			})
		}
	}
}

func TestRunAcquisitionHonorsEarlierCallerDeadline(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			wantDeadline, _ := ctx.Deadline()
			driver := hookedDriver{acquire: func(request context.Context, _ string, _ string, _ time.Duration) (bool, error) {
				deadline, bounded := request.Deadline()
				if !bounded || !deadline.Equal(wantDeadline) {
					t.Fatal("operation deadline outlived caller", deadline, wantDeadline)
				}
				return true, nil
			}}
			entry := runEntries(testLocker(t, driver, WithOperationTimeout(time.Hour)))[name]
			if executed, err := entry(ctx, "k", time.Hour, func(context.Context) error { return nil }); !executed || err != nil {
				t.Fatal(executed, err)
			}
		})
	}
}

func TestRunCanceledCallbackKeepsItsOwnResultAndMarker(t *testing.T) {
	failure := errors.New("callback result after cancel")
	for _, name := range []string{"TryRun", "Run"} {
		for _, callbackErr := range []error{nil, failure, context.Canceled} {
			t.Run(fmt.Sprintf("%s/%v", name, callbackErr), func(t *testing.T) {
				memory := newMemoryDriver()
				entry := runEntries(testLocker(t, memory, WithAutoRenew()))[name]
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				executed, err := entry(ctx, "k", time.Hour, func(callbackCtx context.Context) error {
					cancel()
					if callbackCtx != ctx || callbackCtx.Err() != context.Canceled {
						t.Error("caller cancellation not propagated to callback")
					}
					return callbackErr
				})
				if !executed || err != callbackErr {
					t.Fatal("callback result overridden by context cancellation", executed, err)
				}
				assertRunMarker(t, memory, "k")
				assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
			})
		}
	}
}

func TestRunLateAcquisitionDoesNotExecuteOrCleanUp(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		for _, late := range []string{"caller cancellation", "operation timeout", "backend error"} {
			t.Run(name+"/"+late, func(t *testing.T) {
				memory := newMemoryDriver()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				failure := errors.New("ambiguous backend result")
				want := context.Canceled
				switch late {
				case "operation timeout":
					want = context.DeadlineExceeded
				case "backend error":
					want = failure
				}
				driver := hookedDriver{
					Driver: memory,
					acquire: func(request context.Context, key, owner string, ttl time.Duration) (bool, error) {
						acquired, err := memory.TryAcquire(request, key, owner, ttl)
						if err != nil || !acquired {
							t.Fatal("failed to install remote marker", acquired, err)
						}
						switch late {
						case "caller cancellation":
							cancel()
						case "operation timeout":
							<-request.Done()
						case "backend error":
							return true, failure
						}
						return true, nil
					},
					release: func(context.Context, string, string) (bool, error) {
						t.Error("late acquisition was cleaned up")
						return true, nil
					},
				}
				entry := runEntries(testLocker(t, driver, WithOperationTimeout(20*time.Millisecond)))[name]
				executed, err := entry(ctx, "k", time.Hour, func(context.Context) error {
					t.Error("late or uncertain acquisition invoked callback")
					return nil
				})
				if executed || !errors.Is(err, want) {
					t.Fatal("invalid acquisition admitted callback", executed, err, want)
				}
				assertRunMarker(t, memory, "k")
				assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
			})
		}
	}
}

func TestRunOperationTimeoutBoundsBlockedBackend(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			attempts := 0
			driver := hookedDriver{acquire: func(ctx context.Context, _ string, _ string, _ time.Duration) (bool, error) {
				attempts++
				if _, bounded := ctx.Deadline(); !bounded {
					t.Fatal("backend request has no deadline")
				}
				<-ctx.Done()
				return false, ctx.Err()
			}}
			entry := runEntries(testLocker(t, driver, WithOperationTimeout(20*time.Millisecond)))[name]
			executed, err := entry(context.Background(), "k", time.Hour, func(context.Context) error {
				t.Error("timed out acquisition invoked callback")
				return nil
			})
			if executed || !errors.Is(err, context.DeadlineExceeded) || attempts != 1 {
				t.Fatal("operation timeout was retried or ignored", executed, err, attempts)
			}
		})
	}
}

func TestRunWaitingStopsOnCallerCancellationAndDeadline(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%v", deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				want = context.DeadlineExceeded
			}
			defer cancel()
			started := make(chan struct{})
			var attempts atomic.Int32
			driver := hookedDriver{acquire: func(context.Context, string, string, time.Duration) (bool, error) {
				if attempts.Add(1) == 1 {
					close(started)
				}
				return false, nil
			}}
			locker := testLocker(t, driver, WithRetryInterval(time.Hour))
			result := make(chan error, 1)
			go func() {
				executed, err := locker.Run(ctx, "k", time.Hour, func(context.Context) error {
					t.Error("canceled waiter invoked callback")
					return nil
				})
				if executed {
					t.Error("canceled waiter reported execution")
				}
				result <- err
			}()
			select {
			case <-started:
			case err := <-result:
				t.Fatal("Run returned before its first contended attempt", err)
			case <-time.After(3 * time.Second):
				t.Fatal("Run did not attempt acquisition")
			}
			if !deadline {
				cancel()
			}
			select {
			case err := <-result:
				if !errors.Is(err, want) || attempts.Load() != 1 {
					t.Fatal("wait ignored context or retried after cancellation", err, attempts.Load())
				}
			case <-time.After(3 * time.Second):
				t.Fatal("Run remained blocked after caller cancellation")
			}
		})
	}
}

func assertRunMarker(t *testing.T, memory *memoryDriver, key string) memoryRecord {
	t.Helper()
	memory.mu.Lock()
	defer memory.mu.Unlock()
	record, exists := memory.records[key]
	if !exists || record.owner == "" || !time.Now().Before(record.deadline) {
		t.Fatal("execution marker missing or expired", key, record)
	}
	return record
}

func assertRunBusy(t *testing.T, locker *Locker, key string, ttl time.Duration) {
	t.Helper()
	executed, err := locker.TryRun(context.Background(), key, ttl, func(context.Context) error {
		t.Error("callback admitted inside retained window")
		return nil
	})
	if executed || err != nil {
		t.Fatal("retained window must report (false, nil)", executed, err)
	}
}

func TestRunAndLeaseShareKeyNamespace(t *testing.T) {
	memory := newMemoryDriver()
	locker := testLocker(t, memory, WithAutoRenew())
	other := testLocker(t, memory)
	lease := testLease(t, locker, "shared")
	assertRunBusy(t, other, "shared", time.Hour)
	if err := lease.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	executed, err := other.TryRun(context.Background(), "shared", time.Hour, func(context.Context) error { return nil })
	if !executed || err != nil {
		t.Fatal("released lease still blocked gate", executed, err)
	}
	marker := assertRunMarker(t, memory, "shared")
	if _, lockErr := locker.TryLock(context.Background(), "shared", time.Hour); !errors.Is(lockErr, ErrNotAcquired) {
		t.Fatal("gate marker did not block lease", lockErr)
	}
	if releaseErr := lease.Release(context.Background()); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	if current := assertRunMarker(t, memory, "shared"); current != marker {
		t.Fatal("old lease cleanup changed gate marker", marker, current)
	}
	assertRunBusy(t, other, "shared", time.Hour)
}
