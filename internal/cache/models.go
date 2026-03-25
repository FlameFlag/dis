package cache

import "time"

const TTL = 6 * time.Hour

type MetadataCache struct {
	URL       string `gorm:"primaryKey"`
	Data      []byte `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}

type TranscriptCache struct {
	VideoID   string `gorm:"primaryKey"`
	Data      []byte `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}

type SponsorBlockCache struct {
	VideoID   string `gorm:"primaryKey"`
	Data      []byte `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}
