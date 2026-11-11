package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

var (
	errLaunchesBlocked        = errors.New("browser launches are unavailable while the browser is stopping")
	errBrowserVersionMismatch = errors.New("browser product version does not match the runtime manifest")
)

type Mode string

const (
	ModeLogin Mode = "login"
	ModeSaved Mode = "saved"
)

type Lifecycle string

const (
	LifecycleStarting   Lifecycle = "starting"
	LifecycleActive     Lifecycle = "active"
	LifecycleClosing    Lifecycle = "closing"
	LifecycleValidating Lifecycle = "validating"
	LifecycleClosed     Lifecycle = "closed"
	LifecycleFailed     Lifecycle = "failed"
)

type SessionState struct {
	ID                string    `json:"id"`
	Mode              Mode      `json:"mode"`
	Lifecycle         Lifecycle `json:"lifecycle"`
	AccountID         int64     `json:"accountId,omitempty"`
	Title             string    `json:"title"`
	Attached          bool      `json:"attached"`
	CheckpointWarning string    `json:"checkpointWarning,omitempty"`
	Error             string    `json:"error,omitempty"`
	ActiveDownloads   int       `json:"activeDownloads"`
}

type CandidateState struct {
	ID              string `json:"id"`
	SessionID       string `json:"sessionId"`
	SameAccountAsID string `json:"sameAccountAsId,omitempty"`
	UpdatesExisting bool   `json:"updatesExisting,omitempty"`
	RobloxUserID    int64  `json:"robloxUserId"`
	Username        string `json:"username"`
	DisplayName     string `json:"displayName"`
	AvatarURL       string `json:"avatarUrl,omitempty"`
}

type CleanupState struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}
type Snapshot struct {
	Revision   uint64           `json:"revision"`
	Runtime    RuntimeState     `json:"runtime"`
	Sessions   []SessionState   `json:"sessions"`
	Candidates []CandidateState `json:"candidates"`
	Cleanup    []CleanupState   `json:"cleanup"`
}
type ShutdownEffects struct {
	BrowserSessions   int  `json:"browserSessions"`
	BrowserDownloads  bool `json:"browserDownloads"`
	AccountCandidates int  `json:"accountCandidates"`
	AccountChecks     bool `json:"accountChecks"`
}

type SaveStatus string

const (
	SaveCreated   SaveStatus = "created"
	SaveUpdated   SaveStatus = "updated"
	SaveDuplicate SaveStatus = "duplicate"
	SaveFailed    SaveStatus = "failed"
)

type SaveResultItem struct {
	CandidateID string               `json:"candidateId"`
	Status      SaveStatus           `json:"status"`
	Account     accounts.AccountView `json:"account"`
	Error       string               `json:"error,omitempty"`
}
type SaveResult struct {
	Items []SaveResultItem `json:"items"`
}

type AccountSource interface {
	BrowserAccount(context.Context, int64) (accounts.AccountView, roblox.SessionSnapshot, error)
	BrowserCandidateAccount(context.Context, int64) (accounts.AccountView, bool)
	ValidateBrowserCookie(context.Context, string, string) (roblox.CookieValidation, error)
	CommitBrowserCandidate(context.Context, roblox.CookieValidation) (accounts.AccountView, SaveStatus, error)
	SynchronizeBrowserCookie(context.Context, int64, int64, roblox.CookieValidation) (int64, error)
	BrowserAvatar(context.Context, int64) string
	LaunchBrowserGame(context.Context, string, string)
}

type session struct {
	state               SessionState
	process             BrowserProcess
	cdp                 *CDPClient
	directory           string
	targetID            string
	targetSessions      map[string]string
	pages               map[string]struct{}
	secretVersion       int64
	robloxUserID        int64
	browserID           string
	lastCookie          string
	generation          uint64
	validatedGeneration uint64
	lastCheckpointError string
	gameLaunchPending   bool
	checkpointGate      chan struct{}
	downloads           map[string]struct{}
	trigger             chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
	closeOnce           sync.Once
}

type candidate struct {
	state      CandidateState
	validation roblox.CookieValidation
}

type Coordinator struct {
	paths   appdata.Paths
	runtime *RuntimeManager
	backend AccountSource
	logger  *slog.Logger
	changed func(uint64)
	cleanup *cleanupQueue

	mutex           sync.Mutex
	lifecycleMutex  sync.Mutex
	savedStartMutex sync.Mutex
	starts          sync.WaitGroup
	pendingStarts   int
	revision        uint64
	sessions        map[string]*session
	candidates      map[string]*candidate
	launchesBlocked bool
	stopped         bool
	validationCount int
	root            context.Context
	cancel          context.CancelFunc
}

func NewCoordinator(paths appdata.Paths, runtimeManager *RuntimeManager, backend AccountSource, logger *slog.Logger, changed func(uint64)) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	coordinator := &Coordinator{paths: paths, runtime: runtimeManager, backend: backend, logger: logger, changed: changed, sessions: make(map[string]*session), candidates: make(map[string]*candidate), root: ctx, cancel: cancel}
	coordinator.cleanup = newCleanupQueue(paths.BrowserTempRoot, logger, coordinator.bump)
	runtimeManager.SetChanged(coordinator.bump)
	coordinator.queueStaleSessions()
	coordinator.cleanup.retry()
	return coordinator
}

func (coordinator *Coordinator) Snapshot() Snapshot {
	coordinator.mutex.Lock()
	defer coordinator.mutex.Unlock()
	snapshot := Snapshot{Revision: coordinator.revision, Runtime: coordinator.runtime.State()}
	for _, current := range coordinator.sessions {
		snapshot.Sessions = append(snapshot.Sessions, current.state)
	}
	for _, current := range coordinator.candidates {
		snapshot.Candidates = append(snapshot.Candidates, current.state)
	}
	snapshot.Cleanup = coordinator.cleanup.snapshot()
	sort.Slice(snapshot.Sessions, func(left, right int) bool { return snapshot.Sessions[left].ID < snapshot.Sessions[right].ID })
	sort.Slice(snapshot.Candidates, func(left, right int) bool { return snapshot.Candidates[left].ID < snapshot.Candidates[right].ID })
	return snapshot
}

func (coordinator *Coordinator) Effects() ShutdownEffects {
	coordinator.mutex.Lock()
	defer coordinator.mutex.Unlock()
	effects := ShutdownEffects{BrowserSessions: coordinator.pendingStarts, AccountCandidates: len(coordinator.candidates), AccountChecks: coordinator.validationCount > 0}
	for _, current := range coordinator.sessions {
		if current.state.Lifecycle != LifecycleClosed && current.state.Lifecycle != LifecycleFailed {
			effects.BrowserSessions++
		}
		if current.state.Lifecycle != LifecycleFailed && len(current.downloads) > 0 {
			effects.BrowserDownloads = true
		}
	}
	runtimeState := coordinator.runtime.State()
	effects.BrowserDownloads = effects.BrowserDownloads || runtimeState.Status == RuntimeDownloading || runtimeState.Status == RuntimeInstalling
	return effects
}

func (coordinator *Coordinator) StartLogin(ctx context.Context) (SessionState, error) {
	return coordinator.start(ctx, ModeLogin, 0)
}
func (coordinator *Coordinator) OpenAccount(ctx context.Context, accountID int64) (SessionState, error) {
	coordinator.savedStartMutex.Lock()
	defer coordinator.savedStartMutex.Unlock()
	coordinator.mutex.Lock()
	for _, current := range coordinator.sessions {
		if current.state.Mode == ModeSaved && current.state.AccountID == accountID && current.state.Lifecycle != LifecycleClosed && current.state.Lifecycle != LifecycleFailed {
			state := current.state
			coordinator.mutex.Unlock()
			if err := coordinator.Focus(state.ID); err != nil {
				return state, err
			}
			return state, nil
		}
	}
	coordinator.mutex.Unlock()
	return coordinator.start(ctx, ModeSaved, accountID)
}

func (coordinator *Coordinator) start(ctx context.Context, mode Mode, accountID int64) (SessionState, error) {
	coordinator.mutex.Lock()
	if coordinator.launchesBlocked || coordinator.stopped {
		coordinator.mutex.Unlock()
		return SessionState{}, errLaunchesBlocked
	}
	lifetime := coordinator.root
	coordinator.starts.Add(1)
	coordinator.pendingStarts++
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	registered := false
	defer func() {
		coordinator.finishStart(registered)
	}()
	ctx, cancelStart := context.WithCancel(ctx)
	stop := context.AfterFunc(lifetime, cancelStart)
	defer stop()
	defer cancelStart()
	if lifetime.Err() != nil {
		return SessionState{}, errLaunchesBlocked
	}
	executable, err := coordinator.runtime.Executable()
	if err != nil {
		return SessionState{}, err
	}
	browserSessionID, err := randomID(24)
	if err != nil {
		return SessionState{}, err
	}
	directory, err := coordinator.cleanup.createDirectory(browserSessionID)
	if err != nil {
		coordinator.cleanup.retry()
		return SessionState{}, err
	}
	state := SessionState{ID: browserSessionID, Mode: mode, Lifecycle: LifecycleStarting, AccountID: accountID, Attached: mode == ModeSaved, Title: "Roblox login"}
	var account accounts.AccountView
	record := roblox.SessionSnapshot{BrowserID: roblox.NewBrowserID()}
	if mode == ModeSaved {
		account, record, err = coordinator.backend.BrowserAccount(ctx, accountID)
		if err != nil {
			coordinator.cleanup.enqueue(browserSessionID)
			return SessionState{}, err
		}
		state.Title = "Roblox - @" + account.Username
	}
	if err := ctx.Err(); err != nil {
		coordinator.cleanup.enqueue(browserSessionID)
		return SessionState{}, err
	}
	process, err := startBrowserProcess(ProcessOptions{Executable: executable, UserDataPath: directory})
	if err != nil {
		coordinator.cleanup.enqueue(browserSessionID)
		return SessionState{}, err
	}
	currentContext, cancel := context.WithCancel(lifetime)
	checkpointGate := make(chan struct{}, 1)
	checkpointGate <- struct{}{}
	current := &session{state: state, process: process, directory: directory, secretVersion: record.SecretVersion, robloxUserID: account.RobloxUserID, browserID: record.BrowserID, trigger: make(chan struct{}, 1), checkpointGate: checkpointGate, ctx: currentContext, cancel: cancel, downloads: make(map[string]struct{}), targetSessions: make(map[string]string), pages: make(map[string]struct{})}
	input, output := process.CDPPipes()
	current.cdp = NewCDPClient(input, output, func(failure error) { coordinator.failSession(browserSessionID, failure) })
	coordinator.mutex.Lock()
	if coordinator.launchesBlocked || currentContext.Err() != nil || ctx.Err() != nil {
		coordinator.mutex.Unlock()
		coordinator.finishSession(current, false, "")
		return SessionState{}, errLaunchesBlocked
	}
	coordinator.sessions[browserSessionID] = current
	registered = true
	coordinator.pendingStarts--
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	coordinator.logger.InfoContext(ctx, "managed browser session starting", "operation", "browser-session-start", "session_id", browserSessionID, "session_mode", mode, "account_id", accountID)
	if mode == ModeLogin {
		go coordinator.initializeLogin(currentContext, current)
		return state, nil
	}
	return coordinator.initializeSavedSession(ctx, current, state, record.Cookie)
}

func (coordinator *Coordinator) initializeLogin(ctx context.Context, current *session) {
	startupContext, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := coordinator.initializeCDP(startupContext, current); err != nil {
		if ctx.Err() != nil {
			return
		}
		coordinator.abortStartup(current, err)
		return
	}
	sessionID, err := coordinator.targetSession(startupContext, current)
	if err == nil {
		err = current.cdp.SetBrowserID(startupContext, sessionID, current.browserID)
	}
	if err == nil {
		err = current.cdp.Navigate(startupContext, sessionID, "https://www.roblox.com/Login")
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		coordinator.abortStartup(current, err)
		return
	}
	state := coordinator.activateSession(current)
	if state.Lifecycle != LifecycleActive {
		return
	}
	go coordinator.observe(ctx, current)
	go coordinator.watchProcess(current)
	_ = coordinator.Focus(current.state.ID)
	coordinator.logger.InfoContext(ctx, "managed browser session started", "operation", "browser-session-start", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
}

func (coordinator *Coordinator) initializeCDP(ctx context.Context, current *session) error {
	if err := current.cdp.VerifyVersion(ctx, coordinator.runtime.Manifest().Version); err != nil {
		if errors.Is(err, errBrowserVersionMismatch) {
			coordinator.runtime.MarkDamaged()
		}
		return fmt.Errorf("verify managed browser: %w", err)
	}
	if err := current.cdp.InterceptRobloxLaunches(ctx, func(protocolURL string) {
		coordinator.mutex.Lock()
		allowed := current.ctx.Err() == nil && !coordinator.launchesBlocked && !current.gameLaunchPending && coordinator.sessions[current.state.ID] == current && current.state.Lifecycle == LifecycleActive
		if allowed {
			current.gameLaunchPending = true
		}
		coordinator.mutex.Unlock()
		if allowed {
			go func() {
				defer func() {
					coordinator.mutex.Lock()
					current.gameLaunchPending = false
					coordinator.mutex.Unlock()
				}()
				launchContext, cancel := context.WithTimeout(current.ctx, 90*time.Second)
				defer cancel()
				coordinator.backend.LaunchBrowserGame(launchContext, current.state.ID, protocolURL)
			}()
		}
	}); err != nil {
		return fmt.Errorf("prepare browser game launches: %w", err)
	}
	if err := current.cdp.DiscoverTargets(ctx); err != nil {
		return fmt.Errorf("discover browser targets: %w", err)
	}
	if err := current.cdp.EnableDownloadEvents(ctx); err != nil {
		coordinator.logger.DebugContext(ctx, "browser download events unavailable", "operation", "browser-session-start", "session_id", current.state.ID, "session_mode", current.state.Mode, "error", err)
	}
	return nil
}

func (coordinator *Coordinator) activateSession(current *session) SessionState {
	coordinator.mutex.Lock()
	defer coordinator.mutex.Unlock()
	if stored := coordinator.sessions[current.state.ID]; stored == current && current.state.Lifecycle == LifecycleStarting {
		current.state.Lifecycle = LifecycleActive
		coordinator.bumpLocked()
	}
	return current.state
}

func (coordinator *Coordinator) abortStartup(current *session, cause error) {
	coordinator.mutex.Lock()
	if stored := coordinator.sessions[current.state.ID]; stored != current || current.state.Lifecycle != LifecycleStarting {
		coordinator.mutex.Unlock()
		return
	}
	current.state.Attached = false
	current.state.Lifecycle = LifecycleFailed
	current.state.Error = "The browser session could not be started."
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()

	current.cancel()
	current.closeOnce.Do(func() {
		cause = errors.Join(cause, current.process.ExitError())
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser startup teardown failed", "operation", "browser-session-start", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(2 * time.Second) {
			coordinator.logger.Warn("browser startup process did not exit before deadline", "operation", "browser-session-start", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
		_ = current.cdp.Close()
		_ = current.process.Close()
	})
	coordinator.cleanup.enqueue(current.state.ID)
	coordinator.logger.Error("browser startup failed", "operation", "browser-session-start", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
}

func (coordinator *Coordinator) observe(ctx context.Context, current *session) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	events := current.cdp.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			coordinator.checkpoint(ctx, current, false)
		case event := <-events:
			coordinator.handleSessionEvent(current, event)
			if event.Method == "Page.loadEventFired" || event.Method == "Target.targetCreated" || event.Method == "Target.targetInfoChanged" {
				select {
				case current.trigger <- struct{}{}:
				default:
				}
			}
		case <-current.trigger:
			coordinator.checkpoint(ctx, current, false)
		}
	}
}

func (coordinator *Coordinator) handleSessionEvent(current *session, event cdpMessage) {
	switch event.Method {
	case "Target.targetCreated", "Target.targetInfoChanged":
		var params struct {
			Target CDPTarget `json:"targetInfo"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.Target.Type == "page" {
			coordinator.mutex.Lock()
			current.targetID = params.Target.TargetID
			current.pages[params.Target.TargetID] = struct{}{}
			coordinator.mutex.Unlock()
		}
	case "Target.targetDestroyed":
		coordinator.handleTargetDestroyed(current, event.Params)
	case "Target.detachedFromTarget":
		coordinator.handleTargetDetached(current, event.Params)
	case "Browser.downloadWillBegin":
		var params struct {
			GUID string `json:"guid"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.GUID != "" {
			coordinator.setDownload(current, params.GUID, true)
		}
	case "Browser.downloadProgress":
		var params struct {
			GUID  string `json:"guid"`
			State string `json:"state"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.GUID != "" && params.State != "inProgress" {
			coordinator.setDownload(current, params.GUID, false)
		}
	}
}

func (coordinator *Coordinator) setDownload(current *session, guid string, active bool) {
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] != current || current.state.Lifecycle != LifecycleActive {
		coordinator.mutex.Unlock()
		return
	}
	if active {
		current.downloads[guid] = struct{}{}
	} else {
		delete(current.downloads, guid)
	}
	current.state.ActiveDownloads = len(current.downloads)
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}

func (coordinator *Coordinator) checkpoint(ctx context.Context, current *session, final bool) {
	if _, bounded := ctx.Deadline(); !bounded {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	select {
	case <-ctx.Done():
		if final {
			coordinator.setCheckpointWarning(current, "The final browser cookie could not be checked.")
		}
		return
	case <-current.checkpointGate:
		defer func() { current.checkpointGate <- struct{}{} }()
	}
	coordinator.mutex.Lock()
	if stored := coordinator.sessions[current.state.ID]; stored != current || (!final && current.state.Lifecycle != LifecycleActive) || !current.state.Attached && current.state.Mode == ModeSaved {
		coordinator.mutex.Unlock()
		return
	}
	coordinator.mutex.Unlock()
	sessionID, cookies, err := coordinator.sessionCookies(ctx, current)
	if err != nil {
		coordinator.recordCheckpointFailure(current, err)
		if final {
			coordinator.setCheckpointWarning(current, "The final browser cookie could not be checked.")
		}
		return
	}
	coordinator.clearCheckpointFailure(current)
	if len(cookies) == 0 {
		return
	}
	selectedCookie := cookies[len(cookies)-1]
	for _, cookie := range cookies {
		if strings.TrimPrefix(strings.ToLower(cookie.Domain), ".") == "roblox.com" {
			selectedCookie = cookie
			break
		}
	}
	cookieValue := selectedCookie.Value
	coordinator.checkpointCookie(ctx, current, final, sessionID, cookieValue)
}

func (coordinator *Coordinator) recordCheckpointFailure(current *session, cause error) {
	message := cause.Error()
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] != current || current.lastCheckpointError == message {
		coordinator.mutex.Unlock()
		return
	}
	current.lastCheckpointError = message
	coordinator.mutex.Unlock()
	coordinator.logger.Warn("browser cookie checkpoint unavailable", "operation", "browser-cookie-checkpoint", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
}

func (coordinator *Coordinator) clearCheckpointFailure(current *session) {
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] != current {
		coordinator.mutex.Unlock()
		return
	}
	recovered := current.lastCheckpointError != ""
	current.lastCheckpointError = ""
	coordinator.mutex.Unlock()
	if recovered {
		coordinator.logger.Info("browser cookie checkpoint recovered", "operation", "browser-cookie-checkpoint", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	}
}

func (coordinator *Coordinator) installCandidate(ctx context.Context, current *session, validation roblox.CookieValidation) {
	coordinator.mutex.Lock()
	allowed := coordinator.sessions[current.state.ID] == current && (current.state.Lifecycle == LifecycleActive || current.state.Lifecycle == LifecycleValidating)
	coordinator.mutex.Unlock()
	if !allowed {
		return
	}
	candidateID, err := randomID(24)
	if err != nil {
		coordinator.logger.ErrorContext(ctx, "browser candidate identifier generation failed", "operation", "browser-account-capture", "session_id", current.state.ID, "session_mode", current.state.Mode, "error", err)
		return
	}
	avatar := coordinator.backend.BrowserAvatar(ctx, validation.Identity.RobloxUserID)
	existingAccount, updatesExisting := coordinator.backend.BrowserCandidateAccount(ctx, validation.Identity.RobloxUserID)
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] != current || current.state.Lifecycle != LifecycleActive && current.state.Lifecycle != LifecycleValidating {
		coordinator.mutex.Unlock()
		return
	}
	for _, existing := range coordinator.candidates {
		if existing.state.SessionID != current.state.ID || existing.state.RobloxUserID != validation.Identity.RobloxUserID {
			continue
		}
		existing.validation = validation
		existing.state.Username = validation.Identity.Username
		existing.state.DisplayName = validation.Identity.DisplayName
		existing.state.AvatarURL = avatar
		existing.state.UpdatesExisting = updatesExisting && existingAccount.ID > 0
		updatesSavedAccount := existing.state.UpdatesExisting
		coordinator.bumpLocked()
		coordinator.mutex.Unlock()
		coordinator.logger.InfoContext(ctx, "browser account candidate updated", "operation", "browser-account-capture", "session_id", current.state.ID, "session_mode", current.state.Mode, "roblox_user_id", validation.Identity.RobloxUserID, "updates_existing", updatesSavedAccount)
		return
	}
	state := CandidateState{ID: candidateID, SessionID: current.state.ID, RobloxUserID: validation.Identity.RobloxUserID, Username: validation.Identity.Username, DisplayName: validation.Identity.DisplayName, AvatarURL: avatar, UpdatesExisting: updatesExisting && existingAccount.ID > 0}
	for _, existing := range coordinator.candidates {
		if existing.state.RobloxUserID == state.RobloxUserID {
			state.SameAccountAsID = existing.state.ID
			break
		}
	}
	coordinator.candidates[candidateID] = &candidate{state: state, validation: validation}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	coordinator.logger.InfoContext(ctx, "browser account candidate captured", "operation", "browser-account-capture", "session_id", current.state.ID, "session_mode", current.state.Mode, "roblox_user_id", validation.Identity.RobloxUserID, "updates_existing", state.UpdatesExisting)
}

func (coordinator *Coordinator) Focus(browserSessionID string) error {
	coordinator.mutex.Lock()
	current := coordinator.sessions[browserSessionID]
	if current == nil {
		coordinator.mutex.Unlock()
		return errors.New("browser session not found")
	}
	target := current.targetID
	coordinator.mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if target == "" {
		if targets, err := current.cdp.Targets(ctx); err == nil {
			target = lastPageTarget(targets)
		}
	}
	if target != "" {
		if err := current.cdp.Activate(ctx, target); err != nil {
			coordinator.logger.Debug("browser target activation failed", "operation", "browser-focus", "session_id", browserSessionID, "session_mode", current.state.Mode, "error", err)
		}
	}
	if err := current.process.Focus(); err != nil {
		coordinator.logger.Warn("browser window focus failed", "operation", "browser-focus", "session_id", browserSessionID, "session_mode", current.state.Mode, "error", err)
		return err
	}
	return nil
}

func lastPageTarget(targets []CDPTarget) string {
	for _, target := range slices.Backward(targets) {
		if target.Type == "page" {
			return target.TargetID
		}
	}
	return ""
}

func (coordinator *Coordinator) targetSession(ctx context.Context, current *session) (string, error) {
	coordinator.mutex.Lock()
	targetID := current.targetID
	coordinator.mutex.Unlock()
	for targetID == "" {
		targets, err := current.cdp.Targets(ctx)
		if err != nil {
			return "", err
		}
		if targetID = lastPageTarget(targets); targetID != "" {
			break
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	return coordinator.targetSessionFor(ctx, current, targetID)
}

func (coordinator *Coordinator) targetSessionFor(ctx context.Context, current *session, targetID string) (string, error) {
	coordinator.mutex.Lock()
	if sessionID := current.targetSessions[targetID]; sessionID != "" {
		coordinator.mutex.Unlock()
		return sessionID, nil
	}
	coordinator.mutex.Unlock()
	sessionID, err := current.cdp.Attach(ctx, targetID)
	if err != nil {
		return "", err
	}
	if err := current.cdp.EnableNetwork(ctx, sessionID); err != nil {
		return "", fmt.Errorf("enable browser cookie access: %w", err)
	}
	if err := current.cdp.EnablePage(ctx, sessionID); err != nil {
		return "", fmt.Errorf("enable browser page events: %w", err)
	}
	coordinator.mutex.Lock()
	current.targetSessions[targetID] = sessionID
	current.targetID = targetID
	coordinator.mutex.Unlock()
	return sessionID, nil
}

func (coordinator *Coordinator) sessionCookies(ctx context.Context, current *session) (string, []Cookie, error) {
	if current.state.Mode == ModeSaved {
		sessionID, err := coordinator.targetSession(ctx, current)
		if err != nil {
			return "", nil, err
		}
		cookies, err := current.cdp.Cookies(ctx, sessionID)
		return sessionID, cookies, err
	}
	targets, err := current.cdp.Targets(ctx)
	if err != nil {
		return "", nil, err
	}
	var firstSession string
	var lastError error
	for _, target := range slices.Backward(targets) {
		if target.Type != "page" {
			continue
		}
		sessionID, attachError := coordinator.targetSessionFor(ctx, current, target.TargetID)
		if attachError != nil {
			lastError = attachError
			continue
		}
		if firstSession == "" {
			firstSession = sessionID
		}
		cookies, cookieError := current.cdp.Cookies(ctx, sessionID)
		if cookieError != nil {
			lastError = cookieError
			continue
		}
		if len(cookies) > 0 {
			return sessionID, cookies, nil
		}
	}
	if firstSession != "" {
		return firstSession, nil, nil
	}
	if lastError != nil {
		return "", nil, lastError
	}
	return "", nil, errors.New("browser page target not found")
}

func (coordinator *Coordinator) Close(browserSessionID string) error {
	coordinator.mutex.Lock()
	current := coordinator.sessions[browserSessionID]
	if current == nil {
		coordinator.mutex.Unlock()
		return nil
	}
	if current.state.Lifecycle == LifecycleFailed {
		delete(coordinator.sessions, browserSessionID)
		coordinator.bumpLocked()
		coordinator.mutex.Unlock()
		coordinator.logger.Info("failed browser session dismissed", "operation", "browser-session-close", "session_id", browserSessionID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		return nil
	}
	if current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating || current.state.Lifecycle == LifecycleClosed {
		coordinator.mutex.Unlock()
		return nil
	}
	current.state.Lifecycle = LifecycleClosing
	current.cancel()
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	coordinator.logger.Info("managed browser session closing", "operation", "browser-session-close", "session_id", browserSessionID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	go coordinator.closeGracefully(current)
	return nil
}

func (coordinator *Coordinator) closeGracefully(current *session) {
	coordinator.mutex.Lock()
	if current.state.Mode == ModeLogin {
		current.state.Lifecycle = LifecycleValidating
		coordinator.bumpLocked()
	}
	coordinator.mutex.Unlock()
	checkpointContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	coordinator.checkpoint(checkpointContext, current, true)
	cancel()
	closeContext, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := current.cdp.CloseBrowser(closeContext); err != nil && !errors.Is(err, context.Canceled) {
		coordinator.logger.Debug("graceful browser close unavailable", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
	}
	closeCancel()
	if !current.process.Wait(2 * time.Second) {
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser process termination failed", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(2 * time.Second) {
			coordinator.logger.Warn("browser process did not exit before deadline", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
	}
	coordinator.finishSession(current, false, "")
}

func (coordinator *Coordinator) CloseAllLogin() {
	coordinator.mutex.Lock()
	browserSessionIDs := []string{}
	for browserSessionID, current := range coordinator.sessions {
		if current.state.Mode == ModeLogin && current.state.Lifecycle != LifecycleClosed {
			browserSessionIDs = append(browserSessionIDs, browserSessionID)
		}
	}
	coordinator.mutex.Unlock()
	for _, browserSessionID := range browserSessionIDs {
		_ = coordinator.Close(browserSessionID)
	}
}

func (coordinator *Coordinator) watchProcess(current *session) {
	windowsClosed := current.process.AllWindowsClosed()
	exited := current.process.Exited()
	for windowsClosed != nil || exited != nil {
		select {
		case _, ok := <-windowsClosed:
			if !ok {
				windowsClosed = nil
				continue
			}
			coordinator.logger.Info("all managed browser windows closed", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
			_ = coordinator.Close(current.state.ID)
			return
		case err, ok := <-exited:
			if !ok {
				exited = nil
				continue
			}
			if windowsClosed != nil {
				select {
				case _, closedByUser := <-windowsClosed:
					if closedByUser {
						coordinator.logger.Info("all managed browser windows closed", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
						_ = coordinator.Close(current.state.ID)
						return
					}
				default:
				}
			}
			coordinator.mutex.Lock()
			expected := current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating
			coordinator.mutex.Unlock()
			if !expected {
				coordinator.failSession(current.state.ID, err)
			}
			return
		}
	}
}

func (coordinator *Coordinator) failSession(browserSessionID string, cause error) {
	coordinator.mutex.Lock()
	current := coordinator.sessions[browserSessionID]
	if current == nil {
		coordinator.mutex.Unlock()
		return
	}
	if current.state.Lifecycle == LifecycleStarting {
		coordinator.mutex.Unlock()
		coordinator.abortStartup(current, cause)
		return
	}
	if current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating || current.state.Lifecycle == LifecycleClosed || current.state.Lifecycle == LifecycleFailed {
		coordinator.mutex.Unlock()
		return
	}
	coordinator.mutex.Unlock()
	if !coordinator.finishSession(current, true, "The browser closed unexpectedly. The final cookie could not be checked.") {
		return
	}
	if cause == nil {
		coordinator.logger.Warn("browser session exited unexpectedly", "operation", "browser-session-failure", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	} else {
		coordinator.logger.Warn("browser session failed", "operation", "browser-session-failure", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
	}
}

func (coordinator *Coordinator) finishSession(current *session, failed bool, warning string) bool {
	finished := false
	current.closeOnce.Do(func() {
		finished = true
		current.cancel()
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser process containment termination failed", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(time.Second) {
			coordinator.logger.Warn("browser process containment did not stop before cleanup", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
		_ = current.cdp.Close()
		_ = current.process.Close()
		coordinator.mutex.Lock()
		clear(current.downloads)
		current.state.ActiveDownloads = 0
		if current.state.Mode == ModeSaved {
			current.state.Attached = false
		}
		if failed {
			current.state.Lifecycle = LifecycleFailed
			current.state.Error = warning
		} else {
			current.state.Lifecycle = LifecycleClosed
			delete(coordinator.sessions, current.state.ID)
		}
		coordinator.bumpLocked()
		coordinator.mutex.Unlock()
		coordinator.cleanup.enqueue(current.state.ID)
		if !failed {
			coordinator.logger.Info("managed browser session closed", "operation", "browser-session-close", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
	})
	return finished
}

func (coordinator *Coordinator) setCheckpointWarning(current *session, message string) {
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] != current {
		coordinator.mutex.Unlock()
		return
	}
	current.state.CheckpointWarning = message
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}
func (coordinator *Coordinator) detach(current *session, message string) {
	coordinator.mutex.Lock()
	if coordinator.sessions[current.state.ID] == current && current.state.Attached {
		current.state.Attached = false
		current.state.Error = message
		coordinator.bumpLocked()
	}
	coordinator.mutex.Unlock()
}

func (coordinator *Coordinator) DetachAccount(accountID int64, message string) {
	coordinator.mutex.Lock()
	for _, current := range coordinator.sessions {
		if current.state.AccountID == accountID && current.state.Attached {
			current.state.Attached = false
			current.state.Error = message
		}
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}

func (coordinator *Coordinator) SaveCandidates(ctx context.Context, selected []string) (SaveResult, error) {
	coordinator.mutex.Lock()
	if len(selected) == 0 {
		coordinator.mutex.Unlock()
		return SaveResult{}, errors.New("select at least one account candidate")
	}
	seen := map[int64]struct{}{}
	chosen := make([]*candidate, 0, len(selected))
	chosenIDs := map[string]struct{}{}
	for _, candidateID := range selected {
		candidate := coordinator.candidates[candidateID]
		if candidate == nil {
			coordinator.mutex.Unlock()
			return SaveResult{}, errors.New("an account candidate is no longer available")
		}
		if _, duplicate := seen[candidate.state.RobloxUserID]; duplicate {
			coordinator.mutex.Unlock()
			return SaveResult{}, errors.New("select only one candidate for each account")
		}
		seen[candidate.state.RobloxUserID] = struct{}{}
		chosenIDs[candidateID] = struct{}{}
		chosen = append(chosen, candidate)
	}
	for candidateID, candidate := range coordinator.candidates {
		if _, keep := chosenIDs[candidateID]; !keep {
			candidate.validation.Cookie = ""
		}
		delete(coordinator.candidates, candidateID)
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	saveResult := SaveResult{Items: make([]SaveResultItem, 0, len(chosen))}
	for _, chosenCandidate := range chosen {
		account, status, err := coordinator.backend.CommitBrowserCandidate(ctx, chosenCandidate.validation)
		chosenCandidate.validation.Cookie = ""
		resultItem := SaveResultItem{CandidateID: chosenCandidate.state.ID, Status: status, Account: account}
		if err != nil {
			resultItem.Status, resultItem.Error = SaveFailed, "The account could not be saved."
		}
		saveResult.Items = append(saveResult.Items, resultItem)
	}
	return saveResult, nil
}

func (coordinator *Coordinator) DiscardCandidates() {
	coordinator.mutex.Lock()
	for candidateID, candidate := range coordinator.candidates {
		candidate.validation.Cookie = ""
		delete(coordinator.candidates, candidateID)
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}

func (coordinator *Coordinator) ForceShutdown() {
	coordinator.lifecycleMutex.Lock()
	defer coordinator.lifecycleMutex.Unlock()
	if err := coordinator.forceShutdown(); err != nil {
		coordinator.logger.Warn("browser runtime did not stop", "operation", "browser-shutdown", "error", err)
	}
}

func (coordinator *Coordinator) forceShutdown() error {
	coordinator.mutex.Lock()
	coordinator.launchesBlocked = true
	coordinator.cancel()
	sessions := make([]*session, 0, len(coordinator.sessions))
	for _, current := range coordinator.sessions {
		if current.state.Lifecycle != LifecycleClosed && current.state.Lifecycle != LifecycleFailed {
			current.state.Lifecycle = LifecycleClosing
		}
		sessions = append(sessions, current)
	}
	for id, candidate := range coordinator.candidates {
		candidate.validation.Cookie = ""
		delete(coordinator.candidates, id)
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	for _, current := range sessions {
		current.cancel()
		coordinator.finishSession(current, false, "")
	}
	coordinator.starts.Wait()
	err := coordinator.runtime.StopDownload(30 * time.Second)
	coordinator.mutex.Lock()
	for id := range coordinator.sessions {
		delete(coordinator.sessions, id)
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	return err
}

func (coordinator *Coordinator) AllowLaunches() {
	coordinator.lifecycleMutex.Lock()
	defer coordinator.lifecycleMutex.Unlock()
	coordinator.allowLaunches()
}

func (coordinator *Coordinator) allowLaunches() {
	coordinator.mutex.Lock()
	defer coordinator.mutex.Unlock()
	if coordinator.stopped || !coordinator.launchesBlocked {
		return
	}
	coordinator.root, coordinator.cancel = context.WithCancel(context.Background())
	coordinator.launchesBlocked = false
	coordinator.bumpLocked()
}

func (coordinator *Coordinator) DownloadRuntime(ctx context.Context, replace bool) error {
	coordinator.lifecycleMutex.Lock()
	defer coordinator.lifecycleMutex.Unlock()
	coordinator.mutex.Lock()
	blocked, stopped := coordinator.launchesBlocked, coordinator.stopped
	coordinator.mutex.Unlock()
	if stopped {
		return errLaunchesBlocked
	}
	if !replace {
		state := coordinator.runtime.State()
		if state.Status == RuntimeDownloading || state.Status == RuntimeInstalling {
			return errors.New("a browser runtime download is already active")
		}
		if _, err := coordinator.runtime.Executable(); err == nil {
			return nil
		}
	}
	if !blocked {
		defer coordinator.allowLaunches()
	}
	if err := coordinator.forceShutdown(); err != nil {
		return fmt.Errorf("stop browser before runtime installation: %w", err)
	}
	return coordinator.runtime.Download(ctx, replace)
}

func (coordinator *Coordinator) RemoveRuntime() error {
	coordinator.lifecycleMutex.Lock()
	defer coordinator.lifecycleMutex.Unlock()
	coordinator.mutex.Lock()
	blocked, stopped := coordinator.launchesBlocked, coordinator.stopped
	coordinator.mutex.Unlock()
	if stopped {
		return errLaunchesBlocked
	}
	if !blocked {
		defer coordinator.allowLaunches()
	}
	if err := coordinator.forceShutdown(); err != nil {
		return fmt.Errorf("stop browser before runtime removal: %w", err)
	}
	return coordinator.runtime.Remove()
}

func (coordinator *Coordinator) Shutdown() {
	coordinator.lifecycleMutex.Lock()
	defer coordinator.lifecycleMutex.Unlock()
	coordinator.mutex.Lock()
	if coordinator.stopped {
		coordinator.mutex.Unlock()
		return
	}
	coordinator.stopped = true
	coordinator.mutex.Unlock()
	if err := coordinator.forceShutdown(); err != nil {
		coordinator.logger.Warn("browser runtime did not stop", "operation", "browser-shutdown", "error", err)
	}
	coordinator.cleanup.close()
}
func (coordinator *Coordinator) RetryCleanup() { coordinator.cleanup.retry() }

func (coordinator *Coordinator) bumpLocked() {
	coordinator.revision++
	revision := coordinator.revision
	if coordinator.changed != nil {
		go coordinator.changed(revision)
	}
}
func (coordinator *Coordinator) bump() {
	coordinator.mutex.Lock()
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}
func (coordinator *Coordinator) queueStaleSessions() {
	entries, err := os.ReadDir(coordinator.paths.BrowserTempRoot)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && validRandomID(entry.Name(), 24) {
			coordinator.cleanup.enqueue(entry.Name())
		}
	}
}

type cleanupQueue struct {
	root    string
	logger  *slog.Logger
	changed func()
	mutex   sync.Mutex
	pending map[string]string
	wake    chan struct{}
	done    chan struct{}
	stopped chan struct{}
}

func newCleanupQueue(root string, logger *slog.Logger, changed func()) *cleanupQueue {
	queue := &cleanupQueue{root: root, logger: logger, changed: changed, pending: make(map[string]string), wake: make(chan struct{}, 1), done: make(chan struct{}), stopped: make(chan struct{})}
	go queue.run()
	return queue
}
func (queue *cleanupQueue) createDirectory(browserSessionID string) (string, error) {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	if err := os.MkdirAll(queue.root, 0o755); err != nil {
		return "", fmt.Errorf("create browser temporary root: %w", err)
	}
	directory := filepath.Join(queue.root, browserSessionID)
	if err := os.Mkdir(directory, 0o700); err != nil {
		return "", fmt.Errorf("create browser session directory: %w", err)
	}
	return directory, nil
}
func (queue *cleanupQueue) enqueue(browserSessionID string) {
	queue.mutex.Lock()
	if _, exists := queue.pending[browserSessionID]; !exists {
		queue.pending[browserSessionID] = ""
	}
	queue.mutex.Unlock()
	select {
	case queue.wake <- struct{}{}:
	default:
	}
}
func (queue *cleanupQueue) retry() {
	select {
	case queue.wake <- struct{}{}:
	default:
	}
}
func (queue *cleanupQueue) snapshot() []CleanupState {
	queue.mutex.Lock()
	defer queue.mutex.Unlock()
	cleanupStates := make([]CleanupState, 0, len(queue.pending))
	for browserSessionID, err := range queue.pending {
		if err != "" {
			cleanupStates = append(cleanupStates, CleanupState{ID: browserSessionID, Error: err})
		}
	}
	return cleanupStates
}
func (queue *cleanupQueue) run() {
	defer close(queue.stopped)
	for {
		select {
		case <-queue.done:
			queue.process()
			return
		case <-queue.wake:
			queue.process()
		}
	}
}
func (queue *cleanupQueue) process() {
	queue.mutex.Lock()
	browserSessionIDs := make([]string, 0, len(queue.pending))
	for browserSessionID := range queue.pending {
		browserSessionIDs = append(browserSessionIDs, browserSessionID)
	}
	queue.mutex.Unlock()
	for _, browserSessionID := range browserSessionIDs {
		var err error
		if !validRandomID(browserSessionID, 24) {
			err = errors.New("browser cleanup target has an unexpected name")
		} else {
			err = removeManagedDirectory(queue.root, filepath.Join(queue.root, browserSessionID))
		}
		queue.mutex.Lock()
		if err == nil || errors.Is(err, os.ErrNotExist) {
			delete(queue.pending, browserSessionID)
		} else {
			queue.pending[browserSessionID] = "A temporary browser profile could not be removed."
			queue.logger.Warn("browser profile cleanup failed", "operation", "browser-cleanup", "error", err)
		}
		queue.mutex.Unlock()
	}
	queue.mutex.Lock()
	err := removeEmptyDirectory(queue.root)
	queue.mutex.Unlock()
	if err != nil {
		queue.logger.Warn("browser temporary root cleanup failed", "operation", "browser-cleanup", "error", err)
	}
	if queue.changed != nil {
		queue.changed()
	}
}
func (queue *cleanupQueue) close() {
	close(queue.done)
	<-queue.stopped
}

func (coordinator *Coordinator) finishStart(registered bool) {
	coordinator.mutex.Lock()
	if !registered {
		coordinator.pendingStarts--
		coordinator.bumpLocked()
	}
	coordinator.mutex.Unlock()
	coordinator.starts.Done()
}

func (coordinator *Coordinator) initializeSavedSession(ctx context.Context, current *session, state SessionState, cookie string) (SessionState, error) {
	browserSessionID, mode, accountID := state.ID, state.Mode, state.AccountID
	startupContext, startupCancel := context.WithTimeout(ctx, 12*time.Second)
	defer startupCancel()
	if err := coordinator.initializeCDP(startupContext, current); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	sessionID, err := coordinator.targetSession(startupContext, current)
	if err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	if err := current.cdp.SetBrowserID(startupContext, sessionID, current.browserID); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	if err := current.cdp.SetRobloxCookie(startupContext, sessionID, cookie); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, errors.New("the saved account cookie could not be prepared in the browser")
	}
	current.lastCookie = cookie
	if err := current.cdp.Navigate(startupContext, sessionID, "https://www.roblox.com/home"); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	state = coordinator.activateSession(current)
	if state.Lifecycle != LifecycleActive {
		return SessionState{}, errLaunchesBlocked
	}
	go coordinator.observe(current.ctx, current)
	go coordinator.watchProcess(current)
	_ = coordinator.Focus(browserSessionID)
	coordinator.logger.InfoContext(ctx, "managed browser session started", "operation", "browser-session-start", "session_id", browserSessionID, "session_mode", mode, "account_id", accountID)
	return state, nil
}

func (coordinator *Coordinator) handleTargetDestroyed(current *session, paramsData json.RawMessage) {
	var params struct {
		TargetID string `json:"targetId"`
	}
	if json.Unmarshal(paramsData, &params) == nil && params.TargetID != "" {
		coordinator.mutex.Lock()
		delete(current.targetSessions, params.TargetID)
		if current.targetID == params.TargetID {
			current.targetID = ""
		}
		_, page := current.pages[params.TargetID]
		delete(current.pages, params.TargetID)
		lastPage := page && len(current.pages) == 0
		coordinator.mutex.Unlock()
		if lastPage {
			current.process.PagesClosed()
		}
	}
}

func (coordinator *Coordinator) handleTargetDetached(current *session, paramsData json.RawMessage) {
	var params struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(paramsData, &params) == nil && params.SessionID != "" {
		coordinator.mutex.Lock()
		for targetID, sessionID := range current.targetSessions {
			if sessionID == params.SessionID {
				delete(current.targetSessions, targetID)
			}
		}
		coordinator.mutex.Unlock()
	}
}

func (coordinator *Coordinator) completeCheckpointValidation(ctx context.Context, current *session, final bool, generation uint64, err error) bool {
	if err != nil && ctx.Err() == nil {
		coordinator.setCheckpointWarning(current, "The browser cookie could not be validated. Sign in again and retry.")
		coordinator.logger.WarnContext(ctx, "browser cookie validation failed", "operation", "browser-cookie-checkpoint", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
	} else if err == nil {
		coordinator.setCheckpointWarning(current, "")
	}
	coordinator.mutex.Lock()
	coordinator.validationCount--
	allowed := current.state.Lifecycle == LifecycleActive || final && current.state.Lifecycle == LifecycleValidating
	stale := current.generation != generation || coordinator.sessions[current.state.ID] != current || !allowed
	if !stale && err != nil && ctx.Err() == nil {
		current.validatedGeneration = generation
	}
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	return stale
}

func (coordinator *Coordinator) checkpointCookie(ctx context.Context, current *session, final bool, sessionID, cookieValue string) {
	coordinator.mutex.Lock()
	if cookieValue == current.lastCookie && current.validatedGeneration == current.generation {
		coordinator.mutex.Unlock()
		return
	}
	if cookieValue != current.lastCookie {
		current.lastCookie = cookieValue
		current.generation++
	}
	generation := current.generation
	browserID, robloxUserID := current.browserID, current.robloxUserID
	coordinator.validationCount++
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
	validation, err := coordinator.backend.ValidateBrowserCookie(ctx, cookieValue, browserID)
	stale := coordinator.completeCheckpointValidation(ctx, current, final, generation, err)
	if stale || err != nil {
		validation.Cookie = ""
		return
	}
	if robloxUserID != 0 && robloxUserID != validation.Identity.RobloxUserID && validation.BrowserID == browserID {
		validation.BrowserID = roblox.NewBrowserID()
	}
	if err := current.cdp.SetBrowserID(ctx, sessionID, validation.BrowserID); err != nil {
		coordinator.setCheckpointWarning(current, "The account browser ID could not be synchronized.")
		return
	}
	coordinator.mutex.Lock()
	current.browserID = validation.BrowserID
	current.validatedGeneration = generation
	if current.state.Mode == ModeLogin {
		current.robloxUserID = validation.Identity.RobloxUserID
	}
	coordinator.mutex.Unlock()
	if validation.Cookie != cookieValue {
		if injectionError := current.cdp.SetRobloxCookie(ctx, sessionID, validation.Cookie); injectionError != nil {
			coordinator.setCheckpointWarning(current, "A rotated browser cookie could not be returned to the browser.")
		} else {
			coordinator.mutex.Lock()
			current.lastCookie = validation.Cookie
			coordinator.mutex.Unlock()
		}
	}
	coordinator.synchronizeCheckpoint(ctx, current, &validation)
}

func (coordinator *Coordinator) synchronizeCheckpoint(ctx context.Context, current *session, validation *roblox.CookieValidation) {
	if current.state.Mode == ModeLogin {
		coordinator.installCandidate(ctx, current, *validation)
		return
	}
	if validation.Identity.RobloxUserID != current.robloxUserID {
		validation.Cookie = ""
		coordinator.detach(current, "This browser signed in to a different Roblox account and is no longer synchronized.")
		return
	}
	newVersion, err := coordinator.backend.SynchronizeBrowserCookie(ctx, current.state.AccountID, current.secretVersion, *validation)
	validation.Cookie = ""
	if err != nil {
		coordinator.detach(current, "The saved account changed elsewhere. This browser is no longer synchronized.")
		return
	}
	coordinator.mutex.Lock()
	current.secretVersion = newVersion
	coordinator.bumpLocked()
	coordinator.mutex.Unlock()
}
