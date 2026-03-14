package cache

import (
	"time"

	"gorm.io/gorm/clause"
)

func (s *Store) GetWaveform(url string, numSamples int) ([]byte, bool) {
	var entry WaveformCache
	if s.db.Where("url = ? AND num_samples = ? AND created_at > ?", url, numSamples, cutoffUnix()).
		First(&entry).Error == nil {
		return entry.Data, true
	}
	return nil, false
}

func (s *Store) SetWaveform(url string, numSamples int, data []byte) {
	s.db.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&WaveformCache{URL: url, NumSamples: numSamples, Data: data, CreatedAt: time.Now().Unix()})
}
