package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appupdate"
)

func (service *Service) GetUpdateState() appupdate.State { return service.updates.State() }

func (service *Service) CheckForUpdate(ctx context.Context) error { return service.updates.Check(ctx) }

func (service *Service) InstallUpdate(ctx context.Context) error { return service.updates.Install(ctx) }

func (service *Service) CancelUpdateDownload() { service.updates.CancelDownload() }

func (service *Service) RestartToUpdate(ctx context.Context) error {
	return service.updates.Restart(ctx)
}
