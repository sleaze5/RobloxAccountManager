package protection

import "github.com/sleaze5/RobloxAccountManager/internal/appmeta"

const purpose = appmeta.Identifier + "/autounlock/v1"

type Protector interface {
	Protect(plaintext []byte) ([]byte, error)
	Unprotect(ciphertext []byte) ([]byte, error)
}
