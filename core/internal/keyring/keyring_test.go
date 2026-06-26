package keyring

import "testing"

func TestCollectionBaseName(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"gnome login", "/org/freedesktop/secrets/collection/login", "login"},
		{"kwallet default", "/org/freedesktop/secrets/collection/kdewallet", "kdewallet"},
		{"hex escaped space", "/org/freedesktop/secrets/collection/My_20Wallet", "My Wallet"},
		{"null path", "/", ""},
		{"empty", "", ""},
		{"trailing slash", "/org/freedesktop/secrets/collection/", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := collectionBaseName(tc.path); got != tc.want {
				t.Fatalf("collectionBaseName(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}
