//go:build integration

package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/tr1v3r/pkg/lock"
)

// Run with PKG_LOCK_POSTGRES_DSN='postgres://...?...' go test -tags=integration
// -timeout=30s ./lock/driver/postgres. Every test creates a random public table
// and prefix, and cleanup only drops that table. Never clean a shared database.
type postgresFixture struct {
	db     *sql.DB
	driver *Driver
	ctx    context.Context
	table  string
	prefix string
}

func realPostgres(t *testing.T) *postgresFixture {
	t.Helper()
	dsn := os.Getenv("PKG_LOCK_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set PKG_LOCK_POSTGRES_DSN to run real PostgreSQL integration tests")
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	suffix := hex.EncodeToString(random[:])
	f := &postgresFixture{table: "pkg_lock_test_" + suffix, prefix: "pkg-lock:" + suffix + ":"}
	var err error
	f.db, err = sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	f.db.SetMaxOpenConns(24)
	var cancel context.CancelFunc
	f.ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(func() {
		defer cancel()
		defer f.db.Close()
		// Independent of the test context, which may have expired or canceled.
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		// table contains only a fixed prefix plus crypto/rand hex generated above.
		if _, cleanupErr := f.db.ExecContext(cleanup, `DROP TABLE IF EXISTS `+f.quotedTable()); cleanupErr != nil {
			t.Errorf("drop own integration table: %v", cleanupErr)
		}
	})
	f.driver, err = New(f.db, f.prefix, WithTable("public."+f.table))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.driver.EnsureSchema(f.ctx); err != nil {
		t.Fatalf("initialize own PostgreSQL table: %v", err)
	}
	return f
}

func (f *postgresFixture) quotedTable() string { return `"public"."` + f.table + `"` }

func (f *postgresFixture) anotherDriver(t *testing.T, prefix string) *Driver {
	t.Helper()
	d, err := New(f.db, prefix, WithTable("public."+f.table))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func (f *postgresFixture) row(t *testing.T, key string) (string, time.Time) {
	t.Helper()
	var owner string
	var expires time.Time
	if err := f.db.QueryRowContext(f.ctx, `SELECT owner, expires_at FROM `+f.quotedTable()+` WHERE key = $1`, f.prefix+key).Scan(&owner, &expires); err != nil {
		t.Fatal(err)
	}
	return owner, expires
}

func waitPostgres(t *testing.T, ctx context.Context, description string, ready func() (bool, error)) {
	t.Helper()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		ok, err := ready()
		if err != nil {
			t.Fatalf("%s: %v", description, err)
		}
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s", description)
		case <-ticker.C:
		}
	}
}

func (f *postgresFixture) waitExpired(t *testing.T, expires time.Time) {
	t.Helper()
	waitPostgres(t, f.ctx, "database wall-clock expiry", func() (bool, error) {
		var expired bool
		err := f.db.QueryRowContext(f.ctx, `SELECT clock_timestamp() >= $1::timestamptz`, expires).Scan(&expired)
		return expired, err
	})
}

func TestIntegrationPostgresOwnershipAndExpiry(t *testing.T) {
	f := realPostgres(t)
	first, second := f.driver, f.anotherDriver(t, f.prefix)
	if err := first.EnsureSchema(f.ctx); err != nil {
		t.Fatalf("EnsureSchema must be idempotent: %v", err)
	}
	ok, err := first.TryAcquire(f.ctx, "key", "old", time.Second)
	requireResult(t, true, ok, err)
	_, originalExpiry := f.row(t, "key")
	for _, owner := range []string{"old", "other"} {
		ok, err = second.TryAcquire(f.ctx, "key", owner, time.Hour)
		requireResult(t, false, ok, err)
	}
	ok, err = second.Renew(f.ctx, "key", "other", time.Hour)
	requireResult(t, false, ok, err)
	ok, err = second.Release(f.ctx, "key", "other")
	requireResult(t, false, ok, err)
	owner, expires := f.row(t, "key")
	if owner != "old" || !expires.Equal(originalExpiry) {
		t.Fatal("contention or wrong owner mutated live lease", owner, expires, originalExpiry)
	}
	ok, err = first.Renew(f.ctx, "key", "old", 2*time.Second)
	requireResult(t, true, ok, err)
	_, extended := f.row(t, "key")
	if !extended.After(originalExpiry) {
		t.Fatal("Renew did not extend real TTL", extended, originalExpiry)
	}
	// A real-clock expiry is observable without relying on scheduler sleeps.
	ok, err = first.Renew(f.ctx, "key", "old", 80*time.Millisecond)
	requireResult(t, true, ok, err)
	_, expires = f.row(t, "key")
	f.waitExpired(t, expires)
	ok, err = first.Renew(f.ctx, "key", "old", time.Hour)
	requireResult(t, false, ok, err)
	ok, err = first.Release(f.ctx, "key", "old")
	requireResult(t, false, ok, err)
	owner, unchanged := f.row(t, "key")
	if owner != "old" || !unchanged.Equal(expires) {
		t.Fatal("expired owner revived or changed expired row", owner, unchanged)
	}
	ok, err = second.TryAcquire(f.ctx, "key", "successor", time.Second)
	requireResult(t, true, ok, err)
	_, successorExpiry := f.row(t, "key")
	ok, err = first.Renew(f.ctx, "key", "old", time.Hour)
	requireResult(t, false, ok, err)
	ok, err = first.Release(f.ctx, "key", "old")
	requireResult(t, false, ok, err)
	owner, unchanged = f.row(t, "key")
	if owner != "successor" || !unchanged.Equal(successorExpiry) {
		t.Fatal("stale owner damaged successor", owner, unchanged)
	}
	ok, err = second.Release(f.ctx, "key", "successor")
	requireResult(t, true, ok, err)
	ok, err = second.Release(f.ctx, "key", "successor")
	requireResult(t, false, ok, err)
	ok, err = second.Renew(f.ctx, "key", "successor", time.Second)
	requireResult(t, false, ok, err)
	if err := f.db.PingContext(f.ctx); err != nil {
		t.Fatal("Release closed caller's database", err)
	}
}

func TestIntegrationPostgresIndependentKeysAndPrefixes(t *testing.T) {
	f := realPostgres(t)
	other := f.anotherDriver(t, f.prefix+"other:")
	for _, pair := range []struct {
		d   *Driver
		key string
	}{{f.driver, "a"}, {f.driver, "b"}, {other, "a"}} {
		ok, err := pair.d.TryAcquire(f.ctx, pair.key, "owner", time.Second)
		requireResult(t, true, ok, err)
	}
	ok, err := f.driver.Release(f.ctx, "a", "owner")
	requireResult(t, true, ok, err)
	for _, pair := range []struct {
		d   *Driver
		key string
	}{{f.driver, "b"}, {other, "a"}} {
		ok, err := pair.d.Renew(f.ctx, pair.key, "owner", time.Second)
		requireResult(t, true, ok, err)
	}
}

func TestIntegrationPostgresTextInjectionAndMaximumTTL(t *testing.T) {
	f := realPostgres(t)
	const payload = "'; SELECT pg_sleep(10); --"
	d := f.anotherDriver(t, f.prefix+payload)
	ok, err := d.TryAcquire(f.ctx, payload, payload, time.Second)
	requireResult(t, true, ok, err)
	ok, err = d.Renew(f.ctx, payload, payload, time.Second)
	requireResult(t, true, ok, err)
	ok, err = d.Release(f.ctx, payload, payload)
	requireResult(t, true, ok, err)

	// PostgreSQL 12 rejects oversized integer fields in interval text. The
	// driver's exact, decomposed representation must also work on the server,
	// without converting nanoseconds through floating point or overflowing.
	const maximumTTL time.Duration = 1<<63 - 1
	ok, err = f.driver.TryAcquire(f.ctx, "maximum", "owner", maximumTTL)
	requireResult(t, true, ok, err)
	ok, err = f.driver.Renew(f.ctx, "maximum", "owner", maximumTTL)
	requireResult(t, true, ok, err)
	// Together with the sqlmock argument assertion, check the server parser's
	// exact microseconds through interval's binary representation (not epoch's
	// floating-point representation on older PostgreSQL releases).
	var intervalBytes []byte
	const maximumInterval = "2562047 hours 47 minutes 16 seconds 854776 microseconds"
	if err := f.db.QueryRowContext(f.ctx, `SELECT pg_catalog.interval_send($1::pg_catalog.interval)`, maximumInterval).Scan(&intervalBytes); err != nil {
		t.Fatal(err)
	}
	if len(intervalBytes) != 16 || binary.BigEndian.Uint64(intervalBytes[:8]) != 9223372036854776 || binary.BigEndian.Uint32(intervalBytes[8:12]) != 0 || binary.BigEndian.Uint32(intervalBytes[12:]) != 0 {
		t.Fatalf("maximum interval binary encoding = %x; want exact microseconds and no calendar days/months", intervalBytes)
	}
	var centuriesAway bool
	if err := f.db.QueryRowContext(f.ctx, `SELECT expires_at > clock_timestamp() + interval '292 years' FROM `+f.quotedTable()+` WHERE key = $1`, f.prefix+"maximum").Scan(&centuriesAway); err != nil || !centuriesAway {
		t.Fatal("maximum TTL overflowed or lost scale", centuriesAway, err)
	}
}

func TestIntegrationPostgresConcurrentAcquisitionHasOneWinner(t *testing.T) {
	f := realPostgres(t)
	const count = 16
	start := make(chan struct{})
	results := make(chan string, count)
	errorsFound := make(chan error, count)
	var workers sync.WaitGroup
	for i := range count {
		d := f.anotherDriver(t, f.prefix)
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			owner := fmt.Sprintf("owner-%d", i)
			ok, err := d.TryAcquire(f.ctx, "concurrent", owner, 3*time.Second)
			if err != nil {
				errorsFound <- err
			} else if ok {
				results <- owner
			}
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		t.Error(err)
	}
	if len(results) != 1 {
		t.Fatalf("acquisition winners = %d; want exactly 1", len(results))
	}
	winner := <-results
	if owner, _ := f.row(t, "concurrent"); owner != winner {
		t.Fatalf("stored owner = %q; want winner %q", owner, winner)
	}
}

// holdRow starts a transaction BEFORE acquisition, then locks the live lease
// without modifying any value. The old transaction clock and unchanged tuple
// catch implementations that pre-filter expires_at before SELECT FOR UPDATE.
func (f *postgresFixture) holdRow(t *testing.T, key string, ttl time.Duration) (*sql.Tx, int, time.Time) {
	t.Helper()
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	var pid int
	var began time.Time
	if queryErr := tx.QueryRowContext(f.ctx, `SELECT pg_backend_pid(), CURRENT_TIMESTAMP`).Scan(&pid, &began); queryErr != nil {
		t.Fatal(queryErr)
	}
	ok, err := f.driver.TryAcquire(f.ctx, key, "owner", ttl)
	requireResult(t, true, ok, err)
	var expires time.Time
	if queryErr := tx.QueryRowContext(f.ctx, `SELECT expires_at FROM `+f.quotedTable()+` WHERE key = $1 FOR UPDATE`, f.prefix+key).Scan(&expires); queryErr != nil {
		t.Fatal(queryErr)
	}
	if !began.Before(expires) {
		t.Fatal("blocker transaction did not begin before lease expiry")
	}
	return tx, pid, expires
}

func (f *postgresFixture) waitBlocked(t *testing.T, blockerPID int) time.Time {
	t.Helper()
	var queryStarted time.Time
	waitPostgres(t, f.ctx, "driver actually waiting on row lock", func() (bool, error) {
		err := f.db.QueryRowContext(f.ctx, `SELECT query_start FROM pg_stat_activity
			WHERE datname = current_database() AND pid <> pg_backend_pid()
			AND $1::int = ANY(pg_blocking_pids(pid)) AND wait_event_type = 'Lock'
			ORDER BY query_start LIMIT 1`, blockerPID).Scan(&queryStarted)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return err == nil, err
	})
	return queryStarted
}

func TestIntegrationPostgresExpiryRecheckedAfterRowLockWait(t *testing.T) {
	for _, name := range []string{"renew", "release"} {
		t.Run(name, func(t *testing.T) {
			f := realPostgres(t)
			tx, blockerPID, expires := f.holdRow(t, "blocked", 200*time.Millisecond)
			type result struct {
				ok  bool
				err error
			}
			finished := make(chan result, 1)
			go func() {
				var ok bool
				var err error
				if name == "renew" {
					ok, err = f.driver.Renew(f.ctx, "blocked", "owner", time.Hour)
				} else {
					ok, err = f.driver.Release(f.ctx, "blocked", "owner")
				}
				finished <- result{ok, err}
			}()
			started := f.waitBlocked(t, blockerPID)
			if !started.Before(expires) {
				t.Fatal("operation was not issued before TTL expired; row-lock regression not exercised")
			}
			f.waitExpired(t, expires)
			select {
			case r := <-finished:
				t.Fatalf("operation returned while row remained locked: %v, %v", r.ok, r.err)
			default:
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case r := <-finished:
				requireResult(t, false, r.ok, r.err)
			case <-f.ctx.Done():
				t.Fatal(f.ctx.Err())
			}
			owner, unchanged := f.row(t, "blocked")
			if owner != "owner" || !unchanged.Equal(expires) {
				t.Fatal("post-expiry waiter renewed/deleted lease", owner, unchanged)
			}
			// CURRENT_TIMESTAMP/transaction_timestamp are fixed before the wait;
			// only clock_timestamp at the locked-row decision can reject expiry.
			ok, err := f.driver.TryAcquire(f.ctx, "blocked", "successor", time.Second)
			requireResult(t, true, ok, err)
		})
	}
}

func TestIntegrationPostgresWaiterRechecksCommittedSuccessor(t *testing.T) {
	for _, name := range []string{"acquire", "renew", "release"} {
		t.Run(name, func(t *testing.T) {
			f := realPostgres(t)
			ok, err := f.driver.TryAcquire(f.ctx, "successor", "old", 5*time.Second)
			requireResult(t, true, ok, err)
			tx, err := f.db.BeginTx(f.ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var blockerPID int
			if queryErr := tx.QueryRowContext(f.ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID); queryErr != nil {
				t.Fatal(queryErr)
			}
			var successorExpiry time.Time
			// An external UPDATE is only a test fixture to exercise EvalPlanQual:
			// the waiter initially sees old, but must decide on committed new.
			if err := tx.QueryRowContext(f.ctx, `UPDATE `+f.quotedTable()+` SET owner = $2, expires_at = clock_timestamp() + interval '5 seconds' WHERE key = $1 RETURNING expires_at`, f.prefix+"successor", "new").Scan(&successorExpiry); err != nil {
				t.Fatal(err)
			}
			type result struct {
				ok  bool
				err error
			}
			finished := make(chan result, 1)
			go func() {
				var ok bool
				var err error
				switch name {
				case "acquire":
					ok, err = f.driver.TryAcquire(f.ctx, "successor", "old", time.Hour)
				case "renew":
					ok, err = f.driver.Renew(f.ctx, "successor", "old", time.Hour)
				case "release":
					ok, err = f.driver.Release(f.ctx, "successor", "old")
				}
				finished <- result{ok, err}
			}()
			f.waitBlocked(t, blockerPID)
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case r := <-finished:
				requireResult(t, false, r.ok, r.err)
			case <-f.ctx.Done():
				t.Fatal(f.ctx.Err())
			}
			if owner, expires := f.row(t, "successor"); owner != "new" || !expires.Equal(successorExpiry) {
				t.Fatal("waiter damaged committed successor", owner, expires)
			}
		})
	}
}

func TestIntegrationPostgresCoreRejectsAcquisitionAfterTTLSpentWaiting(t *testing.T) {
	for _, name := range []string{"TryRun", "TryLock"} {
		t.Run(name, func(t *testing.T) {
			f := realPostgres(t)
			tx, err := f.db.BeginTx(f.ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			var blockerPID int
			if queryErr := tx.QueryRowContext(f.ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID); queryErr != nil {
				t.Fatal(queryErr)
			}
			// The driver sees an absent key, then blocks on this uncommitted
			// unique-index entry. Rollback lets INSERT succeed only after TTL.
			if _, insertErr := tx.ExecContext(f.ctx, `INSERT INTO `+f.quotedTable()+` (key, owner, expires_at) VALUES ($1, 'blocker', clock_timestamp() + interval '5 seconds')`, f.prefix+"late"); insertErr != nil {
				t.Fatal(insertErr)
			}
			locker, err := lock.New(f.driver, lock.WithOperationTimeout(2*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			const ttl = 80 * time.Millisecond
			type result struct {
				executed bool
				lease    *lock.Lease
				err      error
			}
			finished := make(chan result, 1)
			go func() {
				var r result
				if name == "TryRun" {
					r.executed, r.err = locker.TryRun(f.ctx, "late", ttl, func(context.Context) error {
						t.Error("TTL-spent acquisition invoked callback")
						return nil
					})
				} else {
					r.lease, r.err = locker.TryLock(f.ctx, "late", ttl)
				}
				finished <- r
			}()
			started := f.waitBlocked(t, blockerPID)
			f.waitExpired(t, started.Add(ttl+20*time.Millisecond))
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				t.Fatal(rollbackErr)
			}
			select {
			case r := <-finished:
				if r.executed || r.lease != nil || !errors.Is(r.err, lock.ErrLeaseLost) {
					if r.lease != nil {
						_ = r.lease.Release(f.ctx)
					}
					t.Fatal("core accepted TTL-spent acquisition", r.executed, r.lease != nil, r.err)
				}
			case <-f.ctx.Done():
				t.Fatal(f.ctx.Err())
			}
			ok, err := f.driver.TryAcquire(f.ctx, "late", "successor", time.Second)
			requireResult(t, true, ok, err)
		})
	}
}

func TestIntegrationPostgresBlockedOperationsHonorDeadline(t *testing.T) {
	for _, name := range []string{"acquire", "renew", "release"} {
		t.Run(name, func(t *testing.T) {
			f := realPostgres(t)
			tx, blockerPID, _ := f.holdRow(t, "deadline", 5*time.Second)
			ctx, cancel := context.WithTimeout(f.ctx, 250*time.Millisecond)
			defer cancel()
			type result struct {
				ok  bool
				err error
			}
			finished := make(chan result, 1)
			started := time.Now()
			go func() {
				var ok bool
				var err error
				switch name {
				case "acquire":
					ok, err = f.driver.TryAcquire(ctx, "deadline", "intruder", time.Hour)
				case "renew":
					ok, err = f.driver.Renew(ctx, "deadline", "owner", time.Hour)
				case "release":
					ok, err = f.driver.Release(ctx, "deadline", "owner")
				}
				finished <- result{ok, err}
			}()
			f.waitBlocked(t, blockerPID)
			select {
			case r := <-finished:
				if r.ok || !errors.Is(r.err, context.DeadlineExceeded) {
					t.Fatalf("deadline request = %v, %v; want false, deadline error", r.ok, r.err)
				}
				if ctx.Err() == nil || time.Since(started) > time.Second {
					t.Fatal("driver did not honor caller deadline promptly", ctx.Err(), time.Since(started))
				}
			case <-time.After(2 * time.Second):
				t.Fatal("blocked operation ignored deadline")
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			// A deadline bounds the local response, not remote commit certainty:
			// cancellation may still be in flight when this blocker is released.
			// Renew may extend its own lease and Release may delete its own row.
			// Stale-owner safety against a successor is tested independently.
			var owner string
			queryErr := f.db.QueryRowContext(f.ctx, `SELECT owner FROM `+f.quotedTable()+` WHERE key = $1`, f.prefix+"deadline").Scan(&owner)
			if queryErr != nil && (name != "release" || !errors.Is(queryErr, sql.ErrNoRows)) {
				t.Fatal(queryErr)
			}
			if queryErr == nil && owner != "owner" {
				t.Fatal("deadline request replaced original owner", owner)
			}
		})
	}
}

func TestIntegrationPostgresPoolExhaustionHonorsDeadline(t *testing.T) {
	f := realPostgres(t)
	ok, err := f.driver.TryAcquire(f.ctx, "pool", "owner", 5*time.Second)
	requireResult(t, true, ok, err)
	_, expires := f.row(t, "pool")
	f.db.SetMaxOpenConns(1)
	conn, err := f.db.Conn(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, name := range []string{"acquire", "renew", "release", "schema"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(f.ctx, 25*time.Millisecond)
			defer cancel()
			started := time.Now()
			ok := false
			var callErr error
			switch name {
			case "acquire":
				ok, callErr = f.driver.TryAcquire(ctx, "pool", "intruder", time.Hour)
			case "renew":
				ok, callErr = f.driver.Renew(ctx, "pool", "owner", time.Hour)
			case "release":
				ok, callErr = f.driver.Release(ctx, "pool", "owner")
			case "schema":
				callErr = f.driver.EnsureSchema(ctx)
			}
			if ok || !errors.Is(callErr, context.DeadlineExceeded) || time.Since(started) > time.Second {
				t.Fatal("pool wait did not honor deadline", ok, callErr, time.Since(started))
			}
		})
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if owner, unchanged := f.row(t, "pool"); owner != "owner" || !unchanged.Equal(expires) {
		t.Fatal("pool-wait cancellation changed original owner", owner, unchanged)
	}
}

func TestIntegrationPostgresCoreGateWindows(t *testing.T) {
	f := realPostgres(t)
	locker, err := lock.New(f.driver, lock.WithRetryInterval(5*time.Millisecond), lock.WithAutoRenew())
	if err != nil {
		t.Fatal(err)
	}
	const ttl = 400 * time.Millisecond
	failure := errors.New("callback failed")
	entries := []struct {
		name string
		call func(context.Context, string, time.Duration, func(context.Context) error) (bool, error)
	}{{"TryRun", locker.TryRun}, {"Run", locker.Run}}
	for _, entry := range entries {
		for _, outcome := range []struct {
			name string
			err  error
		}{{"success", nil}, {"failure", failure}} {
			t.Run(entry.name+"/"+outcome.name, func(t *testing.T) {
				t.Parallel() // Distinct keys keep the longer positive-TTL windows fast.
				key := entry.name + "-" + outcome.name
				executed, err := entry.call(f.ctx, key, ttl, func(ctx context.Context) error {
					if ctx != f.ctx {
						t.Error("gate replaced caller context")
					}
					return outcome.err
				})
				if !executed || err != outcome.err {
					t.Fatal("callback result changed", executed, err)
				}
				owner, expires := f.row(t, key)
				if owner == "" {
					t.Fatal("gate removed marker")
				}
				executed, err = locker.TryRun(f.ctx, key, ttl, func(context.Context) error {
					t.Error("retained gate admitted competing callback")
					return nil
				})
				requireResult(t, false, executed, err)
				executed, err = locker.Run(f.ctx, key, ttl, func(context.Context) error {
					var expired bool
					if queryErr := f.db.QueryRowContext(f.ctx, `SELECT clock_timestamp() >= $1::timestamptz`, expires).Scan(&expired); queryErr != nil || !expired {
						t.Error("Run admitted callback before database TTL", expired, queryErr)
					}
					return nil
				})
				requireResult(t, true, executed, err)
				if successor, _ := f.row(t, key); successor == "" || successor == owner {
					t.Fatal("Run successor marker missing or owner reused", successor)
				}
			})
		}
	}
	t.Run("WithAutoRenew never participates in gates", func(t *testing.T) {
		executed, err := locker.TryRun(f.ctx, "long-callback", ttl, func(ctx context.Context) error {
			_, expires := f.row(t, "long-callback")
			f.waitExpired(t, expires)
			if ctx != f.ctx || ctx.Err() != nil {
				t.Error("gate TTL canceled original callback context", ctx.Err())
			}
			return nil
		})
		requireResult(t, true, executed, err)
		executed, err = locker.TryRun(f.ctx, "long-callback", ttl, func(context.Context) error { return nil })
		requireResult(t, true, executed, err)
	})
}

func TestIntegrationPostgresCoreLeaseAutoRenewAndRelease(t *testing.T) {
	f := realPostgres(t)
	locker, err := lock.New(f.driver, lock.WithAutoRenew())
	if err != nil {
		t.Fatal(err)
	}
	const ttl = 300 * time.Millisecond
	lease, err := locker.TryLock(f.ctx, "automatic", ttl)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = lease.Release(cleanup)
	})
	owner, originalExpiry := f.row(t, "automatic")
	waitPostgres(t, f.ctx, "automatic renewal extends PostgreSQL expiry", func() (bool, error) {
		var expires time.Time
		queryErr := f.db.QueryRowContext(f.ctx, `SELECT expires_at FROM `+f.quotedTable()+` WHERE key = $1`, f.prefix+"automatic").Scan(&expires)
		return expires.After(originalExpiry), queryErr
	})
	// Observe validity past the original remote expiry, not merely a worker tick.
	f.waitExpired(t, originalExpiry)
	select {
	case <-lease.Done():
		t.Fatal("auto-renew lease lost validity", lease.Err())
	default:
	}
	ok, err := f.driver.TryAcquire(f.ctx, "automatic", "competitor", ttl)
	requireResult(t, false, ok, err)
	if got, _ := f.row(t, "automatic"); got != owner {
		t.Fatal("competitor replaced renewed owner", got)
	}
	if releaseErr := lease.Release(f.ctx); releaseErr != nil {
		t.Fatal(releaseErr)
	}
	select {
	case <-lease.Done():
	default:
		t.Fatal("explicit Release did not close Done")
	}
	ok, err = f.driver.TryAcquire(f.ctx, "automatic", "successor", time.Second)
	requireResult(t, true, ok, err)
}
