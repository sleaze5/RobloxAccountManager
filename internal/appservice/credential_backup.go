package appservice

import (
	"context"
	"log/slog"
	"time"
)

func (service *Service) refreshCredentialBackup(ctx context.Context, logger *slog.Logger, purge bool) string {
	if purge {
		if err := service.vault.PurgeBackups(); err != nil {
			logger.Warn("cookie saved but old backups could not be removed", "error", err)
			return "The cookie change is saved, but old backups could not be removed. Check the logs for details."
		}
	}
	backupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := service.vault.CreateBackup(backupCtx); err != nil {
		logger.Warn("cookie saved but fresh backup failed", "error", err)
		return "The cookie change is saved, but a fresh backup could not be created. Check free disk space and the logs."
	}
	return ""
}
