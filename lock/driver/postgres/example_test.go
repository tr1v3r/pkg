package postgres_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	postgresdriver "github.com/tr1v3r/pkg/lock/driver/postgres"
)

// exampleDatabase is a test-only setup: examples need no PostgreSQL server.
// sqlmock checks API usage, not PostgreSQL ownership/expiry semantics; those are
// verified by the integration suite against a real PostgreSQL server.
// In applications, use an application-owned sql.Open("pgx", dsn) pool after
// importing github.com/jackc/pgx/v5/stdlib for registration and context support.
func exampleDatabase() (*sql.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	return db, mock, func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			panic(err)
		}
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			panic(err)
		}
	}
}

func exampleDriver() (*postgresdriver.Driver, sqlmock.Sqlmock, func()) {
	db, mock, cleanup := exampleDatabase()
	d, err := postgresdriver.New(db, "myapp:leases:")
	if err != nil {
		panic(err)
	}
	return d, mock, cleanup
}

func ExampleNew() {
	db, _, cleanup := exampleDatabase()
	defer cleanup()
	// Include a separator: prefix is prepended verbatim to each logical key.
	d, err := postgresdriver.New(db, "myapp:leases:")
	fmt.Println("driver ready:", d != nil, "error:", err)
	// New neither connects nor creates tables nor owns the caller's pool.
	// Call EnsureSchema explicitly before first use (or provision the table).

	// Output:
	// driver ready: true error: <nil>
}

func ExampleWithTable() {
	db, _, cleanup := exampleDatabase()
	defer cleanup()
	// The schema must already exist. A bare table name uses public; the default
	// is public.pkg_lock_leases. Each ASCII identifier is validated and quoted.
	d, err := postgresdriver.New(db, "myapp:", postgresdriver.WithTable("public.app_leases"))
	fmt.Println("custom table ready:", d != nil, "error:", err)
	// Output:
	// custom table ready: true error: <nil>
}

func ExampleDriver_EnsureSchema() {
	d, mock, cleanup := exampleDriver()
	defer cleanup()
	mock.ExpectExec(`CREATE TABLE`).WillReturnResult(sqlmock.NewResult(0, 0))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Explicit, idempotent table initialization; does not create the schema.
	fmt.Println("ensure table:", d.EnsureSchema(ctx))
	// Output:
	// ensure table: <nil>
}

func ExampleDriver_TryAcquire() {
	d, mock, cleanup := exampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Each new acquisition attempt needs a fresh owner token. lock.Locker
	// normally manages tokens and the lease lifecycle for applications.
	owner := rand.Text()
	mock.ExpectExec(`INSERT INTO`).WithArgs("myapp:leases:resource:123", owner, "0 hours 1 minutes 0 seconds 0 microseconds").WillReturnResult(sqlmock.NewResult(0, 1))
	acquired, err := d.TryAcquire(ctx, "resource:123", owner, time.Minute)
	fmt.Println("acquired:", acquired, "error:", err)
	// Even the same owner cannot reacquire a live lease; use Renew instead.
	mock.ExpectExec(`INSERT INTO`).WithArgs("myapp:leases:resource:123", owner, "0 hours 1 minutes 0 seconds 0 microseconds").WillReturnResult(sqlmock.NewResult(0, 0))
	acquired, err = d.TryAcquire(ctx, "resource:123", owner, time.Minute)
	fmt.Println("busy:", !acquired, "error:", err)
	// Output:
	// acquired: true error: <nil>
	// busy: true error: <nil>
}

func ExampleDriver_Renew() {
	d, mock, cleanup := exampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Assume this fresh owner previously acquired the lease successfully.
	owner := rand.Text()
	mock.ExpectExec(`UPDATE`).WithArgs("myapp:leases:resource:123", "another-owner", "0 hours 1 minutes 0 seconds 0 microseconds").WillReturnResult(sqlmock.NewResult(0, 0))
	renewed, err := d.Renew(ctx, "resource:123", "another-owner", time.Minute)
	fmt.Println("other owner renewed:", renewed, "error:", err)
	mock.ExpectExec(`UPDATE`).WithArgs("myapp:leases:resource:123", owner, "0 hours 1 minutes 0 seconds 0 microseconds").WillReturnResult(sqlmock.NewResult(0, 1))
	renewed, err = d.Renew(ctx, "resource:123", owner, time.Minute)
	fmt.Println("current owner renewed:", renewed, "error:", err)
	// An expired owner cannot revive a lease, even without a successor.

	// Output:
	// other owner renewed: false error: <nil>
	// current owner renewed: true error: <nil>
}

func ExampleDriver_Release() {
	d, mock, cleanup := exampleDriver()
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	owner := rand.Text()
	mock.ExpectExec(`DELETE FROM`).WithArgs("myapp:leases:resource:123", "another-owner").WillReturnResult(sqlmock.NewResult(0, 0))
	released, err := d.Release(ctx, "resource:123", "another-owner")
	fmt.Println("other owner released:", released, "error:", err)
	mock.ExpectExec(`DELETE FROM`).WithArgs("myapp:leases:resource:123", owner).WillReturnResult(sqlmock.NewResult(0, 1))
	released, err = d.Release(ctx, "resource:123", owner)
	fmt.Println("current owner released:", released, "error:", err)
	// Release affects only this live owner; it never closes the caller's DB.

	// Output:
	// other owner released: false error: <nil>
	// current owner released: true error: <nil>
}
