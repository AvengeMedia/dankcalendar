package keyring

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/godbus/dbus/v5"
)

// TestLiveSecretService talks to the session's real org.freedesktop.secrets
// daemon; opt in with DANKCAL_KEYRING_LIVE_TEST=1.
func TestLiveSecretService(t *testing.T) {
	if os.Getenv("DANKCAL_KEYRING_LIVE_TEST") != "1" {
		t.Skip("set DANKCAL_KEYRING_LIVE_TEST=1 to run against the session secret service")
	}

	store := Open()
	if !store.Available() {
		t.Fatal("keyring unavailable")
	}
	service, ok := store.backend.(*secretService)
	if !ok {
		t.Fatalf("backend is %T, want *secretService", store.backend)
	}

	const accountID = "__dankcal_live_test__"
	secret := []byte("live-test-secret")

	if err := store.Set(accountID, "token", secret); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := store.Get(accountID, "token")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("Get() = %q, want %q", got, secret)
	}
	if err := store.Delete(accountID, "token"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Get(accountID, "token"); err != ErrNotFound {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}

	// An item as 99designs/keyring wrote it: profile attribute, value wrapped
	// in JSON, content-type application/json.
	legacyKey := entryKey(accountID, "legacy")
	payload, err := json.Marshal(storedSecret{Key: legacyKey, Data: secret, Label: legacyKey, Description: credentialDescription})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.openSession()
	if err != nil {
		t.Fatalf("openSession() error = %v", err)
	}
	defer service.closeSession(session)

	properties := map[string]dbus.Variant{
		itemInterface + ".Label":      dbus.MakeVariant(legacyKey),
		itemInterface + ".Attributes": dbus.MakeVariant(map[string]string{"profile": legacyKey}),
	}
	wire := wireSecret{Session: session, Parameters: []byte{}, Value: payload, ContentType: "application/json"}
	var item, prompt dbus.ObjectPath
	if err := service.object(service.collection).Call(collectionInterface+".CreateItem", 0, properties, wire, true).Store(&item, &prompt); err != nil {
		t.Fatalf("legacy CreateItem error = %v", err)
	}
	if _, err := service.completePrompt(prompt); err != nil {
		t.Fatalf("legacy CreateItem prompt error = %v", err)
	}

	got, err = store.Get(accountID, "legacy")
	if err != nil {
		t.Fatalf("Get(legacy) error = %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("Get(legacy) = %q, want unwrapped %q", got, secret)
	}
	if err := store.Delete(accountID, "legacy"); err != nil {
		t.Fatalf("Delete(legacy) error = %v", err)
	}
}
