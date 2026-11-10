package bindings

import (
	"context"
	"errors"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/appmeta"
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

	startMutex   sync.Mutex
	startContext context.Context
	started      bool
}

func NewService(core *appservice.Service, location *appservice.Location, updates *appupdate.Service, events *appservice.Events, launch *logging.Launch) *Service {
	return &Service{core: core, location: location, updates: updates, events: events, launch: launch}
}

func (service *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	service.launch.Begin("service-startup")
	if app := application.Get(); app != nil {
		service.events.AttachWindow(
			func(name string, payload any) { app.Event.Emit(name, payload) },
			func() { showMainWindow(app) },
		)
	}
	service.startMutex.Lock()
	service.startContext = ctx
	service.startMutex.Unlock()
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
	service.events.DetachWindow()
	service.updates.Shutdown()
	return service.core.Shutdown()
}

func showMainWindow(app *application.App) {
	windows := app.Window.GetAll()
	if len(windows) == 0 {
		return
	}
	window := windows[0]
	if window.IsMinimised() {
		window.UnMinimise()
	}
	window.Show()
	window.Focus()
}

func (service *Service) GetLaunchReport() logging.LaunchReport { return service.launch.Report() }

func (service *Service) GetBuild() appmeta.Build { return appmeta.CurrentBuild() }

func (service *Service) GetAppLocation() appdata.Location { return service.location.State() }

func (service *Service) ConfirmAppLocation(mode appdata.Mode) (bool, error) {
	restarting, err := service.location.Confirm(mode)
	if err != nil || restarting {
		return restarting, err
	}
	return false, service.startCore()
}

func (service *Service) MoveAppData(mode appdata.Mode) error {
	return service.location.Move(mode)
}

// Automatic unlock and migrations must never touch a vault that the user has
// not chosen.
func (service *Service) startCore() error {
	service.startMutex.Lock()
	defer service.startMutex.Unlock()
	if service.started || service.startContext == nil || !service.location.State().Initialized {
		return nil
	}
	if err := service.core.Start(service.startContext); err != nil {
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

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64, section appservice.AccountProfileSection) (appservice.AccountProfileSnapshot, error) {
	if err := positiveID(accountID, "account-profile"); err != nil {
		return appservice.AccountProfileSnapshot{}, err
	}
	return service.core.GetAccountProfile(ctx, accountID, section)
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

func (service *Service) ChangeDisplayName(ctx context.Context, accountID int64, displayName string) error {
	if err := positiveID(accountID, "display-name-update"); err != nil {
		return err
	}
	return service.core.ChangeDisplayName(ctx, accountID, displayName)
}

func (service *Service) ValidateAccountDescription(ctx context.Context, accountID int64, description string) (robloxservices.TextFilter, error) {
	if err := positiveID(accountID, "description-validate"); err != nil {
		return robloxservices.TextFilter{}, err
	}
	return service.core.ValidateAccountDescription(ctx, accountID, description)
}

func (service *Service) ChangeAccountDescription(ctx context.Context, accountID int64, description string) (string, error) {
	if err := positiveID(accountID, "description-update"); err != nil {
		return "", err
	}
	return service.core.ChangeAccountDescription(ctx, accountID, description)
}

func (service *Service) ChangeAccountEmail(ctx context.Context, accountID int64, password, email string) error {
	if err := positiveID(accountID, "email-update"); err != nil {
		return err
	}
	return service.core.ChangeAccountEmail(ctx, accountID, password, email)
}

func (service *Service) ChangeAccountPassword(ctx context.Context, accountID int64, currentPassword, newPassword string) error {
	if err := positiveID(accountID, "password-change"); err != nil {
		return err
	}
	return service.core.ChangeAccountPassword(ctx, accountID, currentPassword, newPassword)
}

func (service *Service) SaveAccountPassword(ctx context.Context, accountID int64, password string) error {
	if err := positiveID(accountID, "password-save"); err != nil {
		return err
	}
	return service.core.SaveAccountPassword(ctx, accountID, password)
}

func (service *Service) GetAccountPassword(ctx context.Context, accountID int64) (string, error) {
	if err := positiveID(accountID, "password-read"); err != nil {
		return "", err
	}
	return service.core.GetAccountPassword(ctx, accountID)
}

func (service *Service) ValidateAccountPassword(ctx context.Context, accountID int64, password string) (bool, error) {
	if err := positiveID(accountID, "password-validate"); err != nil {
		return false, err
	}
	return service.core.ValidateAccountPassword(ctx, accountID, password)
}

func (service *Service) UnlockAccountPin(ctx context.Context, accountID int64, pin string) error {
	if err := positiveID(accountID, "account-pin-unlock"); err != nil {
		return err
	}
	return service.core.UnlockAccountPin(ctx, accountID, pin)
}

func (service *Service) GetBlockedUsers(ctx context.Context, accountID int64) ([]appservice.BlockedUser, error) {
	if err := positiveID(accountID, "blocked-users"); err != nil {
		return nil, err
	}
	return service.core.GetBlockedUsers(ctx, accountID)
}

func (service *Service) UnblockUser(ctx context.Context, accountID, userID int64) error {
	if err := positiveID(accountID, "user-unblock"); err != nil {
		return err
	}
	if err := positiveID(userID, "user-unblock"); err != nil {
		return err
	}
	return service.core.UnblockUser(ctx, accountID, userID)
}

func (service *Service) GetAccountSessions(ctx context.Context, accountID int64) ([]appservice.AccountSession, error) {
	if err := positiveID(accountID, "account-sessions"); err != nil {
		return nil, err
	}
	return service.core.GetAccountSessions(ctx, accountID)
}

func (service *Service) SignOutOtherSessions(ctx context.Context, accountID int64) (appservice.CookieRenewalResult, error) {
	if err := positiveID(accountID, "sessions-sign-out"); err != nil {
		return appservice.CookieRenewalResult{}, err
	}
	return service.core.SignOutOtherSessions(ctx, accountID)
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

func (service *Service) CreateChatConversation(ctx context.Context, accountID int64, userIDs []int64, name string) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.CreateChatConversation(ctx, accountID, userIDs, name)
}

func (service *Service) AddChatMembers(ctx context.Context, accountID int64, conversationID string, userIDs []int64) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.AddChatMembers(ctx, accountID, conversationID, userIDs)
}

func (service *Service) RenameChatConversation(ctx context.Context, accountID int64, conversationID, name string) (appservice.ChatConversationView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatConversationView{}, err
	}
	return service.core.RenameChatConversation(ctx, accountID, conversationID, name)
}

func (service *Service) SendChatMessage(ctx context.Context, accountID int64, conversationID, text string) (appservice.ChatMessageView, error) {
	if err := positiveID(accountID, "chat"); err != nil {
		return appservice.ChatMessageView{}, err
	}
	return service.core.SendChatMessage(ctx, accountID, conversationID, text)
}

func (service *Service) GetRobloxNotifications(ctx context.Context, accountID int64, startIndex int) (appservice.RobloxNotificationPage, error) {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return appservice.RobloxNotificationPage{}, err
	}
	return service.core.GetRobloxNotifications(ctx, accountID, startIndex)
}

func (service *Service) DismissRobloxNotification(ctx context.Context, accountID int64, notificationID, dismissAction string) error {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return err
	}
	return service.core.DismissRobloxNotification(ctx, accountID, notificationID, dismissAction)
}

func (service *Service) DismissAllRobloxNotifications(ctx context.Context, accountID int64, category appservice.RobloxNotificationCategory) (int, error) {
	if err := positiveID(accountID, "roblox-notifications"); err != nil {
		return 0, err
	}
	return service.core.DismissAllRobloxNotifications(ctx, accountID, category)
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

func (service *Service) GetRobloxPlayers() appservice.RobloxPlayers {
	return service.core.GetRobloxPlayers()
}

func (service *Service) CheckRobloxPlayerPath(path string) appservice.RobloxPlayerPathCheck {
	return service.core.CheckRobloxPlayerPath(path)
}

func (service *Service) SetRobloxPlayerPath(path string) error {
	return service.core.SetRobloxPlayerPath(path)
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

func (service *Service) SetRBXServersEnabled(enabled bool) error {
	return service.core.SetRBXServersEnabled(enabled)
}

func (service *Service) ParseTimestampFormat(source string) (timestampformat.Format, error) {
	return timestampformat.Parse(source)
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

func (service *Service) ListBackups() ([]vault.Backup, error) { return service.core.ListBackups() }

func (service *Service) RestoreBackup(ctx context.Context, name, password string) error {
	return service.core.RestoreBackup(ctx, name, password)
}

func (service *Service) ResetVault(confirmation string) error {
	if confirmation != "RESET" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-reset", Message: "Type RESET to confirm vault deletion."}
	}
	return service.core.ResetVault()
}

func (service *Service) MigrateLegacyVault(ctx context.Context, password string) error {
	return service.core.MigrateLegacyVault(ctx, password)
}

func (service *Service) SetAsideLegacyVault(confirmation string) error {
	if confirmation != "CONFIRM" {
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-legacy", Message: "Type CONFIRM to set the old vault aside."}
	}
	return service.core.SetAsideLegacyVault()
}

func (service *Service) ExportVault(ctx context.Context, password, confirmation, hint string) (vault.ExportFile, error) {
	if err := service.core.ValidateExportPassword(password, confirmation, hint); err != nil {
		return vault.ExportFile{}, err
	}
	path, err := chooseSaveFile(service.core.ExportsDirectory(), service.core.DefaultExportName())
	if err != nil {
		return vault.ExportFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "The save dialog could not be opened.", Cause: err}
	}
	if path == "" {
		return vault.ExportFile{}, nil
	}
	return service.core.ExportVault(ctx, path, password, confirmation, hint)
}

func (service *Service) ChooseVaultExport() (string, error) {
	path, err := chooseOpenFile(service.core.ExportsDirectory())
	if err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "The open dialog could not be opened.", Cause: err}
	}
	return path, nil
}

func (service *Service) GetExportDetails(path string) (vault.ExportDetails, error) {
	return service.core.ExportDetails(path)
}

func (service *Service) OpenVaultExport(ctx context.Context, path, password string) (appservice.TransferPreview, error) {
	return service.core.OpenVaultExport(ctx, path, password)
}

func (service *Service) ImportVaultExport(ctx context.Context, id string, request appservice.TransferRequest, confirmation string) (appservice.TransferResult, error) {
	if request.Mode == appservice.TransferOverwrite && confirmation != "CONFIRM" {
		return appservice.TransferResult{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "Type CONFIRM to overwrite the vault."}
	}
	return service.core.ImportVaultExport(ctx, id, request)
}

func (service *Service) DiscardVaultExport(id string) { service.core.DiscardVaultExport(id) }

func (service *Service) ValidateCookies(ctx context.Context, input string) (appservice.ImportPreview, error) {
	return service.core.ValidateCookies(ctx, input)
}

func (service *Service) SaveValidatedCookies(ctx context.Context, batchID string, selectedIndexes []int) (appservice.ImportBatchResult, error) {
	return service.core.SaveValidatedCookies(ctx, batchID, selectedIndexes)
}

func (service *Service) DiscardValidatedCookies(batchID string) {
	service.core.DiscardValidatedCookies(batchID)
}

func (service *Service) RenewCookie(ctx context.Context, accountID int64) (appservice.CookieRenewalResult, error) {
	if err := positiveID(accountID, "cookie-renewal"); err != nil {
		return appservice.CookieRenewalResult{}, err
	}
	return service.core.RenewCookie(ctx, accountID)
}

func (service *Service) ReactivateAccount(ctx context.Context, accountID int64) (accounts.AccountView, error) {
	if err := positiveID(accountID, "account-reactivate"); err != nil {
		return accounts.AccountView{}, err
	}
	return service.core.ReactivateAccount(ctx, accountID)
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
	return service.core.LaunchAccounts(ctx, accountIDs, input)
}

func (service *Service) CopyLaunchOptions(ctx context.Context, accountIDs []int64, input gamelaunch.Input) error {
	if err := accountSelection(accountIDs, "copy-launch-options"); err != nil {
		return err
	}
	app := application.Get()
	if app == nil {
		return clipboardUnavailable()
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
