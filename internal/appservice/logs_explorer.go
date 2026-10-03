package appservice

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (service *Service) GetLogsExplorer(ctx context.Context) (logsexplorer.Snapshot, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.Snapshot{}, mapRepositoryError(vault.ErrLocked)
	}
	snapshot, err := service.logsExplorer.ReadAll(ctx)
	if err != nil {
		return logsexplorer.Snapshot{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "Roblox logs could not be read. Refresh all to try again.",
		}
	}
	return snapshot, nil
}

func (service *Service) RefreshLogsExplorerLog(ctx context.Context, fileName string) (logsexplorer.Session, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.Session{}, mapRepositoryError(vault.ErrLocked)
	}
	session, err := service.logsExplorer.ReadFile(ctx, fileName)
	if err != nil {
		return logsexplorer.Session{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "This log could not be read. It may have been removed or be unavailable. Refresh all to update the list.",
		}
	}
	return session, nil
}
