//go:build darwin

package browsercookies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/browserutils/kooky"
	"github.com/steipete/sweetcookie"
)

func readHelium(
	ctx context.Context,
	path string,
	filters ...kooky.Filter,
) ([]*kooky.Cookie, error) {
	result, err := sweetcookie.Get(ctx, sweetcookie.Options{
		AllowAllHosts: true,
		Browsers:      []sweetcookie.Browser{sweetcookie.BrowserHelium},
		Mode:          sweetcookie.ModeMerge,
		Profiles: map[sweetcookie.Browser]string{
			sweetcookie.BrowserHelium: path,
		},
	})
	if err != nil {
		return nil, err
	}
	cookies := make([]*kooky.Cookie, 0, len(result.Cookies))
	for _, cookie := range result.Cookies {
		converted := &kooky.Cookie{Cookie: http.Cookie{
			Name: cookie.Name, Value: cookie.Value, Domain: cookie.Domain,
			Path: cookie.Path, Secure: cookie.Secure, HttpOnly: cookie.HTTPOnly,
		}}
		if cookie.Expires != nil {
			converted.Expires = *cookie.Expires
		}
		cookies = append(cookies, converted)
	}
	filtered, filterErr := kooky.FilterCookies(ctx, cookies, filters...).
		ReadAllCookies(ctx)
	if len(filtered) == 0 && len(result.Warnings) > 0 {
		warningErr := errors.New(strings.Join(result.Warnings, "; "))
		return nil, errors.Join(filterErr, fmt.Errorf("read Helium Keychain: %w", warningErr))
	}
	return filtered, filterErr
}
