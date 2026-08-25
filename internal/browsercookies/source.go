// Package browsercookies loads browser cookies into a temporary Netscape jar.
package browsercookies

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Browser identifies a supported browser cookie-store format.
type Browser string

const (
	BrowserBrave    Browser = "brave"
	BrowserChrome   Browser = "chrome"
	BrowserChromium Browser = "chromium"
	BrowserEdge     Browser = "edge"
	BrowserFirefox  Browser = "firefox"
	BrowserHelium   Browser = "helium"
)

// Source identifies a browser and an optional profile name or path.
type Source struct {
	Browser Browser
	Profile string
}

// Parse parses BROWSER[:PROFILE].
func Parse(value string) (Source, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Source{}, errors.New("browser cookie source is empty")
	}
	browserName, profile, _ := strings.Cut(value, ":")
	if strings.Contains(browserName, "+") {
		return Source{}, errors.New(
			"browser keyring selectors are not supported; omit the +KEYRING suffix",
		)
	}
	browser := Browser(strings.ToLower(strings.TrimSpace(browserName)))
	if browser == "" {
		return Source{}, errors.New("browser name is empty")
	}
	if !supportedBrowser(browser) {
		return Source{}, fmt.Errorf(
			"unsupported browser %q (supported: %s)",
			browser,
			strings.Join(BrowserNames(), ", "),
		)
	}
	if strings.Contains(profile, "::") {
		return Source{}, errors.New("firefox container selectors are not supported")
	}
	return Source{Browser: browser, Profile: strings.TrimSpace(profile)}, nil
}

// BrowserNames returns the supported browser names in sorted order.
func BrowserNames() []string {
	browsers := slices.Sorted(maps.Keys(readers))
	names := make([]string, len(browsers))
	for index, browser := range browsers {
		names[index] = string(browser)
	}
	return names
}

func supportedBrowser(browser Browser) bool {
	_, ok := readers[browser]
	return ok
}
