package appservice

import (
	"context"
	"log/slog"
	"time"
)

func (service *Service) refreshCredentialBackup(ctx context.Context, logger *slog.Logger, purge bool) string {
	if purge {
		if err := service.vault.PurgeBackups(); err != nil {
			logger.WarnContext(ctx, "cookie saved, but could not remove old backups", "error", err)
			return "The cookie change is saved, but old backups were not removed. Check the logs."
		}
	}
	backupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := service.vault.CreateBackup(backupContext); err != nil {
		logger.WarnContext(ctx, "cookie saved, but could not create a new backup", "error", err)
		return "The cookie change is saved, but the app did not create a backup. Check free disk space and the logs."
	}
	return ""
}
