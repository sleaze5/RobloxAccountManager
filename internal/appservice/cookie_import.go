package appservice

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	importValidationWorkers = 4
	maxImportCookies        = 1000
	maxPendingImportBatches = 8
	pendingImportTTL        = 10 * time.Minute
)

type ImportPreview struct {
	BatchID    string              `json:"batchId,omitempty"`
	Items      []ImportPreviewItem `json:"items"`
	ReadyCount int                 `json:"readyCount"`
}

type ImportPreviewItem struct {
	Index              int                  `json:"index"`
	Authenticated      bool                 `json:"authenticated"`
	Ready              bool                 `json:"ready"`
	SameAccountAsIndex int                  `json:"sameAccountAsIndex,omitempty"`
	UpdatesExisting    bool                 `json:"updatesExisting,omitempty"`
	RobloxUserID       int64                `json:"robloxUserId,omitempty"`
	Username           string               `json:"username,omitempty"`
	DisplayName        string               `json:"displayName,omitempty"`
	AvatarURL          string               `json:"avatarUrl,omitempty"`
	Moderation         *accounts.Moderation `json:"moderation,omitempty"`
	ErrorStatus        string               `json:"errorStatus,omitempty"`
	ErrorMessage       string               `json:"errorMessage,omitempty"`
}

type ImportBatchResult struct {
	Items []ImportBatchItem `json:"items"`
}

type ImportBatchItem struct {
	Index        int                  `json:"index"`
	Status       ImportStatus         `json:"status"`
	Account      accounts.AccountView `json:"account"`
	ErrorStatus  string               `json:"errorStatus,omitempty"`
	ErrorMessage string               `json:"errorMessage,omitempty"`
}

type pendingImportBatch struct {
	createdAt time.Time
	items     []pendingImportItem
	timer     *time.Timer
}

type pendingImportItem struct {
	index             int
	existingAccountID int64
	validation        roblox.CookieValidation
}

type seenImportAccount struct {
	firstIndex        int
	existingAccountID int64
}

type importValidationOutcome struct {
	validation roblox.CookieValidation
	err        error
}

func (service *Service) ValidateCookies(ctx context.Context, input string) (ImportPreview, error) {
	ctx, finishValidation := service.beginValidation(ctx)
	defer finishValidation()
	inputs := accounts.SplitCookieInputs(input)
	if len(inputs) == 0 {
		return ImportPreview{}, &roblox.Error{
			Kind:     roblox.KindInvalidCookieInput,
			Endpoint: "cookie-import",
			Message:  "Enter at least one .ROBLOSECURITY cookie.",
		}
	}
	if len(inputs) > maxImportCookies {
		return ImportPreview{}, &roblox.Error{
			Kind:     roblox.KindInvalidCookieInput,
			Endpoint: "cookie-import",
			Message:  fmt.Sprintf("Paste no more than %d cookies at once.", maxImportCookies),
		}
	}

	outcomes := service.validateImportInputs(ctx, inputs)

	preview := ImportPreview{Items: make([]ImportPreviewItem, len(inputs))}
	pending := pendingImportBatch{}
	seenUsers := make(map[int64]seenImportAccount, len(inputs))
	userIDs := make([]int64, 0, len(inputs))
	for index, outcome := range outcomes {
		previewItem := ImportPreviewItem{Index: index + 1}
		if outcome.err != nil {
			previewItem.ErrorStatus, previewItem.ErrorMessage = importErrorDetails(outcome.err)
			preview.Items[index] = previewItem
			continue
		}

		identity := outcome.validation.Identity
		previewItem.Authenticated = true
		previewItem.Ready = true
		if seen, duplicate := seenUsers[identity.RobloxUserID]; duplicate {
			previewItem.SameAccountAsIndex = seen.firstIndex
			previewItem.UpdatesExisting = seen.existingAccountID > 0
			pending.items = append(pending.items, pendingImportItem{
				index:             previewItem.Index,
				existingAccountID: seen.existingAccountID,
				validation:        outcome.validation,
			})
			preview.Items[index] = previewItem
			continue
		}

		previewItem.RobloxUserID = identity.RobloxUserID
		previewItem.Username = identity.Username
		previewItem.DisplayName = identity.DisplayName
		previewItem.Moderation = outcome.validation.Moderation
		userIDs = append(userIDs, identity.RobloxUserID)

		var existingAccountID int64
		existing, err := service.repo.FindByRobloxUserID(ctx, identity.RobloxUserID)
		switch {
		case err == nil:
			existingAccountID = existing.ID
			previewItem.UpdatesExisting = true
		case errors.Is(err, accounts.ErrNotFound):
		default:
			clearPendingImportBatch(&pending)
			return ImportPreview{}, mapRepositoryError(err)
		}
		seenUsers[identity.RobloxUserID] = seenImportAccount{firstIndex: previewItem.Index, existingAccountID: existingAccountID}
		preview.ReadyCount++
		pending.items = append(pending.items, pendingImportItem{
			index:             previewItem.Index,
			existingAccountID: existingAccountID,
			validation:        outcome.validation,
		})
		preview.Items[index] = previewItem
	}

	service.loadImportAvatars(ctx, userIDs, &preview)

	if len(pending.items) == 0 {
		return preview, nil
	}
	batchID, err := newImportBatchID()
	if err != nil {
		clearPendingImportBatch(&pending)
		return ImportPreview{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cookie-import", Message: "The validated cookies could not be staged.", Cause: err}
	}
	service.importMutex.Lock()
	now := time.Now().UTC()
	service.prunePendingImportsLocked(now)
	service.makePendingImportRoomLocked()
	pending.createdAt = now
	pending.timer = time.AfterFunc(pendingImportTTL, func() {
		service.DiscardValidatedCookies(batchID)
	})
	service.pendingImports[batchID] = pending
	service.importMutex.Unlock()
	preview.BatchID = batchID
	return preview, nil
}

func (service *Service) SaveValidatedCookies(ctx context.Context, batchID string, selectedIndexes []int) (ImportBatchResult, error) {
	batch, ok := service.takePendingImport(batchID)
	if !ok {
		return ImportBatchResult{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cookie-import", Message: "The validated cookies expired. Validate them again."}
	}
	defer clearPendingImportBatch(&batch)
	if !retainSelectedImports(&batch, selectedIndexes) {
		return ImportBatchResult{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cookie-import", Message: "The selected accounts are invalid. Validate the cookies again."}
	}

	batchResult := ImportBatchResult{Items: make([]ImportBatchItem, 0, len(batch.items))}
	for _, batchItem := range batch.items {
		if batchItem.existingAccountID > 0 {
			view, err := service.replaceValidatedCookie(ctx, batchItem.existingAccountID, batchItem.validation)
			if err != nil {
				service.logger.Error("validated account update failed", "cookie_index", batchItem.index, "roblox_user_id", batchItem.validation.Identity.RobloxUserID, "error", err)
				batchResult.Items = append(batchResult.Items, failedImportBatchItem(batchItem.index, "update error", "The cookie could not be updated."))
				continue
			}
			batchResult.Items = append(batchResult.Items, ImportBatchItem{Index: batchItem.index, Status: ImportUpdated, Account: view})
			continue
		}

		view, session, err := service.repo.Create(ctx, batchItem.validation.Identity, batchItem.validation.Cookie, batchItem.validation.ExpiresAt, batchItem.validation.Moderation != nil)
		if err != nil {
			if errors.Is(err, accounts.ErrDuplicateAccount) {
				existing, lookupError := service.repo.FindByRobloxUserID(ctx, batchItem.validation.Identity.RobloxUserID)
				if lookupError == nil {
					batchResult.Items = append(batchResult.Items, ImportBatchItem{Index: batchItem.index, Status: ImportDuplicate, Account: existing})
					continue
				}
			}
			service.logger.Error("validated account import failed", "cookie_index", batchItem.index, "roblox_user_id", batchItem.validation.Identity.RobloxUserID, "error", err)
			batchResult.Items = append(batchResult.Items, failedImportBatchItem(batchItem.index, "storage error", "The account could not be added."))
			continue
		}
		if err := service.sessions.Put(session, batchItem.validation.BrowserID); err != nil {
			if removeError := service.repo.Remove(context.WithoutCancel(ctx), view.ID); removeError != nil {
				service.logger.Error("failed import rollback failed", "account_id", view.ID, "error", removeError)
			}
			status, message := importErrorDetails(err)
			batchResult.Items = append(batchResult.Items, failedImportBatchItem(batchItem.index, status, message))
			continue
		}
		service.events.send("account:added", view)
		batchResult.Items = append(batchResult.Items, ImportBatchItem{Index: batchItem.index, Status: ImportCreated, Account: view})
	}
	return batchResult, nil
}

func retainSelectedImports(batch *pendingImportBatch, selectedIndexes []int) bool {
	if len(selectedIndexes) == 0 || len(selectedIndexes) > len(batch.items) {
		return false
	}
	selected := make(map[int]struct{}, len(selectedIndexes))
	for _, index := range selectedIndexes {
		if index <= 0 {
			return false
		}
		if _, duplicate := selected[index]; duplicate {
			return false
		}
		selected[index] = struct{}{}
	}

	available := make(map[int]struct{}, len(batch.items))
	for _, batchItem := range batch.items {
		available[batchItem.index] = struct{}{}
	}
	for index := range selected {
		if _, exists := available[index]; !exists {
			return false
		}
	}
	if !uniqueSelectedImportUsers(batch, selected) {
		return false
	}

	retained := batch.items[:0]
	for itemIndex := range batch.items {
		batchItem := batch.items[itemIndex]
		if _, keep := selected[batchItem.index]; keep {
			retained = append(retained, batchItem)
		} else {
			batch.items[itemIndex].validation = roblox.CookieValidation{}
		}
	}
	for itemIndex := len(retained); itemIndex < len(batch.items); itemIndex++ {
		batch.items[itemIndex] = pendingImportItem{}
	}
	batch.items = retained
	return len(batch.items) == len(selectedIndexes)
}

func (service *Service) DiscardValidatedCookies(batchID string) {
	if batchID == "" {
		return
	}
	service.importMutex.Lock()
	batch, ok := service.pendingImports[batchID]
	delete(service.pendingImports, batchID)
	service.importMutex.Unlock()
	if ok {
		clearPendingImportBatch(&batch)
	}
}

func (service *Service) takePendingImport(batchID string) (pendingImportBatch, bool) {
	if batchID == "" {
		return pendingImportBatch{}, false
	}
	now := time.Now().UTC()
	service.importMutex.Lock()
	service.prunePendingImportsLocked(now)
	batch, ok := service.pendingImports[batchID]
	delete(service.pendingImports, batchID)
	service.importMutex.Unlock()
	return batch, ok
}

func (service *Service) prunePendingImportsLocked(now time.Time) {
	for batchID, batch := range service.pendingImports {
		if now.Sub(batch.createdAt) <= pendingImportTTL {
			continue
		}
		clearPendingImportBatch(&batch)
		delete(service.pendingImports, batchID)
	}
}

func (service *Service) makePendingImportRoomLocked() {
	if len(service.pendingImports) < maxPendingImportBatches {
		return
	}
	var oldestID string
	var oldestTime time.Time
	for batchID, batch := range service.pendingImports {
		if oldestID == "" || batch.createdAt.Before(oldestTime) {
			oldestID = batchID
			oldestTime = batch.createdAt
		}
	}
	batch := service.pendingImports[oldestID]
	clearPendingImportBatch(&batch)
	delete(service.pendingImports, oldestID)
}

func (service *Service) clearPendingImports() {
	service.importMutex.Lock()
	for batchID, batch := range service.pendingImports {
		clearPendingImportBatch(&batch)
		delete(service.pendingImports, batchID)
	}
	service.importMutex.Unlock()
}

func clearPendingImportBatch(batch *pendingImportBatch) {
	if batch.timer != nil {
		batch.timer.Stop()
		batch.timer = nil
	}
	for index := range batch.items {
		batch.items[index].validation = roblox.CookieValidation{}
	}
	batch.items = nil
}

func newImportBatchID() (string, error) {
	value := make([]byte, 24)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate import batch ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func importErrorDetails(err error) (string, string) {
	if robloxError, ok := errors.AsType[*roblox.Error](err); ok {
		status := strings.ReplaceAll(string(robloxError.Kind), "_", " ")
		if robloxError.Status > 0 {
			status = fmt.Sprintf("%d %s", robloxError.Status, http.StatusText(robloxError.Status))
		}
		message := strings.TrimSpace(robloxError.Message)
		message = strings.TrimPrefix(message, accounts.ErrInvalidCookieInput.Error()+": ")
		if message == "" {
			message = "The cookie could not be validated."
		}
		return status, message
	}
	return "validation error", "The cookie could not be validated."
}

func failedImportBatchItem(index int, status, message string) ImportBatchItem {
	return ImportBatchItem{Index: index, Status: ImportFailed, ErrorStatus: status, ErrorMessage: message}
}

func (service *Service) validateImportInputs(ctx context.Context, inputs []string) []importValidationOutcome {
	outcomes := make([]importValidationOutcome, len(inputs))
	jobs := make(chan int, len(inputs))
	for index := range inputs {
		jobs <- index
	}
	close(jobs)

	var workers sync.WaitGroup
	for range min(importValidationWorkers, len(inputs)) {
		workers.Go(func() {
			for index := range jobs {
				cookie, err := accounts.NormalizeCookie(inputs[index])
				if err != nil {
					outcomes[index].err = &roblox.Error{Kind: roblox.KindInvalidCookieInput, Endpoint: "cookie-import", Message: err.Error()}
					continue
				}
				outcomes[index].validation, outcomes[index].err = service.users.ValidateCookie(ctx, cookie, "")
			}
		})
	}
	workers.Wait()
	return outcomes
}

func (service *Service) loadImportAvatars(ctx context.Context, userIDs []int64, preview *ImportPreview) {
	images := service.avatarURLs(ctx, userIDs)
	for index := range preview.Items {
		preview.Items[index].AvatarURL = images[preview.Items[index].RobloxUserID]
	}
}

func (service *Service) avatarURLs(ctx context.Context, userIDs []int64) map[int64]string {
	if len(userIDs) == 0 {
		return map[int64]string{}
	}
	images, err := service.headshotImages(ctx, userIDs)
	if err != nil {
		service.logger.Debug("import preview headshots unavailable", "error", err)
		return map[int64]string{}
	}
	return images
}

func uniqueSelectedImportUsers(batch *pendingImportBatch, selected map[int]struct{}) bool {
	selectedUsers := make(map[int64]struct{}, len(selected))
	for _, batchItem := range batch.items {
		if _, keep := selected[batchItem.index]; !keep {
			continue
		}
		userID := batchItem.validation.Identity.RobloxUserID
		if userID <= 0 {
			return false
		}
		if _, duplicate := selectedUsers[userID]; duplicate {
			return false
		}
		selectedUsers[userID] = struct{}{}
	}
	return true
}
