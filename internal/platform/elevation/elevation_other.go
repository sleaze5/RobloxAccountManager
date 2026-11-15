//go:build !windows

package elevation

import "context"

const Supported = false

func IsElevated() bool { return false }

func currentUserSID() (string, error) { return "", ErrUnsupported }

func requireCurrentUser(string) error { return ErrUnsupported }

func runElevated(context.Context, []string) (uint32, error) { return 0, ErrUnsupported }
