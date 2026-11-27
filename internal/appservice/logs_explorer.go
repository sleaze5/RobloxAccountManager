package appservice

import (
	"context"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (service *Service) GetLogsExplorer(ctx context.Context) (logsexplorer.Snapshot, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.Snapshot{}, mapRepositoryError(vault.ErrLocked)
	}
	ctx = logging.StartOperation(ctx, "logs-explorer")
	snapshot, err := service.logsExplorer.ReadAll(ctx)
	if err != nil {
		return logsexplorer.Snapshot{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "Could not read Roblox logs. Refresh all logs to try again.",
		}
	}
	return snapshot, nil
}

func (service *Service) RefreshLogsExplorerLog(ctx context.Context, fileName string) (logsexplorer.Session, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.Session{}, mapRepositoryError(vault.ErrLocked)
	}
	ctx = logging.StartOperation(ctx, "logs-explorer")
	session, err := service.logsExplorer.ReadFile(ctx, fileName)
	if err != nil {
		return logsexplorer.Session{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "Could not read this log. It may be gone. Refresh all logs to update the list.",
		}
	}
	return session, nil
}

func (service *Service) FindGameServerVisit(ctx context.Context, placeID int64, serverID string) (logsexplorer.ServerVisit, error) {
	if !service.vault.Unlocked() {
		return logsexplorer.ServerVisit{}, mapRepositoryError(vault.ErrLocked)
	}
	if err := validRobloxID(placeID); err != nil {
		return logsexplorer.ServerVisit{}, err
	}
	if !games.ValidServerID(serverID) {
		return logsexplorer.ServerVisit{}, gamesInputError("The server is invalid. Search again.")
	}
	ctx = logging.StartOperation(ctx, "logs-explorer")
	visit, err := service.logsExplorer.FindServerVisit(ctx, strconv.FormatInt(placeID, 10), serverID)
	if err != nil {
		return logsexplorer.ServerVisit{}, &roblox.Error{
			Kind: roblox.KindProtocol, Endpoint: "logs-explorer", Cause: err,
			Message: "Could not read Roblox logs.",
		}
	}
	return visit, nil
}

// readRobloxLogsAhead parses every Roblox log once, so that the reader's cache
// is filled before the first server visit is looked up. Parsing many logs can
// take a while, and a later read only parses logs that changed.
func (service *Service) readRobloxLogsAhead(ctx context.Context) {
	ctx = logging.StartOperation(ctx, "logs-explorer")
	if _, err := service.logsExplorer.ReadAll(ctx); err != nil && ctx.Err() == nil {
		service.logs.Module("logs-explorer").DebugContext(ctx, "could not read Roblox logs ahead", "operation", "logs-read-ahead", "error", err)
	}
}
