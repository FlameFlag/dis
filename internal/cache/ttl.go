package cache

import "time"

func cutoffUnix() int64 {
	return time.Now().Add(-TTL).Unix()
}
