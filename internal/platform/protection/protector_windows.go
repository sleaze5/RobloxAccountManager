//go:build windows

package protection

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cryptProtectUIForbidden = 0x1

var applicationEntropy = []byte(purpose)

type windowsProtector struct{}

func New() Protector {
	return windowsProtector{}
}

func (windowsProtector) Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("protect data: plaintext is empty")
	}
	inputBlob := dataBlob(plaintext)
	entropy := dataBlob(applicationEntropy)
	var outputBlob windows.DataBlob
	if err := windows.CryptProtectData(&inputBlob, nil, &entropy, 0, nil, cryptProtectUIForbidden, &outputBlob); err != nil {
		return nil, fmt.Errorf("protect data with DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outputBlob.Data)))
	return copyBlob(outputBlob), nil
}

func (windowsProtector) Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("unprotect data: ciphertext is empty")
	}
	inputBlob := dataBlob(ciphertext)
	entropy := dataBlob(applicationEntropy)
	var outputBlob windows.DataBlob
	if err := windows.CryptUnprotectData(&inputBlob, nil, &entropy, 0, nil, cryptProtectUIForbidden, &outputBlob); err != nil {
		return nil, fmt.Errorf("unprotect data with DPAPI: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(outputBlob.Data)))
	return copyBlob(outputBlob), nil
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
