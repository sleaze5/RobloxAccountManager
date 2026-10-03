package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/logsexplorer"
)

func (service *Service) GetLogsExplorer(ctx context.Context) (logsexplorer.Snapshot, error) {
	return service.core.GetLogsExplorer(ctx)
}

func (service *Service) RefreshLogsExplorerLog(ctx context.Context, fileName string) (logsexplorer.Session, error) {
	return service.core.RefreshLogsExplorerLog(ctx, fileName)
}
