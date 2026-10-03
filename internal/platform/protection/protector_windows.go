//go:build windows

package protection

import (
	"fmt"
	"unsafe"

	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
	"golang.org/x/sys/windows"
)

const cryptProtectUIForbidden = 0x1

var applicationEntropy = []byte(appmeta.Identifier + "/autounlock/v1")

type windowsProtector struct{}

// New returns the Windows data protector.
func New() Protector {
	return windowsProtector{}
}

func (windowsProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("protect data: plaintext is empty")
	}
	in := dataBlob(plaintext)
	entropy := dataBlob(applicationEntropy)
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, &entropy, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return nil, fmt.Errorf("protect data with DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return copyBlob(out), nil
}

func (windowsProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("unprotect data: ciphertext is empty")
	}
	in := dataBlob(ciphertext)
	entropy := dataBlob(applicationEntropy)
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, &entropy, 0, nil, cryptProtectUIForbidden, &out); err != nil {
		return nil, fmt.Errorf("unprotect data with DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return copyBlob(out), nil
}

func dataBlob(data []byte) windows.DataBlob {
	return windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}

func copyBlob(blob windows.DataBlob) []byte {
	if blob.Size == 0 || blob.Data == nil {
		return nil
	}
	return append([]byte(nil), unsafe.Slice(blob.Data, int(blob.Size))...)
}
