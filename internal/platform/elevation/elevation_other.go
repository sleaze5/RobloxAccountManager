//go:build !windows

package elevation

import (
	"context"
	"io"
)

const Supported = false

func IsElevated() bool { return false }

func startHelper(context.Context) (*helperConnection, error) {
	return nil, errUnsupported
}

func connectToApplication([]string) (io.ReadCloser, io.WriteCloser, error) {
	return nil, nil, errUnsupported
}
