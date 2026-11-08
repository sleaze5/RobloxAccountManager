//go:build linux

package appupdate

import (
	"fmt"
	"os"
)

func installTarget(executable string) string { return executable }

func copyStaged(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open staged update: %w", err)
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		return fmt.Errorf("create update copy: %w", err)
	}
	return writeUpdateCopy(output, input)
}
