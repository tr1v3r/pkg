// Package postgres implements PostgreSQL 12+ owner-checked lease operations for
// package lock using database/sql. Callers own the pool and choose a PostgreSQL
// SQL driver that honors operation contexts and network deadlines. database/sql
// cannot impose a deadline on a SQL driver that ignores cancellation.
//
// New neither connects nor runs DDL. Use EnsureSchema explicitly, or provision
// the documented table through migrations. Expiry uses the database wall clock;
// expired rows are logically absent and need not be deleted before acquisition.
// Maintenance/cleanup of unused expired rows is the caller's responsibility.
//
// Mutations are single atomic statements. Renewal/release lock the row before
// checking its owner and expiry, so waiting for a row lock cannot revive an
// expired lease. TTLs round up to microseconds. The driver does not start workers,
// close connections, strengthen database failover guarantees, or provide fencing.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tr1v3r/pkg/lock"
)

const createSQL = `CREATE TABLE IF NOT EXISTS %s (
    key text COLLATE pg_catalog."C" PRIMARY KEY,
    owner text COLLATE pg_catalog."C" NOT NULL,
    expires_at timestamp with time zone NOT NULL
)`

const acquireSQL = `INSERT INTO %s AS existing (key, owner, expires_at)
VALUES ($1, $2, pg_catalog.clock_timestamp() + $3::pg_catalog.interval)
ON CONFLICT (key) DO UPDATE
SET owner = EXCLUDED.owner, expires_at = pg_catalog.clock_timestamp() + $3::pg_catalog.interval
WHERE existing.expires_at <= pg_catalog.clock_timestamp()`

// The outer predicates deliberately use held's columns: MATERIALIZED prevents
// them from being pushed below FOR UPDATE and evaluated before a row-lock wait.
const renewSQL = `WITH held AS MATERIALIZED (
    SELECT key, owner, expires_at FROM %s WHERE key = $1 FOR UPDATE
)
UPDATE %[1]s AS target
SET expires_at = pg_catalog.clock_timestamp() + $3::pg_catalog.interval
FROM held
WHERE target.key = held.key AND held.owner = $2 AND held.expires_at > pg_catalog.clock_timestamp()`

const releaseSQL = `WITH held AS MATERIALIZED (
    SELECT key, owner, expires_at FROM %s WHERE key = $1 FOR UPDATE
)
DELETE FROM %[1]s AS target USING held
WHERE target.key = held.key AND held.owner = $2 AND held.expires_at > pg_catalog.clock_timestamp()`

// Driver implements lock.Driver over a shared permanent PostgreSQL table.
// Construct it with New; its zero value is not usable. It is safe for concurrent
// use. Only *sql.DB is accepted, not an externally managed long-lived transaction.
type Driver struct {
	db     *sql.DB
	prefix string
	table  string
}

var _ lock.Driver = (*Driver)(nil)

// Option configures a Driver before it is shared.
type Option func(*Driver) error

// WithTable selects a permanent table, either table or schema.table. Unqualified
// names use public, not search_path. Each component must be an ASCII identifier
// (letter/underscore followed by letters/digits/underscores), at most 63 bytes.
// Names are quoted and case-sensitive; the schema must already exist.
func WithTable(name string) Option {
	return func(d *Driver) error {
		table, err := quoteTable(name)
		if err != nil {
			return err
		}
		d.table = table
		return nil
	}
}

// New constructs a driver without connecting or creating schema. The default
// table is public.pkg_lock_leases. Prefix is prepended verbatim to logical keys;
// reserve a namespace and include separators as desired. Nil pools/options and
// prefixes containing NUL or invalid UTF-8 are rejected. Callers retain pool ownership.
func New(db *sql.DB, prefix string, options ...Option) (*Driver, error) {
	if db == nil {
		return nil, errors.New("lock/postgres: nil database")
	}
	if !validText(prefix) {
		return nil, errors.New("lock/postgres: prefix must be valid UTF-8 without NUL")
	}
	d := &Driver{db: db, prefix: prefix, table: `"public"."pkg_lock_leases"`}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("lock/postgres: nil option")
		}
		if err := option(d); err != nil {
			return nil, err
		}
	}
	return d, nil
}

// EnsureSchema explicitly creates the table if absent. It neither creates a
// schema nor migrates/validates an existing table. Prefer a serialized migration
// for production startup; concurrent CREATE TABLE IF NOT EXISTS can still race
// in PostgreSQL's catalogs. The table must not have triggers or external writers
// that change the lock protocol. Like other calls, this operation is bounded.
func (d *Driver) EnsureSchema(ctx context.Context) error {
	if err := d.validateContext(ctx); err != nil {
		return err
	}
	_, err := d.exec(ctx, "schema", fmt.Sprintf(createSQL, d.table))
	return err
}

// TryAcquire atomically reserves an absent/expired key. An unexpired row is busy
// even if its owner matches. Data values are bound parameters, never SQL text.
func (d *Driver) TryAcquire(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	interval, err := ttlInterval(ttl)
	if err != nil {
		return false, err
	}
	return d.exec(ctx, "acquire", fmt.Sprintf(acquireSQL, d.table), d.prefix+key, owner, interval)
}

// Renew atomically extends only the unexpired matching owner by ttl, measured
// using the database wall clock. A row-lock wait is followed by a fresh expiry
// check, not one cached at statement/transaction start. False means ownership loss.
func (d *Driver) Renew(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	interval, err := ttlInterval(ttl)
	if err != nil {
		return false, err
	}
	return d.exec(ctx, "renew", fmt.Sprintf(renewSQL, d.table), d.prefix+key, owner, interval)
}

// Release atomically deletes only the unexpired matching owner's row. Expired,
// absent and successor-owned rows return false and are not removed.
func (d *Driver) Release(ctx context.Context, key, owner string) (bool, error) {
	if err := d.validate(ctx, key, owner); err != nil {
		return false, err
	}
	return d.exec(ctx, "release", fmt.Sprintf(releaseSQL, d.table), d.prefix+key, owner)
}

func (d *Driver) exec(ctx context.Context, operation, query string, args ...any) (bool, error) {
	request, cancel := operationContext(ctx)
	defer cancel()
	result, err := d.db.ExecContext(request, query, args...)
	if operationErr := requestError(request); err != nil || operationErr != nil {
		return false, fmt.Errorf("lock/postgres: %s: %w", operation, errors.Join(err, operationErr))
	}
	if result == nil {
		return false, fmt.Errorf("lock/postgres: %s: nil result", operation)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("lock/postgres: %s: rows affected: %w", operation, err)
	}
	if count < 0 || count > 1 {
		return false, fmt.Errorf("lock/postgres: %s: unexpected rows affected: %d", operation, count)
	}
	return count == 1, nil
}

func (d *Driver) validateContext(ctx context.Context) error {
	if d == nil || d.db == nil {
		return errors.New("lock/postgres: nil driver or database")
	}
	if ctx == nil {
		return errors.New("lock/postgres: nil context")
	}
	return ctx.Err()
}

func (d *Driver) validate(ctx context.Context, key, owner string) error {
	if err := d.validateContext(ctx); err != nil {
		return err
	}
	if key == "" || owner == "" || !validText(key) || !validText(owner) {
		return errors.New("lock/postgres: key and owner must be nonempty UTF-8 without NUL")
	}
	return nil
}

func operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Second)
}

func requestError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return context.DeadlineExceeded
	}
	return nil
}

func validText(value string) bool { return utf8.ValidString(value) && !strings.ContainsRune(value, 0) }

func quoteTable(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		parts = []string{"public", parts[0]}
	}
	if len(parts) != 2 {
		return "", errors.New("lock/postgres: table must be table or schema.table")
	}
	for _, part := range parts {
		if !validIdentifier(part) {
			return "", errors.New("lock/postgres: invalid table identifier")
		}
	}
	if parts[0] == "pg_temp" || strings.HasPrefix(parts[0], "pg_temp_") {
		return "", errors.New("lock/postgres: temporary schemas are not shared lease storage")
	}
	return `"` + parts[0] + `"."` + parts[1] + `"`, nil
}

func validIdentifier(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}
	for i := range len(name) {
		c := name[i]
		letter := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
		digit := i > 0 && c >= '0' && c <= '9'
		if !letter && !digit {
			return false
		}
	}
	return true
}

func ttlInterval(ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", errors.New("lock/postgres: TTL must be positive")
	}
	microseconds := int64(ttl / time.Microsecond)
	if ttl%time.Microsecond != 0 {
		microseconds++
	}
	// PostgreSQL 12 interval text fields are int32. Split even extreme durations
	// into small integer units, without floating point or calendar-day/DST math.
	hours := microseconds / 3_600_000_000
	minutes := microseconds / 60_000_000 % 60
	seconds := microseconds / 1_000_000 % 60
	return fmt.Sprintf("%d hours %d minutes %d seconds %d microseconds",
		hours, minutes, seconds, microseconds%1_000_000), nil
}
