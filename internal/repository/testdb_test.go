package repository_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"tixora/internal/models"
)

// sharedDB is a single MySQL container reused across every integration test
// in this package (started once in TestMain) - spinning up a fresh container
// per test would make the suite prohibitively slow. It stays nil (and every
// test skips itself via requireDB) when running with `go test -short` or
// when Docker isn't available, so `go test ./...` doesn't hard-depend on it.
var sharedDB *gorm.DB

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("tixora_test"),
		mysql.WithUsername("tixora"),
		mysql.WithPassword("tixora"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "repository integration tests: failed to start MySQL container (is Docker running?): %v\n", err)
		fmt.Fprintln(os.Stderr, "skipping integration tests - run `go test -short ./...` to skip explicitly next time")
		os.Exit(0)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "loc=Local")
	if err != nil {
		fmt.Fprintf(os.Stderr, "repository integration tests: failed to build DSN: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "repository integration tests: failed to connect: %v\n", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(
		&models.User{}, &models.Admin{}, &models.RefreshToken{},
		&models.Category{}, &models.File{}, &models.Event{},
		&models.Order{}, &models.Payment{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "repository integration tests: failed to migrate: %v\n", err)
		os.Exit(1)
	}

	sharedDB = db
	os.Exit(m.Run())
}

// requireDB returns a *gorm.DB scoped to a transaction that's rolled back
// when the test ends, giving each test a clean, isolated view of the schema
// without the cost of a fresh container or database per test. Tests call
// this first and get skipped automatically under `go test -short`.
func requireDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	if sharedDB == nil {
		t.Skip("no test database available (container failed to start - see TestMain output)")
	}

	tx := sharedDB.Begin()
	t.Cleanup(func() { tx.Rollback() })
	return tx
}
