package browsercookies

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/browserutils/kooky"
)

// File is a temporary cookie jar owned by Close.
type File struct {
	path      string
	closeOnce sync.Once
	closeErr  error
}

// Path returns the temporary Netscape cookie jar path.
func (f *File) Path() string {
	if f == nil {
		return ""
	}
	return f.path
}

// Close removes the temporary cookie jar. It is idempotent.
func (f *File) Close() error {
	if f == nil {
		return nil
	}
	f.closeOnce.Do(func() { f.closeErr = os.Remove(f.path) })
	if errors.Is(f.closeErr, os.ErrNotExist) {
		return nil
	}
	return f.closeErr
}

// Export loads browser cookies and writes a temporary Netscape cookie jar.
func Export(ctx context.Context, value string) (_ *File, resultErr error) {
	source, err := Parse(value)
	if err != nil {
		return nil, err
	}
	cookies, err := read(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("read %s cookies: %w", source.Browser, err)
	}
	cookies = deduplicate(cookies)
	if len(cookies) == 0 {
		return nil, fmt.Errorf("no valid cookies found in %s profile", source.Browser)
	}
	return exportCookies(ctx, cookies)
}

func exportCookies(
	ctx context.Context,
	cookies []*kooky.Cookie,
) (_ *File, resultErr error) {
	file, err := os.CreateTemp("", "dis-cookies-*.txt")
	if err != nil {
		return nil, fmt.Errorf("create temporary cookie jar: %w", err)
	}
	result := &File{path: file.Name()}
	defer func() {
		if resultErr != nil {
			_ = file.Close()
			_ = result.Close()
		}
	}()
	normalized, err := normalizeCookies(ctx, cookies)
	if err != nil {
		return nil, err
	}
	buffer := bufio.NewWriter(file)
	kooky.ExportCookies(ctx, buffer, normalized)
	err = errors.Join(ctx.Err(), buffer.Flush(), file.Close())
	if err != nil {
		return nil, fmt.Errorf("write temporary cookie jar: %w", err)
	}
	return result, nil
}

func deduplicate(cookies []*kooky.Cookie) []*kooky.Cookie {
	unique := make([]*kooky.Cookie, 0, len(cookies))
	indexes := make(map[string]int, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		key := cookie.Domain + "\x00" + cookie.Name + "\x00" + cookie.Path
		if index, exists := indexes[key]; exists {
			unique[index] = cookie
			continue
		}
		indexes[key] = len(unique)
		unique = append(unique, cookie)
	}
	return unique
}

func normalizeCookies(
	ctx context.Context,
	cookies []*kooky.Cookie,
) ([]*kooky.Cookie, error) {
	normalized := make([]*kooky.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if cookie == nil {
			continue
		}
		clone := *cookie
		if clone.Path == "" {
			clone.Path = "/"
		}
		if clone.Expires.IsZero() {
			clone.Expires = time.Unix(0, 0)
		}
		if err := clone.Valid(); err != nil {
			return nil, fmt.Errorf("invalid cookie %q: %w", clone.Name, err)
		}
		normalized = append(normalized, &clone)
	}
	return normalized, nil
}
