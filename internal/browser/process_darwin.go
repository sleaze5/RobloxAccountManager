//go:build darwin

package browser

import "syscall"

// --use-mock-keychain keeps Chrome from asking for Keychain access.
const credentialStoreFlag = "--use-mock-keychain"

func browserProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// Chrome on macOS keeps running after its last window closes.
func (process *unixBrowserProcess) PagesClosed() {
	process.mutex.Lock()
	defer process.mutex.Unlock()
	if process.finished {
		return
	}
	select {
	case process.windowsClosed <- struct{}{}:
	default:
	}
}
