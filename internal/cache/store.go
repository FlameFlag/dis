package cache

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	migrateOnce sync.Once
	expireOnce  sync.Once
)

// Store is a typed cache backed by GORM + SQLite.
type Store struct{ db *gorm.DB }

// Open opens (or creates) the cache database at ~/.cache/dis/cache.db.
func Open() (*Store, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "dis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	dsn := filepath.Join(dir, "cache.db") +
		"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	var migrateErr error
	migrateOnce.Do(func() {
		migrateErr = db.AutoMigrate(
			&MetadataCache{}, &TranscriptCache{},
			&SponsorBlockCache{},
		)
	})
	if migrateErr != nil {
		return nil, migrateErr
	}

	return &Store{db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DeleteExpired removes stale entries from all tables. Runs at most once per process.
func (s *Store) DeleteExpired() {
	expireOnce.Do(func() {
		cutoff := cutoffUnix()
		s.db.Where("created_at <= ?", cutoff).Delete(&MetadataCache{})
		s.db.Where("created_at <= ?", cutoff).Delete(&TranscriptCache{})
		s.db.Where("created_at <= ?", cutoff).Delete(&SponsorBlockCache{})
	})
}
