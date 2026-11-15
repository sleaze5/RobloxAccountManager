//go:build !windows

package elevation

import (
	"context"
	"errors"
	"io"
)

const Supported = false

var errUnsupported = errors.New("administrator rights are only available on Windows")

func IsElevated() bool { return false }

func startHelper(context.Context) (*helperConnection, error) {
	return nil, errUnsupported
}

func connectToApplication([]string) (io.ReadCloser, io.WriteCloser, error) {
	return nil, nil, errUnsupported
}
