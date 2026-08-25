package cache

// FetchFrom is a best-effort read-through cache using an injected store. A nil
// store disables caching.
func FetchFrom[T any](
	store *Store,
	bucket Bucket,
	key string,
	fetch func() (T, error),
) (T, error) {
	if store != nil {
		if value, ok := store.Get[T](bucket, key); ok {
			return value, nil
		}
	}
	value, err := fetch()
	if err == nil && store != nil {
		store.Set(bucket, key, value)
	}
	return value, err
}
