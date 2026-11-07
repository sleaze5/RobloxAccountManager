//go:build linux

package browser

import "syscall"

const credentialStoreFlag = "--password-store=basic"

// A separate process group lets Terminate stop Chrome and its helpers
// together. Pdeathsig covers an application crash.
func browserProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
}

func (process *unixBrowserProcess) PagesClosed() {}
