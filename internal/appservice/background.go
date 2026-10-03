package appservice

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

func secureUint64() uint64 {
	var value [8]byte
	_, _ = rand.Read(value[:])
	return binary.LittleEndian.Uint64(value[:])
}

const (
	staleValidationInterval = 15 * time.Minute
	backgroundWorkers       = 3
)

func (service *Service) startBackground() {
	service.stopBackground()
	service.mu.Lock()
	ctx, cancel := context.WithCancel(service.lifecycle)
	service.backgroundCancel = cancel
	service.mu.Unlock()
	service.logger.Debug("background validation started", "worker_count", backgroundWorkers)
	go service.backgroundValidation(ctx)
}

func (service *Service) stopBackground() {
	service.mu.Lock()
	cancel := service.backgroundCancel
	service.backgroundCancel = nil
	service.mu.Unlock()
	if cancel != nil {
		cancel()
		service.logger.Debug("background validation stopped")
	}
}

func (service *Service) backgroundValidation(ctx context.Context) {
	jobs := make(chan int64)
	var workers sync.WaitGroup
	for range backgroundWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case accountID, ok := <-jobs:
					if !ok {
						return
					}
					if _, err := service.ValidateAccount(ctx, accountID); err != nil && ctx.Err() == nil {
						service.logger.Warn("background account validation failed", "account_id", accountID, "error", err)
					}
				}
			}
		}()
	}
	defer func() {
		close(jobs)
		workers.Wait()
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if !service.events.BackgroundPaused(time.Now()) {
			nowMS := time.Now().UTC().UnixMilli()
			var cursor *accounts.AccountCursor
			for {
				page, err := service.repo.Page(ctx, accounts.AccountQuery{Cursor: cursor})
				if err != nil {
					service.logger.Error("background account list failed", "error", err)
					return
				}
				for _, view := range page.Accounts {
					if view.State == accounts.StateDisabled || view.State == accounts.StateReauthRequired || view.LastValidatedAtMS != nil && nowMS-*view.LastValidatedAtMS < staleValidationInterval.Milliseconds() {
						continue
					}
					jitter := time.NewTimer(time.Duration(secureUint64()%1500) * time.Millisecond)
					select {
					case <-ctx.Done():
						jitter.Stop()
						return
					case <-jitter.C:
					}
					select {
					case jobs <- view.ID:
					case <-ctx.Done():
						return
					}
				}
				if page.NextCursor == nil {
					break
				}
				cursor = page.NextCursor
			}
		}
		timer.Reset(time.Minute)
	}
}
