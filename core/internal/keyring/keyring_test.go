package keyring

import (
	"bytes"
	"path/filepath"
	"testing"
)

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

func TestDecodeStoredSecret(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
		want    []byte
	}{
		{"wrapped", []byte(`{"Key":"acc::token","Data":"c2VrcmV0","Label":"l","Description":"d"}`), []byte("sekret")},
		{"key mismatch passes through", []byte(`{"Key":"other::token","Data":"c2VrcmV0"}`), []byte(`{"Key":"other::token","Data":"c2VrcmV0"}`)},
		{"raw bytes", []byte("raw-secret"), []byte("raw-secret")},
		{"raw json without wrapper fields", []byte(`{"access_token":"abc"}`), []byte(`{"access_token":"abc"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decodeStoredSecret("acc::token", tc.payload); !bytes.Equal(got, tc.want) {
				t.Fatalf("decodeStoredSecret() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	store := &fileStore{dir: t.TempDir(), password: "test-password"}

	if _, err := store.Get("acc::token"); err != ErrNotFound {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}

	secret := []byte("oauth-refresh-token")
	if err := store.Set("acc::token", secret, "label"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get("acc::token")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("Get() = %q, want %q", got, secret)
	}

	if _, err := (&fileStore{dir: store.dir, password: "wrong"}).Get("acc::token"); err == nil {
		t.Fatal("Get() with wrong password succeeded")
	}

	if err := store.Delete("acc::token"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if err := store.Delete("acc::token"); err != ErrNotFound {
		t.Fatalf("Delete(missing) error = %v, want ErrNotFound", err)
	}
}

func TestFileStoreFilenameEscaping(t *testing.T) {
	store := &fileStore{dir: t.TempDir(), password: "p"}
	key := "acc/with/slashes::token"

	if err := store.Set(key, []byte("v"), "l"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if got := store.filename(key); filepath.Dir(got) != store.dir {
		t.Fatalf("filename(%q) = %q escapes the store directory", key, got)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(got, []byte("v")) {
		t.Fatalf("Get() = %q, want %q", got, "v")
	}
}
