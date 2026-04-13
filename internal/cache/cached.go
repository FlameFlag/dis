package cache

import "encoding/json"

// FetchCached is a generic read-through cache helper.
// It tries to load a cached JSON blob using get; on miss it calls fetch,
// then stores the result with set.  Both cache operations are best-effort:
// a cache miss or store failure never blocks the caller.
func FetchCached[T any](
	key string,
	get func(*Store, string) ([]byte, bool),
	set func(*Store, string, []byte),
	fetch func() (T, error),
) (T, error) {
	if store, ok := TryOpen(); ok {
		defer func() { _ = store.Close() }()
		store.DeleteExpired()
		if data, ok := get(store, key); ok {
			var v T
			if json.Unmarshal(data, &v) == nil {
				return v, nil
			}
		}
	}

	v, err := fetch()
	if err != nil {
		return v, err
	}

	if store, ok := TryOpen(); ok {
		defer func() { _ = store.Close() }()
		if blob, err := json.Marshal(v); err == nil {
			set(store, key, blob)
		}
	}
	return v, nil
}
