package appservice

import (
	"context"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
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

func (service *Service) FindGameServerVisit(ctx context.Context, placeID int64, jobID string) (logsexplorer.ServerVisit, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.ServerVisit{}, mapRepositoryError(vault.ErrLocked)
	}
	if err := validGameID(placeID); err != nil {
		return logsexplorer.ServerVisit{}, err
	}
	if !games.ValidJobID(jobID) {
		return logsexplorer.ServerVisit{}, gamesInputError("The server is invalid. Search again.")
	}
	visit, err := service.logsExplorer.FindServerVisit(ctx, strconv.FormatInt(placeID, 10), jobID)
	if err != nil {
		return logsexplorer.ServerVisit{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "Roblox logs could not be read.",
		}
	}
	return visit, nil
}

// readRobloxLogsAhead parses every Roblox log once, so that the reader's cache
// is filled before the first server visit is looked up. Parsing many logs can
// take a while, and a later read only parses logs that changed.
func (service *Service) readRobloxLogsAhead(ctx context.Context) {
	if _, err := service.logsExplorer.ReadAll(ctx); err != nil && ctx.Err() == nil {
		service.logs.Module("logs-explorer").Debug("Roblox logs could not be read ahead", "operation", "logs-read-ahead", "error", err)
	}
}
