package keyring

import (
	"errors"
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
	ss "github.com/zalando/go-keyring/secret_service"
)

type secretServiceStore struct {
	svc            *ss.SecretService
	collectionPath dbus.ObjectPath
}

func openSecretServiceStore(collectionPath dbus.ObjectPath) (*secretServiceStore, error) {
	svc, err := ss.NewSecretService()
	if err != nil {
		return nil, err
	}
	if err := svc.CheckCollectionPath(collectionPath); err != nil {
		return nil, fmt.Errorf("check secret collection %q: %w", collectionBaseName(string(collectionPath)), err)
	}
	return &secretServiceStore{svc: svc, collectionPath: collectionPath}, nil
}

func (s *secretServiceStore) Get(key string) ([]byte, error) {
	return s.getFromCollection(s.collectionPath, key)
}

func (s *secretServiceStore) Set(key string, value []byte, label string) error {
	if err := s.unlock(s.collectionPath); err != nil {
		return err
	}

	session, err := s.svc.OpenSession()
	if err != nil {
		return err
	}
	defer s.svc.Close(session)

	secret := ss.Secret{
		Session:     session.Path(),
		Parameters:  []byte{},
		Value:       value,
		ContentType: "application/octet-stream",
	}
	properties := map[string]dbus.Variant{
		secretServiceItemIfc + ".Label": dbus.MakeVariant(label),
		secretServiceItemIfc + ".Attributes": dbus.MakeVariant(map[string]string{
			"service":  serviceName,
			"username": key,
		}),
	}

	var itemPath, prompt dbus.ObjectPath
	err = s.collection().Call(secretServiceCollectionIfc+".CreateItem", 0, properties, secret, true).Store(&itemPath, &prompt)
	if err != nil {
		return err
	}

	dismissed, _, err := s.handlePrompt(prompt)
	if err != nil {
		return err
	}
	if dismissed {
		return errors.New("secret service prompt was dismissed")
	}
	return nil
}

func (s *secretServiceStore) Delete(key string) error {
	return s.deleteFromCollection(s.collectionPath, key)
}

func (s *secretServiceStore) MigrateLoginCollection(refs []SecretRef) (int, error) {
	loginPath := fallbackCollectionPath()
	if s.collectionPath == loginPath || len(refs) == 0 {
		return 0, nil
	}
	if err := s.svc.CheckCollectionPath(loginPath); err != nil {
		return 0, nil
	}

	migrated := 0
	for _, ref := range refs {
		key := entryKey(ref.AccountID, ref.Key)
		value, err := s.getFromCollection(loginPath, key)
		switch {
		case errors.Is(err, ErrNotFound):
			continue
		case err != nil:
			return migrated, fmt.Errorf("read %s from login: %w", key, err)
		}

		if err := s.Set(key, value, entryLabel(ref.AccountID, ref.Key)); err != nil {
			return migrated, err
		}
		_ = s.deleteFromCollection(loginPath, key)
		migrated++
	}
	return migrated, nil
}

func (s *secretServiceStore) getFromCollection(collectionPath dbus.ObjectPath, key string) ([]byte, error) {
	if err := s.unlock(collectionPath); err != nil {
		return nil, err
	}

	collection := s.collectionAt(collectionPath)
	itemPath, err := s.findCurrentItem(collection, key)
	if err == nil {
		return s.readItemSecret(itemPath, key)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	itemPath, err = s.findLegacyItem(collection, key)
	if err == nil {
		return s.readItemSecret(itemPath, key)
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	return nil, ErrNotFound
}

func (s *secretServiceStore) deleteFromCollection(collectionPath dbus.ObjectPath, key string) error {
	collection := s.collectionAt(collectionPath)
	deleted := false

	itemPath, err := s.findCurrentItem(collection, key)
	switch {
	case err == nil:
		if err := s.deleteItem(itemPath); err != nil {
			return err
		}
		deleted = true
	case !errors.Is(err, ErrNotFound):
		return err
	}

	itemPath, err = s.findLegacyItem(collection, key)
	switch {
	case err == nil:
		if err := s.deleteItem(itemPath); err != nil {
			return err
		}
		deleted = true
	case !errors.Is(err, ErrNotFound):
		return err
	}

	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *secretServiceStore) readItemSecret(itemPath dbus.ObjectPath, key string) ([]byte, error) {
	if err := s.unlock(itemPath); err != nil {
		return nil, err
	}

	session, err := s.svc.OpenSession()
	if err != nil {
		return nil, err
	}
	defer s.svc.Close(session)

	secret, err := s.svc.GetSecret(itemPath, session.Path())
	if err != nil {
		return nil, err
	}
	return decodeStoredSecret(key, secret.Value), nil
}

func (s *secretServiceStore) findCurrentItem(collection dbus.BusObject, key string) (dbus.ObjectPath, error) {
	results, err := s.svc.SearchItems(collection, map[string]string{
		"service":  serviceName,
		"username": key,
	})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", ErrNotFound
	}
	return results[0], nil
}

func (s *secretServiceStore) findLegacyItem(collection dbus.BusObject, key string) (dbus.ObjectPath, error) {
	results, err := s.svc.SearchItems(collection, map[string]string{"profile": key})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", ErrNotFound
	}
	return results[0], nil
}

func (s *secretServiceStore) deleteItem(itemPath dbus.ObjectPath) error {
	if err := s.unlock(itemPath); err != nil {
		return err
	}

	var prompt dbus.ObjectPath
	err := s.svc.Object(secretServiceBus, itemPath).Call(secretServiceItemIfc+".Delete", 0).Store(&prompt)
	if err != nil {
		return err
	}

	dismissed, _, err := s.handlePrompt(prompt)
	if err != nil {
		return err
	}
	if dismissed {
		return errors.New("secret service prompt was dismissed")
	}
	return nil
}

func (s *secretServiceStore) unlock(path dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	err := s.service().Call(secretServiceInterface+".Unlock", 0, []dbus.ObjectPath{path}).Store(&unlocked, &prompt)
	if err != nil {
		return err
	}

	dismissed, _, err := s.handlePrompt(prompt)
	if err != nil {
		return err
	}
	if dismissed {
		return errors.New("secret service prompt was dismissed")
	}
	return nil
}

func (s *secretServiceStore) handlePrompt(prompt dbus.ObjectPath) (bool, dbus.Variant, error) {
	if prompt == "" || prompt == "/" {
		return false, dbus.MakeVariant(""), nil
	}

	options := []dbus.MatchOption{
		dbus.WithMatchObjectPath(prompt),
		dbus.WithMatchInterface(secretServicePromptIfc),
	}
	if err := s.svc.AddMatchSignal(options...); err != nil {
		return false, dbus.MakeVariant(""), err
	}
	// Explicitly discard the error from RemoveMatchSignal to satisfy linters.
	defer func() { _ = s.svc.RemoveMatchSignal(options...) }()

	ch := make(chan *dbus.Signal, 1)
	s.svc.Signal(ch)
	defer s.svc.RemoveSignal(ch)

	if err := s.svc.Object(secretServiceBus, prompt).Call(secretServicePromptIfc+".Prompt", 0, "").Err; err != nil {
		return false, dbus.MakeVariant(""), err
	}

	timer := time.NewTimer(secretServicePromptTimeout)
	defer timer.Stop()

	for {
		select {
		case signal := <-ch:
			if signal == nil || signal.Path != prompt || signal.Name != secretServicePromptIfc+".Completed" || len(signal.Body) < 2 {
				continue
			}

			dismissed, _ := signal.Body[0].(bool)
			result, ok := signal.Body[1].(dbus.Variant)
			if !ok {
				return false, dbus.MakeVariant(""), errors.New("secret service prompt returned an invalid result")
			}
			return dismissed, result, nil
		case <-timer.C:
			return false, dbus.MakeVariant(""), fmt.Errorf("secret service prompt timed out after %s", secretServicePromptTimeout)
		}
	}
}

func (s *secretServiceStore) service() dbus.BusObject {
	return s.svc.Object(secretServiceBus, dbus.ObjectPath(secretServicePath))
}

func (s *secretServiceStore) collection() dbus.BusObject {
	return s.collectionAt(s.collectionPath)
}

func (s *secretServiceStore) collectionAt(path dbus.ObjectPath) dbus.BusObject {
	return s.svc.Object(secretServiceBus, path)
}
