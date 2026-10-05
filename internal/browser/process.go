package browser

import (
	"io"
	"time"
)

type ProcessOptions struct {
	Executable   string
	UserDataPath string
}

type BrowserProcess interface {
	CDPPipes() (io.WriteCloser, io.ReadCloser)
	AllWindowsClosed() <-chan struct{}
	// PagesClosed tells the process that the browser has no open pages left.
	PagesClosed()
	Focus() error
	Exited() <-chan error
	ExitError() error
	Wait(time.Duration) bool
	Terminate() error
	Close() error
}
