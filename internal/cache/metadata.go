package cache

import (
	"time"

	"gorm.io/gorm/clause"
)

func (s *Store) GetMetadata(url string) ([]byte, bool) {
	var entry MetadataCache
	if s.db.Where("url = ? AND created_at > ?", url, cutoffUnix()).First(&entry).Error == nil {
		return entry.Data, true
	}
	return nil, false
}

func (s *Store) SetMetadata(url string, data []byte) {
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&MetadataCache{URL: url, Data: data, CreatedAt: time.Now().Unix()})
}
