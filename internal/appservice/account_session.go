package appservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) beginAccountSession(parent context.Context, accountID int64, endpoint string) (context.Context, int64, func(), error) {
	if accountID <= 0 {
		return nil, 0, nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select an account first."}
	}
	if !service.vault.Unlocked() {
		return nil, 0, nil, &roblox.Error{Kind: roblox.KindVaultLocked, Endpoint: endpoint, Message: "The account vault is locked."}
	}
	snapshot, err := service.sessions.Snapshot(accountID)
	if err != nil {
		return nil, 0, nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	stop := context.AfterFunc(snapshot.Context, cancel)
	if snapshot.Context.Err() != nil {
		cancel()
	}
	return ctx, snapshot.SecretVersion, func() { stop(); cancel() }, nil
}

type accountRead struct {
	source string
	fetch  func() error
}

// Session failures end the whole read because no section can succeed.
func (service *Service) runAccountReads(ctx context.Context, accountID, version int64, endpoint string, logger *slog.Logger, reads []accountRead) ([]string, error) {
	failures := make([]error, len(reads))
	var group sync.WaitGroup
	for index, read := range reads {
		group.Go(func() { failures[index] = read.fetch() })
	}
	group.Wait()
	if ctx.Err() != nil || !service.vault.Unlocked() || !service.sessions.VersionCurrent(accountID, version) {
		return nil, &roblox.Error{Kind: roblox.KindCancelled, Endpoint: endpoint, Message: "The request ended or the account session changed. Refresh to retry."}
	}
	unavailable := []string{}
	for index, failure := range failures {
		if failure == nil {
			continue
		}
		kind, status := roblox.KindProtocol, 0
		if remote, ok := errors.AsType[*roblox.Error](failure); ok {
			kind, status = remote.Kind, remote.Status
		}
		logger.Warn("account section unavailable", "source", reads[index].source, "error_kind", kind, "status", status)
		if kind == roblox.KindReauthRequired || kind == roblox.KindInvalidSession || kind == roblox.KindVaultLocked || kind == roblox.KindModerated {
			return nil, failure
		}
		unavailable = append(unavailable, reads[index].source)
	}
	return unavailable, nil
}
