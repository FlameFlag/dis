//go:build !darwin

package browsercookies

import (
	"context"

	"github.com/browserutils/kooky"
	"github.com/browserutils/kooky/browser/chromium"
)

func readHelium(
	ctx context.Context,
	path string,
	filters ...kooky.Filter,
) ([]*kooky.Cookie, error) {
	return chromium.ReadCookies(ctx, path, filters...)
}
