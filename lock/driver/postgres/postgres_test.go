package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"math"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

var testContext = context.Background()

func mockDriver(t *testing.T, prefix string, options ...Option) (*Driver, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp), sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if expectationErr := mock.ExpectationsWereMet(); expectationErr != nil {
			t.Error(expectationErr)
		}
		_ = db.Close()
	})
	d, err := New(db, prefix, options...)
	if err != nil {
		t.Fatal(err)
	}
	return d, mock
}

// noIODatabase starts with no connections. Every database/sql request that could
// reach a backend must first call Connect. Count even rejected attempts, so an
// unexpected SQL error cannot masquerade as the expected validation error.
func noIODatabase(t *testing.T) *sql.DB {
	t.Helper()
	var connections atomic.Int64
	db := sql.OpenDB(&contextConnector{connect: func(context.Context) (driver.Conn, error) {
		connections.Add(1)
		return nil, errors.New("unexpected database access during validation")
	}})
	t.Cleanup(func() {
		_ = db.Close()
		if got := connections.Load(); got != 0 {
			t.Errorf("validation attempted database connection %d times; want zero", got)
		}
	})
	return db
}

func requireResult(t *testing.T, want, got bool, err error) {
	t.Helper()
	if err != nil || got != want {
		t.Fatalf("result = %v, %v; want %v, nil", got, err, want)
	}
}

func driverOperations(d *Driver) []struct {
	name string
	sql  string
	call func(context.Context, string, string, time.Duration) (bool, error)
} {
	return []struct {
		name string
		sql  string
		call func(context.Context, string, string, time.Duration) (bool, error)
	}{
		{"acquire", `(?i)INSERT\s+INTO`, d.TryAcquire},
		{"renew", `(?i)UPDATE\s+`, d.Renew},
		{"release", `(?i)DELETE\s+FROM`, func(ctx context.Context, key, owner string, _ time.Duration) (bool, error) {
			return d.Release(ctx, key, owner)
		}},
	}
}

func TestNewValidationAndNoDatabaseIO(t *testing.T) {
	if d, err := New(nil, ""); d != nil || err == nil {
		t.Fatalf("New(nil) = %v, %v", d, err)
	}
	db := noIODatabase(t)
	for _, prefix := range []string{"", "app:", "'; DROP TABLE users; --"} {
		if d, err := New(db, prefix); d == nil || err != nil {
			t.Fatalf("New(prefix %q) = %v, %v", prefix, d, err)
		}
	}
	for _, prefix := range []string{"nul\x00prefix", "bad\xffprefix"} {
		if d, err := New(db, prefix); d != nil || err == nil {
			t.Errorf("New(invalid prefix %q) = %v, %v", prefix, d, err)
		}
	}
	if d, err := New(db, "", nil); d != nil || err == nil {
		t.Fatalf("New(nil option) = %v, %v", d, err)
	}
	for _, table := range []string{"leases", "public.leases", "App.Leases_2"} {
		if d, err := New(db, "", WithTable(table)); d == nil || err != nil {
			t.Fatalf("New(valid table %q) = %v, %v", table, d, err)
		}
	}
	optionErr := errors.New("option failed without database access")
	if d, err := New(db, "", func(*Driver) error { return optionErr }); d != nil || !errors.Is(err, optionErr) {
		t.Fatalf("New(failing option) = %v, %v", d, err)
	}
	for _, table := range []string{
		"", ".leases", "public.", "a.b.c", "bad-name", "has space", "1leases", "public.1leases",
		"public.leases;DROP TABLE users", `public."leases"`, "é", "public.锁", "x\x00y", "a\nb",
		"a$b", "public.a$b", "pg_temp.leases", "pg_temp_123.leases",
		strings.Repeat("a", 64), strings.Repeat("s", 64) + ".leases", "public." + strings.Repeat("t", 64),
	} {
		if d, err := New(db, "", WithTable(table)); d != nil || err == nil {
			t.Errorf("New(table %q) = %v, %v; want nil, error", table, d, err)
		}
	}
}

func TestOptionsApplyInOrderAndPreserveErrors(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	optionErr := errors.New("option failed")
	if got, newErr := New(db, "", func(*Driver) error { return optionErr }); got != nil || !errors.Is(newErr, optionErr) {
		t.Fatalf("New(failing option) = %v, %v", got, newErr)
	}
	d, err := New(db, "", WithTable("first"), WithTable("public.second"))
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS "public"\."second"`).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := d.EnsureSchema(testContext); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestTableQualificationAndQuoting(t *testing.T) {
	for _, tc := range []struct {
		name, table, quoted string
	}{
		{"default", "", `"public"."pkg_lock_leases"`},
		{"unqualified", "leases", `"public"."leases"`},
		{"qualified", "App.Leases_2", `"App"."Leases_2"`},
		{"keyword", "select", `"public"."select"`},
		{"max identifiers", strings.Repeat("s", 63) + "." + strings.Repeat("t", 63), `"` + strings.Repeat("s", 63) + `"."` + strings.Repeat("t", 63) + `"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var options []Option
			if tc.table != "" {
				options = append(options, WithTable(tc.table))
			}
			d, mock := mockDriver(t, "", options...)
			mock.ExpectExec(`(?i)CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?` + regexp.QuoteMeta(tc.quoted)).WillReturnResult(sqlmock.NewResult(0, 0))
			if err := d.EnsureSchema(testContext); err != nil {
				t.Fatal(err)
			}
			for _, op := range driverOperations(d) {
				e := mock.ExpectExec(op.sql + `\s*` + regexp.QuoteMeta(tc.quoted))
				if op.name == "release" {
					e.WithArgs("key", "owner")
				} else {
					e.WithArgs("key", "owner", "0 hours 0 minutes 1 seconds 0 microseconds")
				}
				e.WillReturnResult(sqlmock.NewResult(0, 1))
				ok, err := op.call(testContext, "key", "owner", time.Second)
				requireResult(t, true, ok, err)
			}
		})
	}
}

func TestPrefixKeyAndOwnerAreBoundParameters(t *testing.T) {
	const prefix = "prefix'; DROP TABLE users; --:"
	const key = "key' OR true --"
	const owner = "owner'; DELETE FROM users; --"
	d, mock := mockDriver(t, prefix)
	for _, op := range driverOperations(d) {
		e := mock.ExpectExec(op.sql)
		if op.name == "release" {
			e.WithArgs(prefix+key, owner)
		} else {
			e.WithArgs(prefix+key, owner, "0 hours 0 minutes 1 seconds 0 microseconds")
		}
		e.WillReturnResult(sqlmock.NewResult(0, 1))
		ok, err := op.call(testContext, key, owner, time.Second)
		requireResult(t, true, ok, err)
	}
}

func TestTTLUsesExactCeilingMicrosecondStrings(t *testing.T) {
	for _, tc := range []struct {
		ttl  time.Duration
		want string
	}{
		{time.Nanosecond, "0 hours 0 minutes 0 seconds 1 microseconds"},
		{time.Microsecond - 1, "0 hours 0 minutes 0 seconds 1 microseconds"},
		{time.Microsecond, "0 hours 0 minutes 0 seconds 1 microseconds"},
		{time.Microsecond + 1, "0 hours 0 minutes 0 seconds 2 microseconds"},
		{time.Second + 1, "0 hours 0 minutes 1 seconds 1 microseconds"},
		{time.Hour + 2*time.Minute + 3*time.Second + 4*time.Microsecond + 1, "1 hours 2 minutes 3 seconds 5 microseconds"},
		{time.Duration(math.MaxInt64), "2562047 hours 47 minutes 16 seconds 854776 microseconds"},
	} {
		t.Run(tc.ttl.String(), func(t *testing.T) {
			d, mock := mockDriver(t, "ttl:")
			for _, op := range driverOperations(d)[:2] {
				// The interval must be bound as text, not a float64 or a rounded duration.
				mock.ExpectExec(op.sql+`(?s).*\$3::(pg_catalog\.)?interval`).WithArgs("ttl:key", "owner", tc.want).WillReturnResult(sqlmock.NewResult(0, 1))
				ok, err := op.call(testContext, "key", "owner", tc.ttl)
				requireResult(t, true, ok, err)
			}
		})
	}
}

func TestInputValidation(t *testing.T) {
	d, newErr := New(noIODatabase(t), "prefix:")
	if newErr != nil {
		t.Fatal(newErr)
	}
	for _, op := range driverOperations(d) {
		t.Run(op.name, func(t *testing.T) {
			for _, pair := range [][2]string{{"", "owner"}, {"key", ""}, {"", ""}, {"nul\x00key", "owner"}, {"key", "nul\x00owner"}, {"bad\xffkey", "owner"}, {"key", "bad\xffowner"}} {
				if ok, err := op.call(testContext, pair[0], pair[1], time.Second); ok || err == nil {
					t.Fatalf("invalid key/owner = %v, %v", ok, err)
				}
			}
			if ok, err := op.call(nil, "key", "owner", time.Second); ok || err == nil {
				t.Fatalf("nil context = %v, %v", ok, err)
			}
			for _, ttl := range []time.Duration{0, -1, time.Duration(math.MinInt64)} {
				if op.name != "release" {
					if ok, err := op.call(testContext, "key", "owner", ttl); ok || err == nil {
						t.Fatalf("invalid TTL %v = %v, %v", ttl, ok, err)
					}
				}
			}
		})
	}
	var nilContext context.Context
	if err := d.EnsureSchema(nilContext); err == nil {
		t.Fatal("EnsureSchema(nil) succeeded")
	}
	for _, d := range []*Driver{nil, {}} {
		for _, op := range driverOperations(d) {
			if ok, err := op.call(testContext, "key", "owner", time.Second); ok || err == nil {
				t.Fatalf("%s nil/zero driver = %v, %v", op.name, ok, err)
			}
		}
		if err := d.EnsureSchema(testContext); err == nil {
			t.Fatal("nil/zero driver EnsureSchema succeeded")
		}
	}
}

func TestPreCancelledContextsDoNotReachDatabase(t *testing.T) {
	d, newErr := New(noIODatabase(t), "")
	if newErr != nil {
		t.Fatal(newErr)
	}
	cancelled, cancel := context.WithCancel(testContext)
	cancel()
	expired, stop := context.WithDeadline(testContext, time.Now().Add(-time.Second))
	defer stop()
	for _, ctx := range []context.Context{cancelled, expired} {
		for _, op := range driverOperations(d) {
			if ok, err := op.call(ctx, "key", "owner", time.Second); ok || !errors.Is(err, ctx.Err()) {
				t.Errorf("%s pre-cancelled = %v, %v; want %v", op.name, ok, err, ctx.Err())
			}
		}
		if err := d.EnsureSchema(ctx); !errors.Is(err, ctx.Err()) {
			t.Errorf("EnsureSchema pre-cancelled = %v; want %v", err, ctx.Err())
		}
	}
}

func TestRowsAffectedAndDatabaseErrors(t *testing.T) {
	backendErr := errors.New("backend failed")
	for _, tc := range []struct {
		name   string
		result driver.Result
		err    error
		want   bool
		bad    bool
	}{
		{"zero", sqlmock.NewResult(0, 0), nil, false, false},
		{"one", sqlmock.NewResult(0, 1), nil, true, false},
		{"two", sqlmock.NewResult(0, 2), nil, false, true},
		{"negative", sqlmock.NewResult(0, -1), nil, false, true},
		{"rows affected error", sqlmock.NewErrorResult(backendErr), nil, false, true},
		{"backend error", nil, backendErr, false, true},
		{"sql no rows", nil, sql.ErrNoRows, false, true},
		{"sql connection done", nil, sql.ErrConnDone, false, true},
		{"sql transaction done", nil, sql.ErrTxDone, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, mock := mockDriver(t, "")
			for _, op := range driverOperations(d) {
				e := mock.ExpectExec(op.sql)
				if tc.err != nil {
					e.WillReturnError(tc.err)
				} else {
					e.WillReturnResult(tc.result)
				}
				ok, err := op.call(testContext, "key", "owner", time.Second)
				if tc.bad {
					if ok || err == nil {
						t.Errorf("%s invalid result = %v, %v", op.name, ok, err)
					}
					wantErr := tc.err
					if tc.name == "rows affected error" {
						wantErr = backendErr
					}
					if wantErr != nil && !errors.Is(err, wantErr) {
						t.Errorf("%s error %v does not wrap %v", op.name, err, wantErr)
					}
				} else {
					requireResult(t, tc.want, ok, err)
				}
			}
		})
	}
	d, mock := mockDriver(t, "")
	mock.ExpectExec(`(?i)CREATE\s+TABLE`).WillReturnError(backendErr)
	if err := d.EnsureSchema(testContext); !errors.Is(err, backendErr) {
		t.Fatalf("EnsureSchema error = %v; want backend error", err)
	}
}

// contextConnector observes the actual database/sql transport boundary. sqlmock
// validates statements elsewhere but intentionally does not expose their context.
type contextConnector struct {
	connect func(context.Context) (driver.Conn, error)
	exec    func(context.Context) (driver.Result, error)
}

type contextConnection struct{ connector *contextConnector }

type contextSQLDriver struct{}

func (c *contextConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if c.connect != nil {
		return c.connect(ctx)
	}
	return &contextConnection{connector: c}, nil
}
func (*contextConnector) Driver() driver.Driver { return contextSQLDriver{} }
func (contextSQLDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use connector")
}
func (*contextConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected Prepare")
}
func (*contextConnection) Close() error { return nil }
func (*contextConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("unexpected transaction")
}
func (c *contextConnection) ExecContext(ctx context.Context, _ string, _ []driver.NamedValue) (driver.Result, error) {
	return c.connector.exec(ctx)
}

func TestOperationContextsReachTransport(t *testing.T) {
	type contextKey struct{}
	valueCtx := context.WithValue(testContext, contextKey{}, "preserved")
	for _, withDeadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "default one second", true: "caller deadline"}[withDeadline], func(t *testing.T) {
			ctx := valueCtx
			if withDeadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
			}
			var observed context.Context
			db := sql.OpenDB(&contextConnector{exec: func(got context.Context) (driver.Result, error) {
				observed = got
				deadline, exists := got.Deadline()
				if !exists || time.Until(deadline) <= 0 {
					t.Error("transport context has no live deadline")
				}
				if withDeadline {
					want, _ := ctx.Deadline()
					if !deadline.Equal(want) {
						t.Errorf("deadline = %v; want caller deadline %v", deadline, want)
					}
				} else if remaining := time.Until(deadline); remaining > time.Second || remaining < 800*time.Millisecond {
					t.Errorf("fallback deadline remaining = %v; want approximately 1s", remaining)
				}
				if got.Value(contextKey{}) != "preserved" {
					t.Error("context value lost")
				}
				return driver.RowsAffected(1), nil
			}})
			defer db.Close()
			d, err := New(db, "")
			if err != nil {
				t.Fatal(err)
			}
			calls := []func() error{func() error { return d.EnsureSchema(ctx) }}
			for _, op := range driverOperations(d) {
				calls = append(calls, func() error { _, err := op.call(ctx, "key", "owner", time.Second); return err })
			}
			for _, call := range calls {
				observed = nil
				if err := call(); err != nil {
					t.Fatal(err)
				}
				if observed == nil {
					t.Fatal("operation did not reach transport")
				}
				if !withDeadline && !errors.Is(observed.Err(), context.Canceled) {
					t.Error("fallback context not canceled after operation")
				}
				if ctx.Err() != nil {
					t.Fatal("operation canceled caller context")
				}
			}
		})
	}
}

// A deadline timer can lag the wall clock. This context lets the transport move
// the deadline into the past while Err still reports nil, without timer races.
type clockDeadlineContext struct {
	context.Context
	deadline time.Time
}

func (c *clockDeadlineContext) Deadline() (time.Time, bool) { return c.deadline, true }

func TestLateSuccessfulResponsesAreNotReportedAsSuccess(t *testing.T) {
	for _, mode := range []string{"canceled", "clock deadline passed"} {
		for _, name := range []string{"acquire", "renew", "release", "schema"} {
			t.Run(mode+"/"+name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(testContext)
				defer cancel()
				request := ctx
				clockCtx := &clockDeadlineContext{Context: ctx, deadline: time.Now().Add(time.Second)}
				if mode == "clock deadline passed" {
					request = clockCtx
				}
				db := sql.OpenDB(&contextConnector{exec: func(context.Context) (driver.Result, error) {
					if mode == "canceled" {
						cancel()
					} else {
						clockCtx.deadline = time.Now().Add(-time.Second)
					}
					// The remote mutation may have happened; local cancellation must
					// still be an error, not (true, nil) or ordinary contention.
					return driver.RowsAffected(1), nil
				}})
				defer db.Close()
				d, err := New(db, "")
				if err != nil {
					t.Fatal(err)
				}
				ok := false
				if name == "schema" {
					err = d.EnsureSchema(request)
				} else {
					for _, op := range driverOperations(d) {
						if op.name == name {
							ok, err = op.call(request, "key", "owner", time.Second)
						}
					}
				}
				wantErr := context.Canceled
				if mode != "canceled" {
					wantErr = context.DeadlineExceeded
				}
				if ok || !errors.Is(err, wantErr) {
					t.Fatalf("late response = %v, %v; want false, %v", ok, err, wantErr)
				}
			})
		}
	}
}

func TestInFlightDeadlineCancelsDatabaseOperation(t *testing.T) {
	for _, name := range []string{"acquire", "renew", "release", "schema"} {
		t.Run(name, func(t *testing.T) {
			d, mock := mockDriver(t, "")
			pattern := `(?i)CREATE\s+TABLE`
			for _, op := range driverOperations(d) {
				if op.name == name {
					pattern = op.sql
				}
			}
			mock.ExpectExec(pattern).WillDelayFor(time.Second).WillReturnResult(sqlmock.NewResult(0, 1))
			ctx, cancel := context.WithTimeout(testContext, 20*time.Millisecond)
			defer cancel()
			started := time.Now()
			var err error
			if name == "schema" {
				err = d.EnsureSchema(ctx)
			} else {
				for _, op := range driverOperations(d) {
					if op.name == name {
						_, err = op.call(ctx, "key", "owner", time.Second)
					}
				}
			}
			// sqlmock's transport reports ErrCancelled, unlike lib/pq; either
			// that wrapped transport error or normalized context error is valid.
			if err == nil || (!errors.Is(err, sqlmock.ErrCancelled) && !errors.Is(err, context.DeadlineExceeded)) {
				t.Fatalf("in-flight deadline error = %v", err)
			}
			if time.Since(started) > 500*time.Millisecond {
				t.Fatal("caller deadline did not promptly interrupt transport")
			}
		})
	}
}
