package browsercookies

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/browserutils/kooky"
	"github.com/browserutils/kooky/browser/brave"
	"github.com/browserutils/kooky/browser/chrome"
	"github.com/browserutils/kooky/browser/chromium"
	"github.com/browserutils/kooky/browser/edge"
	"github.com/browserutils/kooky/browser/firefox"
)

type reader func(
	context.Context,
	string,
	...kooky.Filter,
) ([]*kooky.Cookie, error)

var readers = map[Browser]reader{
	BrowserBrave:    brave.ReadCookies,
	BrowserChrome:   chrome.ReadCookies,
	BrowserChromium: chromium.ReadCookies,
	BrowserEdge:     edge.ReadCookies,
	BrowserFirefox:  firefox.ReadCookies,
	BrowserHelium:   readHelium,
}

func read(ctx context.Context, source Source) ([]*kooky.Cookie, error) {
	if source.Profile != "" {
		path, isPath, err := resolveCookieStorePath(source.Browser, source.Profile)
		if err != nil {
			return nil, err
		}
		if isPath {
			cookies, readErr := readers[source.Browser](ctx, path, kooky.Valid)
			return cookies, normalizeReadError(
				runtime.GOOS,
				source.Browser,
				readErr,
			)
		}
	}
	return readDiscoveredProfile(ctx, source)
}

func resolveCookieStorePath(
	browser Browser,
	profile string,
) (string, bool, error) {
	path, err := expandHome(profile)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		if looksLikePath(profile) {
			return "", false, fmt.Errorf("browser profile not found: %s", path)
		}
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inspect browser profile: %w", err)
	}
	if !info.IsDir() {
		return path, true, nil
	}

	var candidates []string
	if browser == BrowserFirefox {
		candidates = []string{filepath.Join(path, "cookies.sqlite")}
	} else {
		candidates = []string{
			filepath.Join(path, "Network", "Cookies"),
			filepath.Join(path, "Cookies"),
		}
	}
	for _, candidate := range candidates {
		candidateInfo, candidateErr := os.Stat(candidate)
		if candidateErr == nil && !candidateInfo.IsDir() {
			return candidate, true, nil
		}
		if candidateErr != nil && !errors.Is(candidateErr, os.ErrNotExist) {
			return "", false, fmt.Errorf("inspect cookie store: %w", candidateErr)
		}
	}
	return "", false, fmt.Errorf("cookie store not found in profile: %s", path)
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") &&
		!strings.HasPrefix(path, `~\`) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func looksLikePath(value string) bool {
	return filepath.IsAbs(value) || strings.HasPrefix(value, ".") ||
		strings.HasPrefix(value, "~") || strings.ContainsAny(value, `/\`)
}

func readDiscoveredProfile(
	ctx context.Context,
	source Source,
) ([]*kooky.Cookie, error) {
	stores := discoverProfileStores(ctx)
	defer func() {
		for _, store := range stores {
			if store.close != nil {
				_ = store.close()
			}
		}
	}()
	return readProfileFromStores(ctx, source, stores)
}

func readProfileFromStores(
	ctx context.Context,
	source Source,
	stores []profileStore,
) ([]*kooky.Cookie, error) {
	for _, store := range stores {
		if store.browser != source.Browser ||
			!matchesProfile(store, source.Profile) {
			continue
		}
		cookies, err := store.read(ctx)
		return cookies, normalizeReadError(runtime.GOOS, source.Browser, err)
	}
	if source.Profile == "" {
		return nil, fmt.Errorf("no %s browser profile found", source.Browser)
	}
	return nil, fmt.Errorf(
		"%s browser profile %q not found",
		source.Browser,
		source.Profile,
	)
}

func matchesProfile(store profileStore, profile string) bool {
	if profile == "" {
		return true
	}
	if store.profile == profile {
		return true
	}
	directory := filepath.Dir(store.path)
	if filepath.Base(directory) == "Network" {
		directory = filepath.Dir(directory)
	}
	return filepath.Base(directory) == profile
}

func normalizeReadError(goos string, browser Browser, err error) error {
	if err == nil {
		return nil
	}
	if goos == "windows" &&
		strings.Contains(strings.ToLower(err.Error()), "v20 app-bound") {
		return fmt.Errorf(
			"%s profile uses Windows v20 app-bound encryption; use firefox on Windows",
			browser,
		)
	}
	return err
}
