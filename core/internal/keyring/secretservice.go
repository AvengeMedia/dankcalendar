package keyring

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	kr "github.com/99designs/keyring"
	"github.com/godbus/dbus/v5"
)

const promptTimeout = 30 * time.Second

type secretServiceKeyring struct {
	conn       *dbus.Conn
	collection string
	session    dbus.ObjectPath
}

type secretServiceSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func openSecretService(collection string) (*secretServiceKeyring, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	var output dbus.Variant
	var session dbus.ObjectPath
	if err := conn.Object(secretServiceBus, dbus.ObjectPath(secretServicePath)).Call(
		"org.freedesktop.Secret.Service.OpenSession", 0, "plain", dbus.MakeVariant(""),
	).Store(&output, &session); err != nil {
		return nil, err
	}
	return &secretServiceKeyring{conn: conn, collection: collection, session: session}, nil
}

func (k *secretServiceKeyring) Get(key string) ([]byte, error) {
	items, err := k.search(key)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, kr.ErrKeyNotFound
	}
	if err := k.ensureUnlocked(items[0]); err != nil {
		return nil, err
	}

	var secret secretServiceSecret
	if err := k.conn.Object(secretServiceBus, items[0]).Call(
		"org.freedesktop.Secret.Item.GetSecret", 0, k.session,
	).Store(&secret); err != nil {
		return nil, err
	}

	// 99designs/keyring stored Item as JSON. Decode it so existing credentials
	// remain available after moving to the prompt-safe implementation.
	var legacy kr.Item
	if err := json.Unmarshal(secret.Value, &legacy); err == nil && legacy.Key == key {
		return legacy.Data, nil
	}
	return secret.Value, nil
}

func (k *secretServiceKeyring) Set(key string, value []byte, label string) error {
	collection, err := k.collectionPath(true)
	if err != nil {
		return err
	}
	if err := k.ensureUnlocked(collection); err != nil {
		return err
	}

	properties := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label":      dbus.MakeVariant(label),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(map[string]string{"profile": key}),
	}
	var item, prompt dbus.ObjectPath
	if err := k.conn.Object(secretServiceBus, collection).Call(
		"org.freedesktop.Secret.Collection.CreateItem", 0, properties,
		secretServiceSecret{Session: k.session, Value: value, ContentType: "application/octet-stream"}, true,
	).Store(&item, &prompt); err != nil {
		return err
	}
	if prompt == "/" {
		return nil
	}
	_, err = completeSecretPrompt(k.conn, prompt)
	return err
}

func (k *secretServiceKeyring) Remove(key string) error {
	items, err := k.search(key)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return kr.ErrKeyNotFound
	}
	if err := k.ensureUnlocked(items[0]); err != nil {
		return err
	}

	var prompt dbus.ObjectPath
	if err := k.conn.Object(secretServiceBus, items[0]).Call("org.freedesktop.Secret.Item.Delete", 0).Store(&prompt); err != nil {
		return err
	}
	if prompt == "/" {
		return nil
	}
	_, err = completeSecretPrompt(k.conn, prompt)
	return err
}

func (k *secretServiceKeyring) search(key string) ([]dbus.ObjectPath, error) {
	collection, err := k.collectionPath(false)
	if err != nil {
		return nil, err
	}
	if collection == "" {
		return nil, nil
	}

	var items []dbus.ObjectPath
	err = k.conn.Object(secretServiceBus, collection).Call(
		"org.freedesktop.Secret.Collection.SearchItems", 0, map[string]string{"profile": key},
	).Store(&items)
	return items, err
}

func (k *secretServiceKeyring) collectionPath(create bool) (dbus.ObjectPath, error) {
	obj := k.conn.Object(secretServiceBus, dbus.ObjectPath(secretServicePath))
	value, err := obj.GetProperty("org.freedesktop.Secret.Service.Collections")
	if err != nil {
		return "", err
	}
	for _, path := range value.Value().([]dbus.ObjectPath) {
		if collectionBaseName(string(path)) == k.collection {
			return path, nil
		}
	}
	if !create {
		return "", nil
	}

	properties := map[string]dbus.Variant{
		"org.freedesktop.Secret.Collection.Label": dbus.MakeVariant(k.collection),
	}
	var collection, prompt dbus.ObjectPath
	if err := obj.Call("org.freedesktop.Secret.Service.CreateCollection", 0, properties, "").Store(&collection, &prompt); err != nil {
		return "", err
	}
	if prompt == "/" {
		return collection, nil
	}
	result, err := completeSecretPrompt(k.conn, prompt)
	if err != nil {
		return "", err
	}
	collection, ok := result.Value().(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("unexpected collection prompt result %T", result.Value())
	}
	return collection, nil
}

func (k *secretServiceKeyring) ensureUnlocked(path dbus.ObjectPath) error {
	value, err := k.conn.Object(secretServiceBus, path).GetProperty("org.freedesktop.Secret.Collection.Locked")
	if err != nil {
		value, err = k.conn.Object(secretServiceBus, path).GetProperty("org.freedesktop.Secret.Item.Locked")
		if err != nil {
			return err
		}
	}
	if !value.Value().(bool) {
		return nil
	}

	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	if err := k.conn.Object(secretServiceBus, dbus.ObjectPath(secretServicePath)).Call(
		"org.freedesktop.Secret.Service.Unlock", 0, []dbus.ObjectPath{path},
	).Store(&unlocked, &prompt); err != nil {
		return err
	}
	if prompt == "/" {
		return nil
	}
	_, err = completeSecretPrompt(k.conn, prompt)
	return err
}

func completeSecretPrompt(conn *dbus.Conn, prompt dbus.ObjectPath) (dbus.Variant, error) {
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	options := []dbus.MatchOption{
		dbus.WithMatchSender(secretServiceBus),
		dbus.WithMatchInterface("org.freedesktop.Secret.Prompt"),
		dbus.WithMatchMember("Completed"),
		dbus.WithMatchObjectPath(prompt),
	}
	if err := conn.AddMatchSignal(options...); err != nil {
		return dbus.Variant{}, err
	}
	// best-effort cleanup: remove match signal and ignore any error
	defer func() {
		_ = conn.RemoveMatchSignal(options...)
	}()

	if err := conn.Object(secretServiceBus, prompt).Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err; err != nil {
		return dbus.Variant{}, err
	}

	select {
	case signal := <-signals:
		if len(signal.Body) != 2 {
			return dbus.Variant{}, errors.New("secret prompt completion has an unexpected signature")
		}
		dismissed, ok := signal.Body[0].(bool)
		if !ok {
			return dbus.Variant{}, errors.New("secret prompt completion has an invalid dismissed flag")
		}
		if dismissed {
			return dbus.Variant{}, errors.New("secret prompt was dismissed")
		}
		result, ok := signal.Body[1].(dbus.Variant)
		if !ok {
			return dbus.Variant{}, errors.New("secret prompt completion has an invalid result")
		}
		return result, nil
	case <-time.After(promptTimeout):
		return dbus.Variant{}, errors.New("timed out waiting for secret prompt completion")
	}
}
