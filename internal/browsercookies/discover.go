package browsercookies

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/browserutils/kooky"
)

// Candidate is one browser session and its temporary cookie jar.
type Candidate struct {
	Source string
	Path   string
}

// Collection owns the temporary jars created during browser discovery.
type Collection struct {
	candidates []Candidate
	files      []*File
	closeOnce  sync.Once
	closeErr   error
}

// Candidates returns browser sessions in profile recency order.
func (c *Collection) Candidates() []Candidate {
	if c == nil {
		return nil
	}
	return slices.Clone(c.candidates)
}

// Close removes every temporary cookie jar. It is idempotent.
func (c *Collection) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		errs := make([]error, 0, len(c.files))
		for _, file := range c.files {
			errs = append(errs, file.Close())
		}
		c.closeErr = errors.Join(errs...)
	})
	return c.closeErr
}

type profileStore struct {
	browser  Browser
	profile  string
	path     string
	modified int64
	read     func(context.Context) ([]*kooky.Cookie, error)
	close    func() error
}

// Discover exports every browser profile containing valid cookies. A broken or
// locked profile is reported without hiding usable ones.
func Discover(
	ctx context.Context,
) (_ *Collection, warnings []error, resultErr error) {
	stores := discoverProfileStores(ctx)
	defer func() {
		for _, store := range stores {
			if store.close != nil {
				_ = store.close()
			}
		}
	}()
	return collect(ctx, stores)
}

func collect(
	ctx context.Context,
	stores []profileStore,
) (_ *Collection, warnings []error, resultErr error) {
	collection := new(Collection)
	defer func() {
		if resultErr != nil {
			_ = collection.Close()
		}
	}()
	for _, store := range stores {
		if err := ctx.Err(); err != nil {
			return nil, warnings, err
		}
		cookies, err := store.read(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Errorf(
				"read %s: %w",
				storeLabel(store.browser, store.profile, ""),
				normalizeReadError(runtime.GOOS, store.browser, err),
			))
		}
		for _, group := range cookieGroups(cookies) {
			file, exportErr := exportCookies(
				ctx,
				deduplicate(group.cookies),
			)
			if exportErr != nil {
				warnings = append(warnings, fmt.Errorf(
					"export %s: %w",
					storeLabel(store.browser, store.profile, group.container),
					exportErr,
				))
				continue
			}
			collection.files = append(collection.files, file)
			collection.candidates = append(collection.candidates, Candidate{
				Source: storeLabel(store.browser, store.profile, group.container),
				Path:   file.Path(),
			})
		}
	}
	return collection, warnings, nil
}

func discoverProfileStores(ctx context.Context) []profileStore {
	var stores []profileStore
	seen := make(map[string]struct{})
	for _, store := range kooky.FindAllCookieStores(ctx) {
		path := store.FilePath()
		info, err := os.Stat(path)
		browser := Browser(store.Browser())
		if err != nil || info.IsDir() || !supportedBrowser(browser) {
			_ = store.Close()
			continue
		}
		key := canonicalStorePath(path)
		if _, exists := seen[key]; exists {
			_ = store.Close()
			continue
		}
		seen[key] = struct{}{}
		cookieStore := store
		stores = append(stores, profileStore{
			browser: browser, profile: profileName(store.Profile(), path),
			path: path, modified: info.ModTime().UnixNano(),
			read: func(readCtx context.Context) ([]*kooky.Cookie, error) {
				return cookieStore.TraverseCookies(kooky.Valid).
					ReadAllCookies(readCtx)
			},
			close: cookieStore.Close,
		})
	}
	for _, store := range supplementaryProfileStores() {
		key := canonicalStorePath(store.path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		stores = append(stores, store)
	}
	slices.SortStableFunc(stores, func(a, b profileStore) int {
		if a.modified != b.modified {
			return cmp.Compare(b.modified, a.modified)
		}
		return strings.Compare(a.path, b.path)
	})
	return stores
}

func canonicalStorePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}

func profileName(profile, path string) string {
	if strings.TrimSpace(profile) != "" {
		return profile
	}
	directory := filepath.Dir(path)
	if filepath.Base(directory) == "Network" {
		directory = filepath.Dir(directory)
	}
	return filepath.Base(directory)
}

func supplementaryProfileStores() []profileStore {
	chromiumRoots := supplementaryChromiumRoots()
	firefoxRoots := supplementaryFirefoxRoots()
	stores := make([]profileStore, 0, len(chromiumRoots)+len(firefoxRoots))
	for _, root := range chromiumRoots {
		stores = append(stores, chromiumStoresInRoot(root.browser, root.path)...)
	}
	for _, root := range firefoxRoots {
		stores = append(stores, firefoxStoresInRoot(root)...)
	}
	return stores
}

type browserRoot struct {
	browser Browser
	path    string
}

func supplementaryChromiumRoots() []browserRoot {
	config, configErr := os.UserConfigDir()
	home, homeErr := os.UserHomeDir()
	var roots []browserRoot
	if configErr == nil {
		roots = append(roots, browserRoot{
			browser: BrowserHelium,
			path:    filepath.Join(config, "net.imput.helium"),
		})
	}
	if runtime.GOOS != "linux" || homeErr != nil {
		return roots
	}
	flatpak := filepath.Join(home, ".var", "app")
	return append(roots,
		browserRoot{BrowserChrome, filepath.Join(
			flatpak, "com.google.Chrome", "config", "google-chrome",
		)},
		browserRoot{BrowserChromium, filepath.Join(
			flatpak, "org.chromium.Chromium", "config", "chromium",
		)},
		browserRoot{BrowserBrave, filepath.Join(
			flatpak, "com.brave.Browser", "config", "BraveSoftware", "Brave-Browser",
		)},
		browserRoot{BrowserEdge, filepath.Join(
			flatpak, "com.microsoft.Edge", "config", "microsoft-edge",
		)},
		browserRoot{BrowserHelium, filepath.Join(
			flatpak, "net.imput.helium", "config", "net.imput.helium",
		)},
	)
}

func chromiumStoresInRoot(browser Browser, root string) []profileStore {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var stores []profileStore
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		profile := filepath.Join(root, entry.Name())
		path, found := firstRegularFile(
			filepath.Join(profile, "Network", "Cookies"),
			filepath.Join(profile, "Cookies"),
		)
		if !found {
			continue
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		storePath := path
		reader := readers[browser]
		stores = append(stores, profileStore{
			browser: browser, profile: entry.Name(), path: path,
			modified: info.ModTime().UnixNano(),
			read: func(readCtx context.Context) ([]*kooky.Cookie, error) {
				return reader(
					readCtx,
					storePath,
					kooky.Valid,
				)
			},
		})
	}
	return stores
}

func supplementaryFirefoxRoots() []string {
	if runtime.GOOS != "linux" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".var", "app", "org.mozilla.firefox", ".mozilla", "firefox"),
		filepath.Join(home, ".var", "app", "org.mozilla.firefox", "config", "mozilla", "firefox"),
	}
}

func firefoxStoresInRoot(root string) []profileStore {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var stores []profileStore
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name(), "cookies.sqlite")
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			continue
		}
		storePath := path
		stores = append(stores, profileStore{
			browser: BrowserFirefox, profile: entry.Name(), path: path,
			modified: info.ModTime().UnixNano(),
			read: func(readCtx context.Context) ([]*kooky.Cookie, error) {
				return readers[BrowserFirefox](
					readCtx,
					storePath,
					kooky.Valid,
				)
			},
		})
	}
	return stores
}

func firstRegularFile(paths ...string) (string, bool) {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

type cookieGroup struct {
	container string
	cookies   []*kooky.Cookie
}

func cookieGroups(cookies []*kooky.Cookie) []cookieGroup {
	grouped := make(map[string][]*kooky.Cookie)
	for _, cookie := range cookies {
		if cookie != nil {
			grouped[cookie.Container] = append(grouped[cookie.Container], cookie)
		}
	}
	containers := slices.Sorted(maps.Keys(grouped))
	groups := make([]cookieGroup, 0, len(containers))
	for _, container := range containers {
		groups = append(groups, cookieGroup{
			container: container,
			cookies:   grouped[container],
		})
	}
	return groups
}

func storeLabel(browser Browser, profile, container string) string {
	label := string(browser)
	if profile != "" {
		label += "/" + profile
	}
	if container != "" {
		label += " (container " + container + ")"
	}
	return label
}
