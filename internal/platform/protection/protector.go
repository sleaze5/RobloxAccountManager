package protection

// Protector secures data for the current operating-system user.
// Implementations must not display operating-system prompts.
type Protector interface {
	Protect(plaintext []byte) ([]byte, error)
	Unprotect(ciphertext []byte) ([]byte, error)
}
