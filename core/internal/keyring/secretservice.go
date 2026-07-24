package keyring

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/godbus/dbus/v5"
)

// secretService talks org.freedesktop.secrets directly. The previous
// go-libsecret-based backend waited on prompts without matching the Completed
// signal, hanging forever against KeePassXC (issue #68).
type secretService struct {
	conn       *dbus.Conn
	collection dbus.ObjectPath
}

// wireSecret is the Secret struct from the spec (oayays).
type wireSecret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

func openSecretService() (*secretService, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}

	s := &secretService{conn: conn}
	session, err := s.openSession()
	if err != nil {
		return nil, err
	}
	s.closeSession(session)

	s.collection = resolveDefaultCollection(conn)
	return s, nil
}

// resolveDefaultCollection reuses the Secret Service "default" collection
// (e.g. KWallet's existing wallet) instead of forcing a separate "login" one.
func resolveDefaultCollection(conn *dbus.Conn) dbus.ObjectPath {
	var path dbus.ObjectPath
	obj := conn.Object(secretServiceBus, secretServicePath)
	if err := obj.Call(serviceInterface+".ReadAlias", 0, "default").Store(&path); err != nil || path == "/" {
		return loginCollectionPath
	}
	return path
}

func (s *secretService) Get(key string) ([]byte, error) {
	return s.getFrom(s.collection, key)
}

func (s *secretService) Set(key string, value []byte, label string) error {
	collection, err := s.ensureCollection()
	if err != nil {
		return err
	}
	if err := s.unlock(collection); err != nil {
		return err
	}

	session, err := s.openSession()
	if err != nil {
		return err
	}
	defer s.closeSession(session)

	// The "profile" attribute matches items written through 99designs/keyring,
	// so pre-rewrite credentials are found and replaced in place.
	properties := map[string]dbus.Variant{
		itemInterface + ".Label":      dbus.MakeVariant(label),
		itemInterface + ".Attributes": dbus.MakeVariant(map[string]string{"profile": key}),
	}
	secret := wireSecret{Session: session, Parameters: []byte{}, Value: value, ContentType: "application/octet-stream"}

	var item, prompt dbus.ObjectPath
	if err := s.object(collection).Call(collectionInterface+".CreateItem", 0, properties, secret, true).Store(&item, &prompt); err != nil {
		return err
	}
	_, err = s.completePrompt(prompt)
	return err
}

func (s *secretService) Delete(key string) error {
	return s.deleteFrom(s.collection, key)
}

func (s *secretService) getFrom(collection dbus.ObjectPath, key string) ([]byte, error) {
	items, err := s.search(collection, key)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	if err := s.unlock(items[0]); err != nil {
		return nil, err
	}

	session, err := s.openSession()
	if err != nil {
		return nil, err
	}
	defer s.closeSession(session)

	var secret wireSecret
	if err := s.object(items[0]).Call(itemInterface+".GetSecret", 0, session).Store(&secret); err != nil {
		return nil, err
	}
	return decodeStoredSecret(key, secret.Value), nil
}

func (s *secretService) deleteFrom(collection dbus.ObjectPath, key string) error {
	items, err := s.search(collection, key)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return ErrNotFound
	}

	for _, item := range items {
		var prompt dbus.ObjectPath
		if err := s.object(item).Call(itemInterface+".Delete", 0).Store(&prompt); err != nil {
			return err
		}
		if _, err := s.completePrompt(prompt); err != nil {
			return err
		}
	}
	return nil
}

// search treats a missing collection as no matches so lookups against an
// absent fallback collection degrade to ErrNotFound instead of a D-Bus error.
func (s *secretService) search(collection dbus.ObjectPath, key string) ([]dbus.ObjectPath, error) {
	ok, err := s.hasCollection(collection)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}

	var items []dbus.ObjectPath
	if err := s.object(collection).Call(collectionInterface+".SearchItems", 0, map[string]string{"profile": key}).Store(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *secretService) hasCollection(collection dbus.ObjectPath) (bool, error) {
	value, err := s.service().GetProperty(serviceInterface + ".Collections")
	if err != nil {
		return false, err
	}
	paths, ok := value.Value().([]dbus.ObjectPath)
	if !ok {
		return false, fmt.Errorf("unexpected Collections property type %T", value.Value())
	}
	return slices.Contains(paths, collection), nil
}

func (s *secretService) ensureCollection() (dbus.ObjectPath, error) {
	ok, err := s.hasCollection(s.collection)
	if err != nil {
		return "", err
	}
	if ok {
		return s.collection, nil
	}

	properties := map[string]dbus.Variant{
		collectionInterface + ".Label": dbus.MakeVariant(collectionBaseName(string(s.collection))),
	}
	var collection, prompt dbus.ObjectPath
	if err := s.service().Call(serviceInterface+".CreateCollection", 0, properties, "default").Store(&collection, &prompt); err != nil {
		return "", err
	}
	if collection != "/" {
		return collection, nil
	}

	result, err := s.completePrompt(prompt)
	if err != nil {
		return "", err
	}
	created, ok := result.Value().(dbus.ObjectPath)
	if !ok {
		return "", fmt.Errorf("unexpected collection prompt result %T", result.Value())
	}
	return created, nil
}

func (s *secretService) unlock(path dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	if err := s.service().Call(serviceInterface+".Unlock", 0, []dbus.ObjectPath{path}).Store(&unlocked, &prompt); err != nil {
		return err
	}
	_, err := s.completePrompt(prompt)
	return err
}

// completePrompt subscribes before calling Prompt so Completed cannot be
// missed, and filters by path and member because the session bus connection
// is shared with the app's other signal consumers.
func (s *secretService) completePrompt(prompt dbus.ObjectPath) (dbus.Variant, error) {
	if prompt == "" || prompt == "/" {
		return dbus.Variant{}, nil
	}

	match := []dbus.MatchOption{
		dbus.WithMatchObjectPath(prompt),
		dbus.WithMatchInterface(promptInterface),
		dbus.WithMatchMember("Completed"),
	}
	if err := s.conn.AddMatchSignal(match...); err != nil {
		return dbus.Variant{}, err
	}
	defer func() { _ = s.conn.RemoveMatchSignal(match...) }()

	signals := make(chan *dbus.Signal, 16)
	s.conn.Signal(signals)
	defer s.conn.RemoveSignal(signals)

	if err := s.object(prompt).Call(promptInterface+".Prompt", 0, "").Err; err != nil {
		return dbus.Variant{}, err
	}

	timeout := time.NewTimer(promptTimeout)
	defer timeout.Stop()

	for {
		select {
		case sig := <-signals:
			if sig == nil || sig.Path != prompt || sig.Name != promptInterface+".Completed" || len(sig.Body) != 2 {
				continue
			}
			if dismissed, _ := sig.Body[0].(bool); dismissed {
				return dbus.Variant{}, errors.New("secret service prompt dismissed")
			}
			result, ok := sig.Body[1].(dbus.Variant)
			if !ok {
				return dbus.Variant{}, errors.New("unexpected secret service prompt result")
			}
			return result, nil
		case <-timeout.C:
			return dbus.Variant{}, fmt.Errorf("secret service prompt timed out after %s", promptTimeout)
		}
	}
}

func (s *secretService) openSession() (dbus.ObjectPath, error) {
	var output dbus.Variant
	var session dbus.ObjectPath
	if err := s.service().Call(serviceInterface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&output, &session); err != nil {
		return "", err
	}
	return session, nil
}

func (s *secretService) closeSession(session dbus.ObjectPath) {
	s.object(session).Call(sessionInterface+".Close", 0)
}

func (s *secretService) service() dbus.BusObject {
	return s.conn.Object(secretServiceBus, secretServicePath)
}

func (s *secretService) object(path dbus.ObjectPath) dbus.BusObject {
	return s.conn.Object(secretServiceBus, path)
}
