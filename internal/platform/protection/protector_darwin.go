//go:build darwin

package protection

const keyringName = "the Keychain"

func checkKeyring() error { return nil }
