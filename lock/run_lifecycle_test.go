//go:build go1.25

package lock

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestRunLifecycleWindowSurvivesCallbackResultAndExpires(t *testing.T) {
	failure := errors.New("callback failed")
	for _, name := range []string{"TryRun", "Run"} {
		for _, callbackErr := range []error{nil, failure} {
			t.Run(fmt.Sprintf("%s/%v", name, callbackErr), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					memory := newMemoryDriver()
					locker := testLocker(t, memory)
					entry := runEntries(locker)[name]
					start := time.Now()
					executed, err := entry(context.Background(), "k", 3*time.Second, func(context.Context) error {
						time.Sleep(time.Second)
						return callbackErr
					})
					if !executed || err != callbackErr {
						t.Fatal(executed, err)
					}
					initial := assertRunMarker(t, memory, "k")
					if !initial.deadline.Equal(start.Add(3 * time.Second)) {
						t.Fatal("window restarted at callback completion", initial.deadline)
					}
					assertRunBusy(t, locker, "k", 3*time.Second)
					time.Sleep(2*time.Second - time.Nanosecond)
					assertRunBusy(t, locker, "k", 3*time.Second)
					time.Sleep(time.Nanosecond)
					callbacks := 0
					executed, err = entry(context.Background(), "k", 3*time.Second, func(context.Context) error {
						callbacks++
						return nil
					})
					if !executed || err != nil || callbacks != 1 {
						t.Fatal("expired window did not reopen", executed, err, callbacks)
					}
					if next := assertRunMarker(t, memory, "k"); next.owner == initial.owner {
						t.Fatal("successor reused owner token")
					}
				})
			})
		}
	}
}

func TestRunLifecycleLongCallbackCanOverlapSuccessorWithoutAutoRenew(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				memory := newMemoryDriver()
				var renewals, releases atomic.Int32
				driver := hookedDriver{
					Driver: memory,
					renew: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
						renewals.Add(1)
						return memory.Renew(ctx, key, owner, ttl)
					},
					release: func(ctx context.Context, key, owner string) (bool, error) {
						releases.Add(1)
						return memory.Release(ctx, key, owner)
					},
				}
				locker := testLocker(t, driver, WithAutoRenew())
				entry := runEntries(locker)[name]
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				entered := make(chan struct{})
				finished := make(chan struct{})
				go func() {
					defer close(finished)
					executed, err := entry(ctx, "k", 3*time.Second, func(callbackCtx context.Context) error {
						close(entered)
						time.Sleep(4 * time.Second)
						if callbackCtx != ctx || callbackCtx.Err() != nil {
							t.Error("TTL or operation timeout canceled long callback", callbackCtx.Err())
						}
						return nil
					})
					if !executed || err != nil {
						t.Error("long callback result changed", executed, err)
					}
				}()
				<-entered
				synctest.Wait()
				initial := assertRunMarker(t, memory, "k")
				time.Sleep(2 * time.Second)
				synctest.Wait()
				if current := assertRunMarker(t, memory, "k"); current != initial {
					t.Fatal("WithAutoRenew changed execution window", initial, current)
				}
				assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
				time.Sleep(time.Second)
				synctest.Wait()
				select {
				case <-finished:
					t.Fatal("TTL forced callback completion")
				default:
				}
				other := testLocker(t, memory)
				executed, err := other.TryRun(context.Background(), "k", 10*time.Second, func(context.Context) error { return nil })
				if !executed || err != nil {
					t.Fatal("expired window still excludes successor", executed, err)
				}
				successor := assertRunMarker(t, memory, "k")
				if successor.owner == initial.owner {
					t.Fatal("successor reused owner")
				}
				time.Sleep(time.Second)
				<-finished
				cancel()
				synctest.Wait()
				if current := assertRunMarker(t, memory, "k"); current != successor {
					t.Fatal("old callback completion or cancellation changed successor", successor, current)
				}
				time.Sleep(20 * time.Second)
				synctest.Wait()
				if renewals.Load() != 0 || releases.Load() != 0 {
					t.Fatal("execution API started lease lifecycle work", renewals.Load(), releases.Load())
				}
			})
		})
	}
}

func TestRunLifecycleAutoRenewDoesNotExtendCompletedWindow(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		for _, callbackErr := range []error{nil, errors.New("callback failed")} {
			t.Run(fmt.Sprintf("%s/%v", name, callbackErr), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					memory := newMemoryDriver()
					var renewals, releases atomic.Int32
					driver := hookedDriver{
						Driver: memory,
						renew: func(context.Context, string, string, time.Duration) (bool, error) {
							renewals.Add(1)
							return true, nil
						},
						release: func(context.Context, string, string) (bool, error) {
							releases.Add(1)
							return true, nil
						},
					}
					locker := testLocker(t, driver, WithAutoRenew())
					entry := runEntries(locker)[name]
					executed, err := entry(context.Background(), "k", 3*time.Second, func(context.Context) error { return callbackErr })
					if !executed || err != callbackErr {
						t.Fatal(executed, err)
					}
					initial := assertRunMarker(t, memory, "k")
					synctest.Wait()
					time.Sleep(2 * time.Second)
					synctest.Wait()
					if current := assertRunMarker(t, memory, "k"); current != initial {
						t.Fatal("completed callback changed marker", initial, current)
					}
					assertRunBusy(t, testLocker(t, memory), "k", time.Hour)
					time.Sleep(time.Second)
					synctest.Wait()
					executed, err = entry(context.Background(), "k", 3*time.Second, func(context.Context) error { return nil })
					if !executed || err != nil {
						t.Fatal("WithAutoRenew extended completed execution window", executed, err)
					}
					time.Sleep(10 * time.Second)
					synctest.Wait()
					if renewals.Load() != 0 || releases.Load() != 0 {
						t.Fatal("execution entry called lease operations", renewals.Load(), releases.Load())
					}
				})
			})
		}
	}
}

func TestRunLifecycleOperationTimeoutDoesNotBoundCallback(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				locker := testLocker(t, newMemoryDriver(), WithOperationTimeout(time.Second))
				entry := runEntries(locker)[name]
				executed, err := entry(ctx, "k", 10*time.Second, func(callbackCtx context.Context) error {
					time.Sleep(3 * time.Second)
					if callbackCtx != ctx || callbackCtx.Err() != nil {
						t.Error("operation timeout leaked into callback context", callbackCtx.Err())
					}
					return nil
				})
				if !executed || err != nil {
					t.Fatal(executed, err)
				}
			})
		})
	}
}

func TestRunLifecycleSlowSuccessRejectsElapsedTTLWithoutCleanup(t *testing.T) {
	for _, name := range []string{"TryRun", "Run"} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				memory := newMemoryDriver()
				releases := 0
				driver := hookedDriver{
					Driver: memory,
					acquire: func(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
						// Model a slow response whose remote marker is still live on arrival.
						time.Sleep(3 * time.Second)
						return memory.TryAcquire(ctx, key, owner, ttl)
					},
					release: func(context.Context, string, string) (bool, error) {
						releases++
						return true, nil
					},
				}
				entry := runEntries(testLocker(t, driver, WithOperationTimeout(10*time.Second)))[name]
				executed, err := entry(context.Background(), "k", 3*time.Second, func(context.Context) error {
					t.Error("acquisition used response time instead of request start")
					return nil
				})
				if executed || !errors.Is(err, ErrLeaseLost) || releases != 0 {
					t.Fatal("late successful acquisition admitted or cleaned up", executed, err, releases)
				}
				assertRunMarker(t, memory, "k")
				assertRunBusy(t, testLocker(t, memory), "k", 3*time.Second)
			})
		})
	}
}

func TestRunLifecycleRunWaitsForWindowExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		memory := newMemoryDriver()
		locker := testLocker(t, memory, WithRetryInterval(500*time.Millisecond))
		if executed, err := locker.TryRun(context.Background(), "k", 3*time.Second, func(context.Context) error { return nil }); !executed || err != nil {
			t.Fatal(executed, err)
		}
		initial := assertRunMarker(t, memory, "k")
		started := time.Now()
		callbacks := 0
		executed, err := locker.Run(context.Background(), "k", 3*time.Second, func(context.Context) error {
			callbacks++
			if !time.Now().Equal(started.Add(3 * time.Second)) {
				t.Error("Run did not wait for original acquisition window", time.Since(started))
			}
			return nil
		})
		if !executed || err != nil || callbacks != 1 {
			t.Fatal("Run failed after natural expiry", executed, err, callbacks)
		}
		if successor := assertRunMarker(t, memory, "k"); successor.owner == initial.owner {
			t.Fatal("Run reused predecessor token")
		}
	})
}
