package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"strings"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

const (
	// CoreVersionTable keeps core migration history separate from future review migrations
	// when both roles share one PostgreSQL database.
	CoreVersionTable = "inkwords_core_schema_migrations"
	// ReviewVersionTable isolates review-service schema history when it shares
	// a database with core-api or uses its own review database.
	ReviewVersionTable = "inkwords_review_schema_migrations"

	// coreMigrationLockID prevents separate local service processes from applying the
	// same core migration concurrently during startup.
	coreMigrationLockID   int64 = 786549811
	reviewMigrationLockID int64 = 786549812
)

// NewCoreProvider builds an isolated provider over the embedded core migration assets.
// The provider deliberately ignores Goose's global migration registry so another package
// cannot affect local schema startup by registering a Go migration as a side effect.
func NewCoreProvider(database *sql.DB) (*goose.Provider, error) {
	return newProvider(database, "core", CoreVersionTable, coreMigrationLockID)
}

// NewReviewProvider builds a provider over only review-owned schema assets.
func NewReviewProvider(database *sql.DB) (*goose.Provider, error) {
	return newProvider(database, "review", ReviewVersionTable, reviewMigrationLockID)
}

func newProvider(database *sql.DB, role, versionTable string, lockID int64) (*goose.Provider, error) {
	if database == nil {
		return nil, fmt.Errorf("%s migration database is nil", role)
	}
	assets, err := assetsForRole(role)
	if err != nil {
		return nil, err
	}
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockID(lockID))
	if err != nil {
		return nil, fmt.Errorf("create %s migration lock: %w", role, err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		database,
		assets,
		goose.WithTableName(versionTable),
		goose.WithSessionLocker(locker),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return nil, fmt.Errorf("create %s migration provider: %w", role, err)
	}
	return provider, nil
}

func assetsForRole(role string) (fs.FS, error) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		return nil, fmt.Errorf("list migration assets: %w", err)
	}
	assets := fstest.MapFS{}
	marker := "-- InkWords migration role: " + role
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		contents, err := Files.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration asset %s: %w", entry.Name(), err)
		}
		if role == "core" {
			if strings.Contains(string(contents), "-- InkWords migration role: review") {
				continue
			}
		} else if !strings.Contains(string(contents), marker) {
			continue
		}
		assets[entry.Name()] = &fstest.MapFile{Data: contents, Mode: 0o444}
	}
	return assets, nil
}

// UpCore applies every pending core schema migration and reports only migrations applied now.
func UpCore(ctx context.Context, database *sql.DB) ([]*goose.MigrationResult, error) {
	provider, err := NewCoreProvider(database)
	if err != nil {
		return nil, err
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply core migrations: %w", err)
	}
	return results, nil
}

// DownCore rolls back the latest core migration. It is intended for recovery tests and
// explicitly safe development reversals; production rollback follows the ADR backup rules.
func DownCore(ctx context.Context, database *sql.DB) (*goose.MigrationResult, error) {
	provider, err := NewCoreProvider(database)
	if err != nil {
		return nil, err
	}

	result, err := provider.Down(ctx)
	if err != nil {
		return nil, fmt.Errorf("roll back core migration: %w", err)
	}
	return result, nil
}

// UpReview applies only review-service migrations.
func UpReview(ctx context.Context, database *sql.DB) ([]*goose.MigrationResult, error) {
	provider, err := NewReviewProvider(database)
	if err != nil {
		return nil, err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return nil, fmt.Errorf("apply review migrations: %w", err)
	}
	return results, nil
}
