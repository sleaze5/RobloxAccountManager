package appservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	transferstore "github.com/sleaze5/RobloxAccountManager/internal/storage/transfer"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

const transferPreviewTTL = 10 * time.Minute

type TransferMode string

const (
	TransferOverwrite TransferMode = "overwrite"
	TransferMerge     TransferMode = "merge"
)

type TransferPreview struct {
	ID                string            `json:"id"`
	ExportedAtMS      int64             `json:"exportedAtMs"`
	Accounts          []TransferAccount `json:"accounts"`
	FavoritePlaces    int               `json:"favoritePlaces"`
	NewFavoritePlaces int               `json:"newFavoritePlaces"`
}

type TransferAccount struct {
	RobloxUserID int64    `json:"robloxUserId"`
	Username     string   `json:"username"`
	DisplayName  string   `json:"displayName"`
	AvatarURL    string   `json:"avatarUrl,omitempty"`
	Existing     bool     `json:"existing"`
	Favorite     bool     `json:"favorite"`
	Tags         []string `json:"tags"`
}

type TransferRequest struct {
	Mode             TransferMode          `json:"mode"`
	RobloxUserIDs    []int64               `json:"robloxUserIds"`
	FavoritePlaces   bool                  `json:"favoritePlaces"`
	SettingsSections []appsettings.Section `json:"settingsSections"`
}

type TransferSettingsStatus string

const (
	TransferSettingsSkipped  TransferSettingsStatus = "skipped"
	TransferSettingsImported TransferSettingsStatus = "imported"
	TransferSettingsPending  TransferSettingsStatus = "pending"
)

type TransferResult struct {
	Added          int                    `json:"added"`
	Updated        int                    `json:"updated"`
	KeptCookies    []TransferKeptCookie   `json:"keptCookies"`
	FavoritePlaces int                    `json:"favoritePlaces"`
	Settings       TransferSettingsStatus `json:"settings"`
}

type TransferKeptCookie struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

type pendingTransfer struct {
	id       string
	payload  transferstore.Payload
	settings appsettings.Settings
	timer    *time.Timer
}

func transferError(message string, cause error) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: message, Cause: cause}
}

func (service *Service) ExportsDirectory() string { return service.vault.ExportsDirectory() }

func (service *Service) DefaultExportName() string { return service.vault.DefaultExportName() }

func (service *Service) ValidateExportPassword(password, confirmation, hint string) error {
	if password != confirmation {
		return transferError("The export passwords do not match.", nil)
	}
	if err := vault.ValidateExportPassword(password); err != nil {
		return transferError("The export password does not meet every requirement.", err)
	}
	if err := vault.ValidateExportHint(hint, password); errors.Is(err, vault.ErrExportHintPassword) {
		return transferError("The password hint cannot contain the export password.", err)
	} else if err != nil {
		return transferError("The password hint is too long.", err)
	}
	return nil
}

func (service *Service) ExportVault(ctx context.Context, path, password, confirmation, hint string) (vault.ExportInfo, error) {
	if err := service.ValidateExportPassword(password, confirmation, hint); err != nil {
		return vault.ExportInfo{}, err
	}
	data, err := service.exportData(ctx, "vault-export")
	if err != nil {
		return vault.ExportInfo{}, err
	}
	defer clear(data)
	info, err := service.vault.WriteExport(path, password, hint, data)
	if err != nil {
		service.logger.Error("export file could not be written", "operation", "vault-export", "error", err)
		return vault.ExportInfo{}, transferError("The export file could not be saved. Check that the folder is writable and has free space, then try again.", err)
	}
	service.logger.Info("vault exported", "operation", "vault-export")
	return info, nil
}

func (service *Service) exportData(ctx context.Context, operation string) ([]byte, error) {
	payload, err := service.transfers.Read(ctx)
	if err != nil {
		if errors.Is(err, vault.ErrLocked) {
			return nil, mapRepositoryError(err)
		}
		service.logger.Error("vault data could not be read", "operation", operation, "error", err)
		return nil, transferError("The vault data could not be read.", err)
	}
	if payload.Settings, err = service.settings.Export(); err != nil {
		service.logger.Error("settings could not be read", "operation", operation, "error", err)
		return nil, transferError("The settings could not be read.", err)
	}
	data, err := transferstore.Encode(payload)
	if err != nil {
		service.logger.Error("vault data could not be encoded", "operation", operation, "error", err)
		return nil, transferError("The vault data could not be encoded.", err)
	}
	service.logger.Info("vault data read", "operation", operation, "accounts", len(payload.Accounts), "tags", len(payload.Tags), "favorite_places", len(payload.FavoritePlaces))
	return data, nil
}

// ExportDetails returns the format version and password hint of an export file.
func (service *Service) ExportDetails(path string) (vault.ExportDetails, error) {
	details, err := service.vault.ReadExportDetails(path)
	if err != nil {
		return vault.ExportDetails{}, service.exportReadError(err)
	}
	return details, nil
}

func (service *Service) OpenVaultExport(ctx context.Context, path, password string) (TransferPreview, error) {
	info, data, err := service.vault.ReadExport(path, password)
	if err != nil {
		return TransferPreview{}, service.exportReadError(err)
	}
	payload, err := transferstore.Decode(data)
	clear(data)
	if errors.Is(err, transferstore.ErrUnsupportedVersion) {
		return TransferPreview{}, transferError("This export is from a newer version. Update the app to import it.", err)
	}
	if err != nil {
		service.logger.Warn("export data could not be decoded", "operation", "vault-import", "error", err)
		return TransferPreview{}, transferError("The export file is damaged.", err)
	}
	settings, err := appsettings.Decode(payload.Settings)
	if err != nil {
		service.logger.Warn("exported settings could not be decoded", "operation", "vault-import", "error", err)
		return TransferPreview{}, transferError("The settings in this export could not be read. Update the app and try again.", err)
	}
	preview, err := service.transferPreview(ctx, payload, info.CreatedAtMS)
	if err != nil {
		return TransferPreview{}, err
	}
	preview.ID, err = newImportBatchID()
	if err != nil {
		return TransferPreview{}, transferError("The export could not be opened.", err)
	}
	service.storePendingTransfer(&pendingTransfer{id: preview.ID, payload: payload, settings: settings})
	service.logger.Info("export opened for import", "operation", "vault-import", "accounts", len(payload.Accounts))
	return preview, nil
}

func (service *Service) exportReadError(err error) error {
	switch {
	case errors.Is(err, vault.ErrExportPassword):
		return transferError("The export password is incorrect, or the file is damaged.", err)
	case errors.Is(err, vault.ErrUnsupportedExport):
		return transferError("This export is from a newer version. Update the app to import it.", err)
	case errors.Is(err, vault.ErrInvalidExport):
		return transferError("This file is not a valid export.", err)
	}
	service.logger.Warn("export file could not be read", "operation", "vault-import", "error", err)
	return transferError("The export file could not be read.", err)
}

func (service *Service) transferPreview(ctx context.Context, payload transferstore.Payload, exportedAtMS int64) (TransferPreview, error) {
	preview := TransferPreview{ExportedAtMS: exportedAtMS, Accounts: make([]TransferAccount, 0, len(payload.Accounts)), FavoritePlaces: len(payload.FavoritePlaces)}
	for _, account := range payload.Accounts {
		_, err := service.repo.FindByRobloxUserID(ctx, account.RobloxUserID)
		if err != nil && !errors.Is(err, accounts.ErrNotFound) {
			return TransferPreview{}, mapRepositoryError(err)
		}
		preview.Accounts = append(preview.Accounts, TransferAccount{
			RobloxUserID: account.RobloxUserID,
			Username:     transferUsername(account),
			DisplayName:  account.DisplayName,
			Existing:     err == nil,
			Favorite:     account.Favorite,
			Tags:         account.Tags,
		})
	}
	userIDs := make([]int64, len(preview.Accounts))
	for index, account := range preview.Accounts {
		userIDs[index] = account.RobloxUserID
	}
	images := service.avatarURLs(ctx, userIDs)
	for index := range preview.Accounts {
		preview.Accounts[index].AvatarURL = images[preview.Accounts[index].RobloxUserID]
	}
	favorites, err := service.favoritePlaces.List(ctx)
	if err != nil {
		return TransferPreview{}, mapRepositoryError(err)
	}
	for _, place := range payload.FavoritePlaces {
		if !slices.ContainsFunc(favorites, func(favorite games.Place) bool { return favorite.PlaceID == place.PlaceID }) {
			preview.NewFavoritePlaces++
		}
	}
	return preview, nil
}

func (service *Service) ImportVaultExport(ctx context.Context, id string, request TransferRequest) (TransferResult, error) {
	for _, section := range request.SettingsSections {
		if !slices.Contains(appsettings.Sections, section) {
			return TransferResult{}, transferError("The selected settings are invalid.", nil)
		}
	}
	pending, ok := service.takePendingTransfer(id)
	if !ok {
		return TransferResult{}, transferError("The opened export expired. Open it again.", nil)
	}
	defer clearPendingTransfer(pending)
	switch request.Mode {
	case TransferOverwrite:
		return service.overwriteVaultData(ctx, pending)
	case TransferMerge:
		return service.mergeVaultData(ctx, pending, request)
	default:
		return TransferResult{}, transferError("The import mode is invalid.", nil)
	}
}

func (service *Service) DiscardVaultExport(id string) {
	if pending, ok := service.takePendingTransfer(id); ok {
		clearPendingTransfer(pending)
	}
}

func (service *Service) overwriteVaultData(ctx context.Context, pending *pendingTransfer) (TransferResult, error) {
	logger := service.logger.With("operation", "vault-import", "mode", TransferOverwrite)
	if err := service.vault.CreateBackup(ctx); err != nil {
		logger.Error("vault could not be backed up before replacing it", "error", err)
		return TransferResult{}, transferError("The current vault could not be backed up, so nothing was imported. Check free disk space and try again.", err)
	}
	removed, err := service.transfers.Replace(ctx, pending.payload, pendingSettings(pending, appsettings.Sections))
	if err != nil {
		logger.Error("vault data could not be replaced", "error", err)
		return TransferResult{}, transferError("The export could not be imported. Nothing was changed.", err)
	}
	service.clearPendingImports()
	for _, accountID := range removed {
		service.sessions.Remove(accountID)
		if service.browser != nil {
			service.browser.DetachAccount(accountID, "The vault was replaced by an import. This browser is no longer synchronized.")
		}
		service.events.send("account:removed", struct {
			AccountID int64 `json:"accountId"`
		}{AccountID: accountID})
	}
	settings := service.finishSettingsImport(ctx, logger)
	service.events.CredentialBackupChanged(service.refreshCredentialBackup(ctx, logger, false))
	return TransferResult{Added: len(pending.payload.Accounts), FavoritePlaces: len(pending.payload.FavoritePlaces), KeptCookies: []TransferKeptCookie{}, Settings: settings}, nil
}

func (service *Service) mergeVaultData(ctx context.Context, pending *pendingTransfer, request TransferRequest) (TransferResult, error) {
	logger := service.logger.With("operation", "vault-import", "mode", TransferMerge)
	selected, err := selectedTransferAccounts(pending.payload.Accounts, request.RobloxUserIDs)
	if err != nil {
		return TransferResult{}, err
	}
	if len(selected) == 0 && !request.FavoritePlaces && len(request.SettingsSections) == 0 {
		return TransferResult{}, transferError("Choose something to import.", nil)
	}
	replacements, kept, err := service.validateTransferCookies(ctx, selected)
	if err != nil {
		return TransferResult{}, err
	}
	input := transferstore.MergeInput{Accounts: selected, Replacements: replacements, Settings: pendingSettings(pending, request.SettingsSections)}
	if request.FavoritePlaces {
		input.FavoritePlaces = pending.payload.FavoritePlaces
	}
	merged, err := service.transfers.Merge(ctx, input)
	if err != nil {
		logger.Error("vault data could not be merged", "error", err)
		return TransferResult{}, transferError("The export could not be imported. Nothing was changed.", err)
	}
	for _, record := range merged.Replaced {
		if err := service.sessions.ApplyRotation(record); err != nil {
			logger.Warn("imported cookie saved but runtime session changed before reconciliation", "account_id", record.AccountID, "error", err)
		}
		if service.browser != nil {
			service.browser.DetachAccount(record.AccountID, "The saved cookie changed elsewhere. This browser is no longer synchronized.")
		}
	}
	settings := TransferSettingsSkipped
	if input.Settings != nil {
		settings = service.finishSettingsImport(ctx, logger)
	}
	if len(merged.Replaced) > 0 {
		service.events.CredentialBackupChanged(service.refreshCredentialBackup(ctx, logger, true))
	}
	return TransferResult{
		Added:          len(merged.Added),
		Updated:        len(merged.Updated) + len(merged.Replaced),
		KeptCookies:    kept,
		FavoritePlaces: merged.FavoritePlaces,
		Settings:       settings,
	}, nil
}

func pendingSettings(pending *pendingTransfer, sections []appsettings.Section) *transferstore.PendingSettings {
	if len(sections) == 0 {
		return nil
	}
	names := make([]string, len(sections))
	for index, section := range sections {
		names[index] = string(section)
	}
	return &transferstore.PendingSettings{Data: pending.payload.Settings, Sections: names}
}

func (service *Service) finishSettingsImport(ctx context.Context, logger *slog.Logger) TransferSettingsStatus {
	if err := service.applyPendingSettings(ctx); err != nil {
		logger.Warn("imported settings will be saved on the next unlock", "error", err)
		return TransferSettingsPending
	}
	return TransferSettingsImported
}

// applyPendingSettings saves settings that an import committed to the vault. It runs after each import and each unlock,
// so imported settings are saved even if the application stopped after the vault transaction.
func (service *Service) applyPendingSettings(ctx context.Context) error {
	pending, err := service.transfers.PendingSettings(ctx)
	if err != nil || pending == nil {
		return err
	}
	settings, err := appsettings.Decode(pending.Data)
	if err != nil {
		if clearErr := service.transfers.ClearPendingSettings(ctx); clearErr != nil {
			return errors.Join(err, clearErr)
		}
		return fmt.Errorf("decode pending settings: %w", err)
	}
	sections := make([]appsettings.Section, len(pending.Sections))
	for index, section := range pending.Sections {
		sections[index] = appsettings.Section(section)
	}
	if err := service.settings.Import(settings, sections); err != nil {
		return fmt.Errorf("save imported settings: %w", err)
	}
	if slices.Contains(sections, appsettings.SectionLogging) {
		if err := service.applyLoggingSettings(); err != nil {
			service.logger.Warn("imported logging settings could not be applied", "operation", "vault-import", "error", err)
		}
	}
	if slices.Contains(sections, appsettings.SectionRoblox) {
		service.launchMu.Lock()
		service.multiInstance.SetEnabled(service.settings.Roblox().MultiInstance)
		service.launchMu.Unlock()
	}
	if err := service.transfers.ClearPendingSettings(ctx); err != nil {
		return fmt.Errorf("clear saved settings import: %w", err)
	}
	return nil
}

func selectedTransferAccounts(available []transferstore.Account, robloxUserIDs []int64) ([]transferstore.Account, error) {
	wanted := make(map[int64]struct{}, len(robloxUserIDs))
	for _, userID := range robloxUserIDs {
		if _, duplicate := wanted[userID]; duplicate {
			return nil, transferError("The selected accounts are invalid. Open the export again.", nil)
		}
		wanted[userID] = struct{}{}
	}
	selected := make([]transferstore.Account, 0, len(wanted))
	for _, account := range available {
		if _, ok := wanted[account.RobloxUserID]; ok {
			selected = append(selected, account)
		}
	}
	if len(selected) != len(wanted) {
		return nil, transferError("The selected accounts are invalid. Open the export again.", nil)
	}
	return selected, nil
}

// validateTransferCookies checks each imported cookie that differs from the saved one, so a stale export never replaces a working login.
func (service *Service) validateTransferCookies(ctx context.Context, selected []transferstore.Account) (map[int64]transferstore.Replacement, []TransferKeptCookie, error) {
	candidates := make([]transferstore.Account, 0, len(selected))
	for _, account := range selected {
		existing, err := service.repo.FindByRobloxUserID(ctx, account.RobloxUserID)
		if errors.Is(err, accounts.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, nil, mapRepositoryError(err)
		}
		session, err := service.repo.GetSession(ctx, existing.ID)
		if err != nil {
			return nil, nil, mapRepositoryError(err)
		}
		if session.Roblosecurity != account.Cookie {
			candidates = append(candidates, account)
		}
	}
	replacements := make(map[int64]transferstore.Replacement, len(candidates))
	kept := []TransferKeptCookie{}
	if len(candidates) == 0 {
		return replacements, kept, nil
	}
	cookies := make([]string, len(candidates))
	for index, account := range candidates {
		cookies[index] = account.Cookie
	}
	validationCtx, finishValidation := service.beginValidation(ctx)
	outcomes := service.validateImportInputs(validationCtx, cookies)
	finishValidation()
	for index, outcome := range outcomes {
		account := candidates[index]
		switch {
		case outcome.err != nil:
			_, reason := importErrorDetails(outcome.err)
			kept = append(kept, TransferKeptCookie{Username: transferUsername(account), Reason: reason})
		case outcome.validation.Identity.RobloxUserID != account.RobloxUserID:
			kept = append(kept, TransferKeptCookie{Username: transferUsername(account), Reason: "The imported cookie belongs to a different Roblox account."})
		default:
			replacements[account.RobloxUserID] = transferstore.Replacement{Identity: outcome.validation.Identity, Cookie: outcome.validation.Cookie, ExpiresAt: outcome.validation.ExpiresAt}
		}
		outcomes[index] = importValidationOutcome{}
	}
	return replacements, kept, nil
}

func transferUsername(account transferstore.Account) string {
	if account.Username != "" {
		return account.Username
	}
	return fmt.Sprintf("User %d", account.RobloxUserID)
}

func (service *Service) storePendingTransfer(pending *pendingTransfer) {
	pending.timer = time.AfterFunc(transferPreviewTTL, func() { service.DiscardVaultExport(pending.id) })
	service.transferMu.Lock()
	previous := service.pendingTransfer
	service.pendingTransfer = pending
	service.transferMu.Unlock()
	if previous != nil {
		clearPendingTransfer(previous)
	}
}

func (service *Service) takePendingTransfer(id string) (*pendingTransfer, bool) {
	service.transferMu.Lock()
	defer service.transferMu.Unlock()
	pending := service.pendingTransfer
	if pending == nil || id == "" || pending.id != id {
		return nil, false
	}
	service.pendingTransfer = nil
	return pending, true
}

func (service *Service) discardPendingTransfer() {
	service.transferMu.Lock()
	pending := service.pendingTransfer
	service.pendingTransfer = nil
	service.transferMu.Unlock()
	if pending != nil {
		clearPendingTransfer(pending)
	}
}

func clearPendingTransfer(pending *pendingTransfer) {
	if pending.timer != nil {
		pending.timer.Stop()
	}
	clear(pending.payload.Accounts)
	clear(pending.payload.Settings)
	pending.payload = transferstore.Payload{}
}
