package accounts

import "errors"

var (
	ErrNotFound         = errors.New("account not found")
	ErrDuplicateAccount = errors.New("account already exists for this Roblox user")
	ErrTagNotFound      = errors.New("account tag not found")
	ErrDuplicateTag     = errors.New("account tag already exists")
	ErrInvalidTagName   = errors.New("invalid account tag name")
	ErrInvalidOrder     = errors.New("invalid account order")
	ErrStaleSecret      = errors.New("session secret changed")
)
