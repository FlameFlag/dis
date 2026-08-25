package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

const (
	TTL                            = 6 * time.Hour
	cacheDirectory                 = "dis"
	cacheFilename                  = "cache.bolt"
	cacheDirectoryMode os.FileMode = 0o700
	cacheFileMode      os.FileMode = 0o600
	cacheOpenTimeout               = time.Second
)

// Bucket identifies a typed cache namespace.
type Bucket string

const (
	Metadata     Bucket = "metadata"
	Transcript   Bucket = "transcript"
	SponsorBlock Bucket = "sponsorblock"
	Storyboard   Bucket = "storyboard"
)

var allBuckets = []Bucket{Metadata, Transcript, SponsorBlock, Storyboard}

// entry retains the original on-disk envelope. Data contains a second JSON
// document so databases written before Store became generic remain readable.
type entry struct {
	Data      []byte    `json:"d"`
	CreatedAt time.Time `json:"t"`
}

// Store is a typed cache backed by bbolt.
type Store struct {
	db         *bbolt.DB
	expireOnce sync.Once
}

// TryOpen opens the cache, returning the store and true on success.
// On failure it returns nil, false.
func TryOpen() (*Store, bool) {
	s, err := Open()
	if err != nil {
		return nil, false
	}
	return s, true
}

// Open opens (or creates) the cache database at ~/.cache/dis/cache.bolt.
func Open() (*Store, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, cacheDirectory)
	if err := os.MkdirAll(dir, cacheDirectoryMode); err != nil {
		return nil, err
	}

	db, err := bbolt.Open(filepath.Join(dir, cacheFilename), cacheFileMode, &bbolt.Options{
		Timeout: cacheOpenTimeout,
	})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range allBuckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(name)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// DeleteExpired removes stale entries from all buckets. It runs once per store.
func (s *Store) DeleteExpired() {
	s.expireOnce.Do(func() {
		_ = s.db.Update(func(tx *bbolt.Tx) error {
			for _, name := range allBuckets {
				b := tx.Bucket([]byte(name))
				if b == nil {
					continue
				}
				c := b.Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					var e entry
					if json.Unmarshal(v, &e) != nil || time.Since(e.CreatedAt) > TTL {
						_ = c.Delete()
					}
				}
			}
			return nil
		})
	})
}

// Get retrieves and decodes a value from a bucket.
func (s *Store) Get[T any](bucket Bucket, key string) (value T, ok bool) {
	_ = s.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket([]byte(bucket)).Get([]byte(key))
		if v == nil {
			return nil
		}
		var e entry
		if err := json.Unmarshal(v, &e); err != nil {
			return nil
		}
		if time.Since(e.CreatedAt) > TTL {
			return nil
		}
		if err := json.Unmarshal(e.Data, &value); err != nil {
			return nil
		}
		ok = true
		return nil
	})
	return value, ok
}

// Set encodes and stores a value in a bucket.
func (s *Store) Set[T any](bucket Bucket, key string, value T) {
	_ = s.db.Update(func(tx *bbolt.Tx) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(entry{Data: data, CreatedAt: time.Now()})
		if err != nil {
			return err
		}
		return tx.Bucket([]byte(bucket)).Put([]byte(key), raw)
	})
}
