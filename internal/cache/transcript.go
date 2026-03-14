package cache

import (
	"time"

	"gorm.io/gorm/clause"
)

func (s *Store) GetTranscript(videoID string) ([]byte, bool) {
	var entry TranscriptCache
	if s.db.Where("video_id = ? AND created_at > ?", videoID, cutoffUnix()).First(&entry).Error == nil {
		return entry.Data, true
	}
	return nil, false
}

func (s *Store) SetTranscript(videoID string, data []byte) {
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&TranscriptCache{VideoID: videoID, Data: data, CreatedAt: time.Now().Unix()})
}
