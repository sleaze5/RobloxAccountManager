//go:build linux

package protection

import (
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	keyringName         = "the Secret Service"
	secretsService      = "org.freedesktop.secrets"
	secretsAlias        = "/org/freedesktop/secrets/aliases/default"
	collectionInterface = "org.freedesktop.Secret.Collection"
)

func checkKeyring() error {
	connection, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("connect to the session bus: %w", err)
	}
	value, err := connection.Object(secretsService, dbus.ObjectPath(secretsAlias)).GetProperty(collectionInterface + ".Locked")
	if err != nil {
		return fmt.Errorf("read the Secret Service collection: %w", err)
	}
	locked, ok := value.Value().(bool)
	if !ok {
		return errors.New("read the Secret Service collection lock state")
	}
	if locked {
		return errors.New("the Secret Service collection is locked")
	}
	return nil
}
