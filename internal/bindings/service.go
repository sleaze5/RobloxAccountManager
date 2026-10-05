package bindings

import (
	"context"
	"errors"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/appsettings"
	"github.com/sleaze5/RobloxAccountManager/internal/appupdate"
	"github.com/sleaze5/RobloxAccountManager/internal/browser"
	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
	"github.com/sleaze5/RobloxAccountManager/internal/timestampformat"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	core     *appservice.Service
	location *appservice.Location
	updates  *appupdate.Service
	events   *appservice.Events
	launch   *logging.Launch

	startMu  sync.Mutex
	startCtx context.Context
	started  bool
}

func NewService(core *appservice.Service, location *appservice.Location, updates *appupdate.Service, events *appservice.Events, launch *logging.Launch) *Service {
	return &Service{core: core, location: location, updates: updates, events: events, launch: launch}
}

func (service *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	service.launch.Begin("service-startup")
	if app := application.Get(); app != nil {
		service.events.SetEmitter(func(name string, data any) {
			app.Event.Emit(name, data)
			if name != "game:launch-confirmation-changed" || service.core.GetLaunchConfirmation().ID == "" {
				return
			}
			if windows := app.Window.GetAll(); len(windows) > 0 {
				window := windows[0]
				if window.IsMinimised() {
					window.UnMinimise()
				}
				window.Show()
				window.Focus()
			}
		})
	}
	service.startMu.Lock()
	service.startCtx = ctx
	service.startMu.Unlock()
	if err := service.startCore(); err != nil {
		return err
	}
	if app := application.Get(); app != nil {
		service.updates.Start(ctx, app)
	}
	service.launch.WaitForWebview()
	return nil
}

func (service *Service) ServiceShutdown() error {
	service.events.SetEmitter(nil)
	service.updates.Shutdown()
	return service.core.Shutdown()
}

func (service *Service) GetLaunchReport() logging.LaunchReport { return service.launch.Report() }

func (service *Service) GetAppLocation() appdata.Location { return service.location.State() }

// ConfirmAppLocation sets up the data root of mode. It reports whether the
// application restarts to use it.
func (service *Service) ConfirmAppLocation(mode appdata.Mode) (bool, error) {
	restarting, err := service.location.Confirm(mode)
	if err != nil || restarting {
		return restarting, err
	}
	return false, service.startCore()
}

// startCore starts vault startup only for a confirmed data root, so automatic
// unlock and migrations never touch a vault that the user has not chosen.
func (service *Service) startCore() error {
	service.startMu.Lock()
	defer service.startMu.Unlock()
	if service.started || service.startCtx == nil || !service.location.State().Initialized {
		return nil
	}
	if err := service.core.Start(service.startCtx); err != nil {
		return err
	}
	service.started = true
	return nil
}

func (service *Service) GetVaultState() appservice.VaultState { return service.core.GetVaultState() }

func (service *Service) GetUpdateState() appupdate.State { return service.updates.State() }

func (service *Service) CheckForUpdate(ctx context.Context) error { return service.updates.Check(ctx) }

func (service *Service) InstallUpdate(ctx context.Context) error { return service.updates.Install(ctx) }

func (service *Service) CancelUpdateDownload() { service.updates.CancelDownload() }

func (service *Service) RestartToUpdate(ctx context.Context) error {
	return service.updates.Restart(ctx)
}

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64) (appservice.AccountProfileSnapshot, error) {
	if err := positiveID(accountID, "account-profile"); err != nil {
		return appservice.AccountProfileSnapshot{}, err
	}
	return service.core.GetAccountProfile(ctx, accountID)
}

func (service *Service) GetAccountInfo(ctx context.Context, accountID int64) (appservice.AccountInfoSnapshot, error) {
	if err := positiveID(accountID, "account-info"); err != nil {
		return appservice.AccountInfoSnapshot{}, err
	}
	return service.core.GetAccountInfo(ctx, accountID)
}

func (service *Service) GetAccountSettings(ctx context.Context, accountID int64) (appservice.AccountSettingsSnapshot, error) {
	if err := positiveID(accountID, "account-settings"); err != nil {
		return appservice.AccountSettingsSnapshot{}, err
	}
	return service.core.GetAccountSettings(ctx, accountID)
}

func (service *Service) UpdateAccountSetting(ctx context.Context, accountID int64, change appservice.AccountSettingChange) (appservice.AccountSettingUpdateResult, error) {
	if err := positiveID(accountID, "account-settings-update"); err != nil {
		return appservice.AccountSettingUpdateResult{}, err
	}
	return service.core.UpdateAccountSetting(ctx, accountID, change)
}

func (service *Service) GetChatConversations(ctx context.Context, accountID int64, cursor string) (appservice.ChatConversationPage, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationPage{}, err
	}
	return service.core.GetChatConversations(ctx, accountID, cursor)
}

func (service *Service) GetChatMessages(ctx context.Context, accountID int64, conversationID, cursor string) (appservice.ChatMessagePage, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatMessagePage{}, err
	}
	return service.core.GetChatMessages(ctx, accountID, conversationID, cursor)
}

func (service *Service) MarkChatConversationRead(ctx context.Context, accountID int64, conversationID string) error {
	if err := positiveID(accountID, "chat"); err != nil {
		return err
	}
	return service.core.MarkChatConversationRead(ctx, accountID, conversationID)
}

func (service *Service) MarkAllChatConversationsRead(ctx context.Context, accountID int64) (int, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return 0, err
	}
	return service.core.MarkAllChatConversationsRead(ctx, accountID)
}

func (service *Service) CreateChatConversation(ctx context.Context, accountID int64, username string) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.CreateChatConversation(ctx, accountID, username)
}

func (service *Service) SendChatMessage(ctx context.Context, accountID int64, conversationID, text string) (appservice.ChatMessageView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatMessageView{}, err
	}
	return service.core.SendChatMessage(ctx, accountID, conversationID, text)
}

func (service *Service) GetAppSettings() (appservice.AppSettingsState, error) {
	return service.core.GetAppSettings()
}

func (service *Service) GetMultiInstanceState() robloxmulti.Snapshot {
	return service.core.GetMultiInstanceState()
}

func (service *Service) GetRobloxProcesses() (robloxmulti.ProcessSnapshot, error) {
	return service.core.GetRobloxProcesses()
}

func (service *Service) KillRobloxProcesses(ctx context.Context, processes []robloxmulti.Process) error {
	for _, process := range processes {
		if process.PID == 0 || process.StartTime == "" || process.Name == "" {
			return errors.New("refresh the Roblox process list and try again")
		}
	}
	return service.core.KillRobloxProcesses(ctx, processes)
}

func (service *Service) GetLaunchConfirmation() appservice.LaunchConfirmation {
	return service.core.GetLaunchConfirmation()
}

func (service *Service) ResolveLaunchConfirmation(id string, approved bool) {
	service.core.ResolveLaunchConfirmation(id, approved)
}

func (service *Service) SetMultiInstanceEnabled(enabled bool) (robloxmulti.Snapshot, error) {
	return service.core.SetMultiInstanceEnabled(enabled)
}

func (service *Service) GetRobloxClients() gamelaunch.ClientState {
	return service.core.GetRobloxClients()
}

func (service *Service) SetLinuxClient(client string) (gamelaunch.ClientState, error) {
	return service.core.SetLinuxClient(client)
}

func (service *Service) SetLoggingLevelEnabled(level string, enabled bool) error {
	return service.core.SetLoggingLevelEnabled(level, enabled)
}

func (service *Service) SetAllLoggingLevelsEnabled(enabled bool) error {
	return service.core.SetAllLoggingLevelsEnabled(enabled)
}

func (service *Service) SetRoValraEnabled(enabled bool) error {
	return service.core.SetRoValraEnabled(enabled)
}

func (service *Service) ParseTimestampFormat(source string) (timestampformat.Format, error) {
	return service.core.ParseTimestampFormat(source)
}

func (service *Service) SetTimestampFormats(timestampFormat, timestampHoverFormat string) (appservice.TimestampFormats, error) {
	return service.core.SetTimestampFormats(timestampFormat, timestampHoverFormat)
}

func (service *Service) SetMotion(motion appsettings.MotionPreference) error {
	return service.core.SetMotion(motion)
}

func (service *Service) CreateVault(ctx context.Context, password, confirmation, hint string, automaticUnlock bool) error {
	return service.core.CreateVault(ctx, password, confirmation, hint, automaticUnlock)
}

func (service *Service) UnlockVault(ctx context.Context, password string) error {
	return service.core.UnlockVault(ctx, password)
}

func (service *Service) CheckPassword(password string) vault.PasswordRequirements {
	return vault.CheckPassword(password)
}

func (service *Service) LockVault() error { return service.core.LockVault() }

func (service *Service) GetBrowserState() browser.Snapshot { return service.core.GetBrowserState() }
func (service *Service) GetBrowserShutdownEffects() browser.ShutdownEffects {
	return service.core.GetBrowserShutdownEffects()
}
func (service *Service) DownloadBrowserRuntime(ctx context.Context) error {
	return service.core.DownloadBrowserRuntime(ctx)
}
func (service *Service) CancelBrowserRuntimeDownload() { service.core.CancelBrowserRuntimeDownload() }
func (service *Service) RedownloadBrowserRuntime(ctx context.Context) error {
	return service.core.RedownloadBrowserRuntime(ctx)
}
func (service *Service) RemoveBrowserRuntime() error { return service.core.RemoveBrowserRuntime() }
func (service *Service) StartLoginBrowser(ctx context.Context) (browser.SessionState, error) {
	return service.core.StartLoginBrowser(ctx)
}
func (service *Service) OpenAccountBrowser(ctx context.Context, accountID int64) (browser.SessionState, error) {
	if err := positiveID(accountID, "browser-open"); err != nil {
		return browser.SessionState{}, err
	}
	return service.core.OpenAccountBrowser(ctx, accountID)
}
func (service *Service) FocusBrowserSession(sessionID string) error {
	return service.core.FocusBrowserSession(sessionID)
}
func (service *Service) CloseBrowserSession(sessionID string) error {
	return service.core.CloseBrowserSession(sessionID)
}
func (service *Service) CloseAllLoginBrowsers() { service.core.CloseAllLoginBrowsers() }
func (service *Service) SaveBrowserCandidates(ctx context.Context, selected []string) (browser.SaveResult, error) {
	return service.core.SaveBrowserCandidates(ctx, selected)
}
func (service *Service) DiscardBrowserCandidates() { service.core.DiscardBrowserCandidates() }
func (service *Service) RetryBrowserCleanup()      { service.core.RetryBrowserCleanup() }

func (service *Service) TestVaultPassword(ctx context.Context, password string) error {
	return service.core.TestVaultPassword(ctx, password)
}

func (service *Service) DismissPasswordReminder() error {
	return service.core.DismissPasswordReminder()
}

func (service *Service) SetPasswordTestIntervalDays(days int) error {
	return service.core.SetPasswordTestIntervalDays(days)
}

func (service *Service) ChangeVaultPassword(ctx context.Context, currentPassword, newPassword, confirmation, hint string, automaticUnlock bool) error {
	return service.core.ChangeVaultPassword(ctx, currentPassword, newPassword, confirmation, hint, automaticUnlock)
}

func (service *Service) SetAutomaticUnlock(enabled bool) error {
	return service.core.SetAutomaticUnlock(enabled)
}

func (service *Service) ListBackups() ([]vault.BackupInfo, error) { return service.core.ListBackups() }

func (service *Service) RestoreBackup(ctx context.Context, name, password string) error {
	return service.core.RestoreBackup(ctx, name, password)
}

func (service *Service) ResetVault(confirmation string) error {
	if confirmation != "RESET" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-reset", Message: "Type RESET to confirm vault deletion."}
	}
	return service.core.ResetVault()
}

func (service *Service) ValidateCookies(ctx context.Context, input string) (appservice.ImportPreview, error) {
	return service.core.ValidateCookies(ctx, input)
}

func (service *Service) SaveValidatedCookies(ctx context.Context, batchID string, selectedIndexes []int) (appservice.ImportBatchResult, error) {
	return service.core.SaveValidatedCookies(ctx, batchID, selectedIndexes)
}

func (service *Service) DiscardValidatedCookies(batchID string) {
	service.core.DiscardValidatedCookies(batchID)
}

func (service *Service) ReplaceCookie(ctx context.Context, accountID int64, input string) (accounts.AccountView, error) {
	if err := positiveID(accountID, "cookie-replacement"); err != nil {
		return accounts.AccountView{}, err
	}
	return service.core.ReplaceCookie(ctx, accountID, input)
}

func (service *Service) RenewCookie(ctx context.Context, accountID int64) (appservice.CookieRenewalResult, error) {
	if err := positiveID(accountID, "cookie-renewal"); err != nil {
		return appservice.CookieRenewalResult{}, err
	}
	return service.core.RenewCookie(ctx, accountID)
}

func (service *Service) RemoveAccount(ctx context.Context, accountID int64) error {
	if err := positiveID(accountID, "account-remove"); err != nil {
		return err
	}
	return service.core.RemoveAccount(ctx, accountID)
}

func (service *Service) ListAccounts(ctx context.Context, query accounts.AccountQuery) (accounts.AccountPage, error) {
	return service.core.ListAccounts(ctx, query)
}

func (service *Service) MoveAccount(ctx context.Context, accountID int64, beforeID, afterID *int64) error {
	if err := positiveID(accountID, "account-order"); err != nil {
		return err
	}
	return service.core.MoveAccount(ctx, accountID, beforeID, afterID)
}

func (service *Service) ListTags(ctx context.Context) ([]accounts.TagView, error) {
	return service.core.ListTags(ctx)
}

func (service *Service) CreateTag(ctx context.Context, name string) (accounts.TagView, error) {
	return service.core.CreateTag(ctx, name)
}

func (service *Service) RemoveTag(ctx context.Context, tagID int64) error {
	if err := positiveID(tagID, "account-tag"); err != nil {
		return err
	}
	return service.core.RemoveTag(ctx, tagID)
}

func (service *Service) SetAccountsTag(ctx context.Context, accountIDs []int64, tagID int64, selected bool) ([]accounts.AccountView, error) {
	if err := accountSelection(accountIDs, "account-tag"); err != nil {
		return nil, err
	}
	if err := positiveID(tagID, "account-tag"); err != nil {
		return nil, err
	}
	return service.core.SetAccountsTag(ctx, accountIDs, tagID, selected)
}

func (service *Service) GetAvatarHeadshots(ctx context.Context, userIDs []int64) ([]robloxservices.AvatarHeadshotView, error) {
	for _, userID := range userIDs {
		if err := positiveID(userID, "avatar-headshots"); err != nil {
			return nil, err
		}
	}
	return service.core.GetAvatarHeadshots(ctx, userIDs)
}

func (service *Service) GetAccountPresences(ctx context.Context, accountID *int64) ([]robloxservices.UserPresence, error) {
	if accountID != nil {
		if err := positiveID(*accountID, "user-presences"); err != nil {
			return nil, err
		}
	}
	return service.core.GetAccountPresences(ctx, accountID)
}

func (service *Service) SetPresenceUpdates(scope appsettings.PresenceScope, enabled bool, intervalSeconds int) error {
	return service.core.SetPresenceUpdates(scope, enabled, intervalSeconds)
}

func (service *Service) FindAccountByRobloxUserID(ctx context.Context, robloxUserID int64) (*accounts.AccountView, error) {
	if err := positiveID(robloxUserID, "account-lookup"); err != nil {
		return nil, err
	}
	return service.core.FindAccountByRobloxUserID(ctx, robloxUserID)
}

func (service *Service) GetRobloxUser(ctx context.Context, robloxUserID int64) (robloxservices.UserView, error) {
	if err := positiveID(robloxUserID, "user-profile"); err != nil {
		return robloxservices.UserView{}, err
	}
	return service.core.GetRobloxUser(ctx, robloxUserID)
}

func (service *Service) ValidateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	if err := positiveID(accountID, "account-validation"); err != nil {
		return accounts.AccountView{}, err
	}
	return service.core.ValidateAccount(ctx, accountID)
}

func (service *Service) LaunchAccounts(ctx context.Context, accountIDs []int64, input gamelaunch.Input) (appservice.LaunchResult, error) {
	if err := accountSelection(accountIDs, "game-launch"); err != nil {
		return appservice.LaunchResult{}, err
	}
	result, err := service.core.LaunchAccounts(ctx, accountIDs, input)
	var failure *roblox.Error
	if errors.As(err, &failure) {
		return result, errors.New(failure.Message)
	}
	return result, err
}

func (service *Service) CopyLaunchOptions(ctx context.Context, accountIDs []int64, input gamelaunch.Input) error {
	if err := accountSelection(accountIDs, "copy-launch-options"); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "copy-launch-options", Message: "The clipboard is unavailable. Restart the app and try again."}
	}
	return service.core.CopyLaunchOptions(ctx, accountIDs, input, app.Clipboard.SetText)
}

func accountSelection(accountIDs []int64, endpoint string) error {
	if len(accountIDs) == 0 {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Select at least one account."}
	}
	seen := make(map[int64]bool, len(accountIDs))
	for _, accountID := range accountIDs {
		if err := positiveID(accountID, endpoint); err != nil {
			return err
		}
		if seen[accountID] {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The account selection contains duplicates."}
		}
		seen[accountID] = true
	}
	return nil
}

func positiveID(value int64, endpoint string) error {
	if value > 0 {
		return nil
	}
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "The requested identifier is invalid."}
}
