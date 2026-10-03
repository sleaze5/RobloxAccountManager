package appservice

import (
	"context"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

type Vault interface {
	Initialized() bool
	FileState() vault.FileState
	Unlocked() bool
	UnlockedWithDPAPI() bool
	AutomaticUnlockEnabled() bool
	ValidAutomaticUnlock() bool
	PasswordHint() string
	Create(context.Context, string, string, bool) error
	Unlock(context.Context, string) error
	AutoUnlock(context.Context) error
	Lock() error
	Close() error
	TestPassword(context.Context, string) error
	ChangePassword(context.Context, string, string, string, string, bool) error
	SetAutomaticUnlock(bool) error
	MaybeCreateBackup(context.Context) error
	CreateBackup(context.Context) error
	PurgeBackups() error
	ListBackups() ([]vault.BackupInfo, error)
	RestoreBackup(context.Context, string, string) error
	ResetVault() error
}

type AccountRepository interface {
	Create(context.Context, accounts.Identity, string, *time.Time) (accounts.AccountView, accounts.SessionRecord, error)
	FindByRobloxUserID(context.Context, int64) (accounts.AccountView, error)
	ReplaceCookie(context.Context, int64, accounts.Identity, string, *time.Time) (accounts.AccountView, accounts.SessionRecord, error)
	RotateCookie(context.Context, int64, int64, string, *time.Time) (accounts.SessionRecord, bool, error)
	Remove(context.Context, int64) error
	Page(context.Context, accounts.AccountQuery) (accounts.AccountPage, error)
	Move(context.Context, int64, *int64, *int64) error
	ListTags(context.Context) ([]accounts.TagView, error)
	CreateTag(context.Context, string) (accounts.TagView, error)
	RemoveTag(context.Context, int64) error
	SetAccountsTag(context.Context, []int64, int64, bool) ([]accounts.AccountView, error)
	GetSession(context.Context, int64) (accounts.SessionRecord, error)
	GetView(context.Context, int64) (accounts.AccountView, error)
	MarkState(context.Context, int64, int64, accounts.SessionState, string) error
	RecordValidation(context.Context, int64, int64, accounts.Identity) error
}
