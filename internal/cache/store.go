package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"go.etcd.io/bbolt"
)

const TTL = 6 * time.Hour

var (
	bucketMetadata     = []byte("metadata")
	bucketTranscript   = []byte("transcript")
	bucketSponsorBlock = []byte("sponsorblock")

	allBuckets = [][]byte{bucketMetadata, bucketTranscript, bucketSponsorBlock}

	expireOnce sync.Once
)

type entry struct {
	Data      []byte `json:"d"`
	CreatedAt int64  `json:"t"`
}

// Store is a typed cache backed by bbolt.
type Store struct{ db *bbolt.DB }

// TryOpen opens the cache, returning the store and true on success.
// On failure it logs a debug message and returns nil, false.
func TryOpen() (*Store, bool) {
	s, err := Open()
	if err != nil {
		log.Debug("cache unavailable", "err", err)
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
	dir = filepath.Join(dir, "dis")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	db, err := bbolt.Open(filepath.Join(dir, "cache.bolt"), 0o644, &bbolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range allBuckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
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

// DeleteExpired removes stale entries from all buckets. Runs at most once per process.
func (s *Store) DeleteExpired() {
	expireOnce.Do(func() {
		cutoff := time.Now().Add(-TTL).Unix()
		_ = s.db.Update(func(tx *bbolt.Tx) error {
			for _, name := range allBuckets {
				b := tx.Bucket(name)
				if b == nil {
					continue
				}
				c := b.Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					var e entry
					if json.Unmarshal(v, &e) != nil || e.CreatedAt <= cutoff {
						_ = c.Delete()
					}
				}
			}
			return nil
		})
	})
}

func get(s *Store, bucket []byte, key string) ([]byte, bool) {
	var data []byte
	_ = s.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(bucket).Get([]byte(key))
		if v == nil {
			return nil
		}
		var e entry
		if err := json.Unmarshal(v, &e); err != nil {
			return nil
		}
		if e.CreatedAt <= time.Now().Add(-TTL).Unix() {
			return nil
		}
		data = e.Data
		return nil
	})
	return data, data != nil
}

func set(s *Store, bucket []byte, key string, data []byte) {
	_ = s.db.Update(func(tx *bbolt.Tx) error {
		raw, err := json.Marshal(entry{Data: data, CreatedAt: time.Now().Unix()})
		if err != nil {
			return err
		}
		return tx.Bucket(bucket).Put([]byte(key), raw)
	})
}

func (s *Store) GetMetadata(key string) ([]byte, bool)    { return get(s, bucketMetadata, key) }
func (s *Store) SetMetadata(key string, data []byte)       { set(s, bucketMetadata, key, data) }
func (s *Store) GetTranscript(key string) ([]byte, bool)   { return get(s, bucketTranscript, key) }
func (s *Store) SetTranscript(key string, data []byte)     { set(s, bucketTranscript, key, data) }
func (s *Store) GetSponsorBlock(key string) ([]byte, bool) { return get(s, bucketSponsorBlock, key) }
func (s *Store) SetSponsorBlock(key string, data []byte)   { set(s, bucketSponsorBlock, key, data) }
