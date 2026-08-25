package browsercookies

import (
	"context"
	"testing"

	"github.com/browserutils/kooky"
)

func TestReadProfileFromStoresIncludesSupplementaryStore(t *testing.T) {
	t.Parallel()

	want := new(kooky.Cookie)
	want.Name = "session"
	stores := []profileStore{{
		browser: BrowserHelium,
		profile: "Profile 1",
		path:    "/supplementary/helium/Profile 1/Network/Cookies",
		read: func(context.Context) ([]*kooky.Cookie, error) {
			return []*kooky.Cookie{want}, nil
		},
	}}

	cookies, err := readProfileFromStores(context.Background(), Source{
		Browser: BrowserHelium,
		Profile: "Profile 1",
	}, stores)
	if err != nil {
		t.Fatalf("readProfileFromStores returned an error: %v", err)
	}
	if len(cookies) != 1 || cookies[0] != want {
		t.Fatalf("readProfileFromStores returned %v, want the named profile", cookies)
	}
}
