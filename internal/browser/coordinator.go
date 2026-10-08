package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
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

	mu              sync.Mutex
	savedStartMu    sync.Mutex
	revision        uint64
	sessions        map[string]*session
	candidates      map[string]*candidate
	launchesBlocked bool
	validationCount int
	root            context.Context
	cancel          context.CancelFunc
}

func NewCoordinator(paths appdata.Paths, runtimeManager *RuntimeManager, backend AccountSource, logger *slog.Logger, changed func(uint64)) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	coordinator := &Coordinator{paths: paths, runtime: runtimeManager, backend: backend, logger: logger, changed: changed, sessions: make(map[string]*session), candidates: make(map[string]*candidate), root: ctx, cancel: cancel}
	coordinator.cleanup = newCleanupQueue(paths.BrowserTempRoot, logger, coordinator.cleanupChanged)
	runtimeManager.SetChanged(coordinator.runtimeChanged)
	coordinator.queueStaleSessions()
	coordinator.cleanup.retry()
	return coordinator
}

func (coordinator *Coordinator) Snapshot() Snapshot {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	snapshot := Snapshot{Revision: coordinator.revision, Runtime: coordinator.runtime.State()}
	for _, current := range coordinator.sessions {
		snapshot.Sessions = append(snapshot.Sessions, current.state)
	}
	for _, current := range coordinator.candidates {
		snapshot.Candidates = append(snapshot.Candidates, current.state)
	}
	snapshot.Cleanup = coordinator.cleanup.snapshot()
	sort.Slice(snapshot.Sessions, func(i, j int) bool { return snapshot.Sessions[i].ID < snapshot.Sessions[j].ID })
	sort.Slice(snapshot.Candidates, func(i, j int) bool { return snapshot.Candidates[i].ID < snapshot.Candidates[j].ID })
	return snapshot
}

func (coordinator *Coordinator) Effects() ShutdownEffects {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	effects := ShutdownEffects{AccountCandidates: len(coordinator.candidates), AccountChecks: coordinator.validationCount > 0}
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
	coordinator.savedStartMu.Lock()
	defer coordinator.savedStartMu.Unlock()
	coordinator.mu.Lock()
	for _, current := range coordinator.sessions {
		if current.state.Mode == ModeSaved && current.state.AccountID == accountID && current.state.Lifecycle != LifecycleClosed && current.state.Lifecycle != LifecycleFailed {
			state := current.state
			coordinator.mu.Unlock()
			if err := coordinator.Focus(state.ID); err != nil {
				return state, err
			}
			return state, nil
		}
	}
	coordinator.mu.Unlock()
	return coordinator.start(ctx, ModeSaved, accountID)
}

func (coordinator *Coordinator) start(ctx context.Context, mode Mode, accountID int64) (SessionState, error) {
	coordinator.mu.Lock()
	blocked := coordinator.launchesBlocked
	coordinator.mu.Unlock()
	if blocked {
		return SessionState{}, errors.New("browser launches are unavailable while the vault is locking")
	}
	executable, err := coordinator.runtime.Executable()
	if err != nil {
		return SessionState{}, err
	}
	id, err := randomID(24)
	if err != nil {
		return SessionState{}, err
	}
	directory, err := coordinator.cleanup.createDirectory(id)
	if err != nil {
		coordinator.cleanup.retry()
		return SessionState{}, err
	}
	state := SessionState{ID: id, Mode: mode, Lifecycle: LifecycleStarting, AccountID: accountID, Attached: mode == ModeSaved, Title: "Roblox login"}
	var account accounts.AccountView
	record := roblox.SessionSnapshot{BrowserID: roblox.NewBrowserID()}
	if mode == ModeSaved {
		account, record, err = coordinator.backend.BrowserAccount(ctx, accountID)
		if err != nil {
			coordinator.cleanup.enqueue(id)
			return SessionState{}, err
		}
		state.Title = "Roblox - @" + account.Username
	}
	process, err := startBrowserProcess(ProcessOptions{Executable: executable, UserDataPath: directory})
	if err != nil {
		coordinator.cleanup.enqueue(id)
		return SessionState{}, err
	}
	currentCtx, cancel := context.WithCancel(coordinator.root)
	checkpointGate := make(chan struct{}, 1)
	checkpointGate <- struct{}{}
	current := &session{state: state, process: process, directory: directory, secretVersion: record.SecretVersion, robloxUserID: account.RobloxUserID, browserID: record.BrowserID, trigger: make(chan struct{}, 1), checkpointGate: checkpointGate, ctx: currentCtx, cancel: cancel, downloads: make(map[string]struct{}), targetSessions: make(map[string]string), pages: make(map[string]struct{})}
	input, output := process.CDPPipes()
	current.cdp = NewCDPClient(input, output, func(failure error) { coordinator.failSession(id, failure) })
	coordinator.mu.Lock()
	coordinator.sessions[id] = current
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	coordinator.logger.Info("managed browser session starting", "session_id", id, "session_mode", mode, "account_id", accountID)
	if mode == ModeLogin {
		go coordinator.initializeLogin(currentCtx, current)
		return state, nil
	}
	startupCtx, startupCancel := context.WithTimeout(ctx, 12*time.Second)
	defer startupCancel()
	if err := coordinator.initializeCDP(startupCtx, current); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	sessionID, err := coordinator.targetSession(startupCtx, current)
	if err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	if err := current.cdp.SetBrowserID(startupCtx, sessionID, current.browserID); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	if err := current.cdp.SetRobloxCookie(startupCtx, sessionID, record.Cookie); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, errors.New("the saved account cookie could not be prepared in the browser")
	}
	current.lastCookie = record.Cookie
	if err := current.cdp.Navigate(startupCtx, sessionID, "https://www.roblox.com/home"); err != nil {
		coordinator.abortStartup(current, err)
		return SessionState{}, err
	}
	state = coordinator.activateSession(current)
	go coordinator.observe(currentCtx, current)
	go coordinator.watchProcess(current)
	_ = coordinator.Focus(id)
	coordinator.logger.Info("managed browser session started", "session_id", id, "session_mode", mode, "account_id", accountID)
	return state, nil
}

func (coordinator *Coordinator) initializeLogin(ctx context.Context, current *session) {
	startupCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := coordinator.initializeCDP(startupCtx, current); err != nil {
		if ctx.Err() != nil {
			return
		}
		coordinator.abortStartup(current, err)
		return
	}
	sessionID, err := coordinator.targetSession(startupCtx, current)
	if err == nil {
		err = current.cdp.SetBrowserID(startupCtx, sessionID, current.browserID)
	}
	if err == nil {
		err = current.cdp.Navigate(startupCtx, sessionID, "https://www.roblox.com/Login")
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
	coordinator.logger.Info("managed browser session started", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
}

func (coordinator *Coordinator) initializeCDP(ctx context.Context, current *session) error {
	if err := current.cdp.VerifyVersion(ctx, coordinator.runtime.Manifest().Version); err != nil {
		if strings.Contains(err.Error(), "does not match") {
			coordinator.runtime.MarkDamaged()
		}
		return fmt.Errorf("verify managed browser: %w", err)
	}
	if err := current.cdp.InterceptRobloxLaunches(ctx, func(protocolURL string) {
		coordinator.mu.Lock()
		allowed := current.ctx.Err() == nil && !coordinator.launchesBlocked && !current.gameLaunchPending && coordinator.sessions[current.state.ID] == current && current.state.Lifecycle == LifecycleActive
		if allowed {
			current.gameLaunchPending = true
		}
		coordinator.mu.Unlock()
		if allowed {
			go func() {
				defer func() {
					coordinator.mu.Lock()
					current.gameLaunchPending = false
					coordinator.mu.Unlock()
				}()
				launchCtx, cancel := context.WithTimeout(current.ctx, 90*time.Second)
				defer cancel()
				coordinator.backend.LaunchBrowserGame(launchCtx, current.state.ID, protocolURL)
			}()
		}
	}); err != nil {
		return fmt.Errorf("prepare browser game launches: %w", err)
	}
	if err := current.cdp.DiscoverTargets(ctx); err != nil {
		return fmt.Errorf("discover browser targets: %w", err)
	}
	if err := current.cdp.EnableDownloadEvents(ctx); err != nil {
		coordinator.logger.Debug("browser download events unavailable", "session_id", current.state.ID, "session_mode", current.state.Mode, "error", err)
	}
	return nil
}

func (coordinator *Coordinator) activateSession(current *session) SessionState {
	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if stored := coordinator.sessions[current.state.ID]; stored == current && current.state.Lifecycle == LifecycleStarting {
		current.state.Lifecycle = LifecycleActive
		coordinator.bumpLocked()
	}
	return current.state
}

func (coordinator *Coordinator) abortStartup(current *session, cause error) {
	coordinator.mu.Lock()
	if stored := coordinator.sessions[current.state.ID]; stored != current || current.state.Lifecycle != LifecycleStarting {
		coordinator.mu.Unlock()
		return
	}
	current.state.Attached = false
	current.state.Lifecycle = LifecycleFailed
	current.state.Error = "The browser session could not be started."
	coordinator.bumpLocked()
	coordinator.mu.Unlock()

	current.cancel()
	current.closeOnce.Do(func() {
		cause = errors.Join(cause, current.process.ExitError())
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser startup teardown failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(2 * time.Second) {
			coordinator.logger.Warn("browser startup process did not exit before deadline", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
		_ = current.cdp.Close()
		_ = current.process.Close()
	})
	coordinator.cleanup.enqueue(current.state.ID)
	coordinator.logger.Error("browser startup failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
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
			TargetInfo TargetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.TargetInfo.Type == "page" {
			coordinator.mu.Lock()
			current.targetID = params.TargetInfo.TargetID
			current.pages[params.TargetInfo.TargetID] = struct{}{}
			coordinator.mu.Unlock()
		}
	case "Target.targetDestroyed":
		var params struct {
			TargetID string `json:"targetId"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.TargetID != "" {
			coordinator.mu.Lock()
			delete(current.targetSessions, params.TargetID)
			if current.targetID == params.TargetID {
				current.targetID = ""
			}
			_, page := current.pages[params.TargetID]
			delete(current.pages, params.TargetID)
			lastPage := page && len(current.pages) == 0
			coordinator.mu.Unlock()
			if lastPage {
				current.process.PagesClosed()
			}
		}
	case "Target.detachedFromTarget":
		var params struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.SessionID != "" {
			coordinator.mu.Lock()
			for targetID, sessionID := range current.targetSessions {
				if sessionID == params.SessionID {
					delete(current.targetSessions, targetID)
				}
			}
			coordinator.mu.Unlock()
		}
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
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] != current || current.state.Lifecycle != LifecycleActive {
		coordinator.mu.Unlock()
		return
	}
	if active {
		current.downloads[guid] = struct{}{}
	} else {
		delete(current.downloads, guid)
	}
	current.state.ActiveDownloads = len(current.downloads)
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
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
	coordinator.mu.Lock()
	if stored := coordinator.sessions[current.state.ID]; stored != current || (!final && current.state.Lifecycle != LifecycleActive) || !current.state.Attached && current.state.Mode == ModeSaved {
		coordinator.mu.Unlock()
		return
	}
	coordinator.mu.Unlock()
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
	value := selectedCookie.Value
	coordinator.mu.Lock()
	if value == current.lastCookie && current.validatedGeneration == current.generation {
		coordinator.mu.Unlock()
		return
	}
	if value != current.lastCookie {
		current.lastCookie = value
		current.generation++
	}
	generation := current.generation
	browserID, robloxUserID := current.browserID, current.robloxUserID
	coordinator.validationCount++
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	validation, err := coordinator.backend.ValidateBrowserCookie(ctx, value, browserID)
	if err != nil && ctx.Err() == nil {
		coordinator.setCheckpointWarning(current, "The browser cookie could not be validated. Sign in again and retry.")
		coordinator.logger.Warn("browser cookie validation failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
	} else if err == nil {
		coordinator.setCheckpointWarning(current, "")
	}
	coordinator.mu.Lock()
	coordinator.validationCount--
	allowed := current.state.Lifecycle == LifecycleActive || final && current.state.Lifecycle == LifecycleValidating
	stale := current.generation != generation || coordinator.sessions[current.state.ID] != current || !allowed
	if !stale && err != nil && ctx.Err() == nil {
		current.validatedGeneration = generation
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	if stale || err != nil {
		value = ""
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
	coordinator.mu.Lock()
	current.browserID = validation.BrowserID
	current.validatedGeneration = generation
	if current.state.Mode == ModeLogin {
		current.robloxUserID = validation.Identity.RobloxUserID
	}
	coordinator.mu.Unlock()
	if validation.Cookie != value {
		if injectionErr := current.cdp.SetRobloxCookie(ctx, sessionID, validation.Cookie); injectionErr != nil {
			coordinator.setCheckpointWarning(current, "A rotated browser cookie could not be returned to the browser.")
		} else {
			coordinator.mu.Lock()
			current.lastCookie = validation.Cookie
			coordinator.mu.Unlock()
		}
	}
	value = ""
	if current.state.Mode == ModeLogin {
		coordinator.installCandidate(ctx, current, validation)
		return
	}
	if validation.Identity.RobloxUserID != current.robloxUserID {
		validation.Cookie = ""
		coordinator.detach(current, "This browser signed in to a different Roblox account and is no longer synchronized.")
		return
	}
	newVersion, err := coordinator.backend.SynchronizeBrowserCookie(ctx, current.state.AccountID, current.secretVersion, validation)
	validation.Cookie = ""
	if err != nil {
		coordinator.detach(current, "The saved account changed elsewhere. This browser is no longer synchronized.")
		return
	}
	coordinator.mu.Lock()
	current.secretVersion = newVersion
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) recordCheckpointFailure(current *session, cause error) {
	message := cause.Error()
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] != current || current.lastCheckpointError == message {
		coordinator.mu.Unlock()
		return
	}
	current.lastCheckpointError = message
	coordinator.mu.Unlock()
	coordinator.logger.Warn("browser cookie checkpoint unavailable", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
}

func (coordinator *Coordinator) clearCheckpointFailure(current *session) {
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] != current {
		coordinator.mu.Unlock()
		return
	}
	recovered := current.lastCheckpointError != ""
	current.lastCheckpointError = ""
	coordinator.mu.Unlock()
	if recovered {
		coordinator.logger.Info("browser cookie checkpoint recovered", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	}
}

func (coordinator *Coordinator) installCandidate(ctx context.Context, current *session, validation roblox.CookieValidation) {
	coordinator.mu.Lock()
	allowed := coordinator.sessions[current.state.ID] == current && (current.state.Lifecycle == LifecycleActive || current.state.Lifecycle == LifecycleValidating)
	coordinator.mu.Unlock()
	if !allowed {
		validation.Cookie = ""
		return
	}
	id, err := randomID(24)
	if err != nil {
		validation.Cookie = ""
		coordinator.logger.Error("browser candidate identifier generation failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "error", err)
		return
	}
	avatar := coordinator.backend.BrowserAvatar(ctx, validation.Identity.RobloxUserID)
	existingAccount, updatesExisting := coordinator.backend.BrowserCandidateAccount(ctx, validation.Identity.RobloxUserID)
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] != current || current.state.Lifecycle != LifecycleActive && current.state.Lifecycle != LifecycleValidating {
		coordinator.mu.Unlock()
		validation.Cookie = ""
		return
	}
	for _, existing := range coordinator.candidates {
		if existing.state.SessionID == current.state.ID && existing.state.RobloxUserID == validation.Identity.RobloxUserID {
			existing.validation.Cookie = ""
			existing.validation = validation
			existing.state.Username = validation.Identity.Username
			existing.state.DisplayName = validation.Identity.DisplayName
			existing.state.AvatarURL = avatar
			existing.state.UpdatesExisting = updatesExisting && existingAccount.ID > 0
			updatesSavedAccount := existing.state.UpdatesExisting
			coordinator.bumpLocked()
			coordinator.mu.Unlock()
			coordinator.logger.Info("browser account candidate updated", "session_id", current.state.ID, "session_mode", current.state.Mode, "roblox_user_id", validation.Identity.RobloxUserID, "updates_existing", updatesSavedAccount)
			return
		}
	}
	state := CandidateState{ID: id, SessionID: current.state.ID, RobloxUserID: validation.Identity.RobloxUserID, Username: validation.Identity.Username, DisplayName: validation.Identity.DisplayName, AvatarURL: avatar, UpdatesExisting: updatesExisting && existingAccount.ID > 0}
	for _, existing := range coordinator.candidates {
		if existing.state.RobloxUserID == state.RobloxUserID {
			state.SameAccountAsID = existing.state.ID
			break
		}
	}
	coordinator.candidates[id] = &candidate{state: state, validation: validation}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	coordinator.logger.Info("browser account candidate captured", "session_id", current.state.ID, "session_mode", current.state.Mode, "roblox_user_id", validation.Identity.RobloxUserID, "updates_existing", state.UpdatesExisting)
}

func (coordinator *Coordinator) Focus(id string) error {
	coordinator.mu.Lock()
	current := coordinator.sessions[id]
	if current == nil {
		coordinator.mu.Unlock()
		return errors.New("browser session not found")
	}
	target := current.targetID
	coordinator.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if target == "" {
		targets, err := current.cdp.Targets(ctx)
		if err == nil {
			for index := len(targets) - 1; index >= 0; index-- {
				if targets[index].Type == "page" {
					target = targets[index].TargetID
					break
				}
			}
		}
	}
	if target != "" {
		if err := current.cdp.Activate(ctx, target); err != nil {
			coordinator.logger.Debug("browser target activation failed", "session_id", id, "session_mode", current.state.Mode, "error", err)
		}
	}
	if err := current.process.Focus(); err != nil {
		coordinator.logger.Warn("browser window focus failed", "session_id", id, "session_mode", current.state.Mode, "error", err)
		return err
	}
	return nil
}

func (coordinator *Coordinator) targetSession(ctx context.Context, current *session) (string, error) {
	coordinator.mu.Lock()
	targetID := current.targetID
	coordinator.mu.Unlock()
	if targetID == "" {
		for targetID == "" {
			targets, err := current.cdp.Targets(ctx)
			if err != nil {
				return "", err
			}
			for index := len(targets) - 1; index >= 0; index-- {
				if targets[index].Type == "page" {
					targetID = targets[index].TargetID
					break
				}
			}
			if targetID != "" {
				break
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
		}
	}
	return coordinator.targetSessionFor(ctx, current, targetID)
}

func (coordinator *Coordinator) targetSessionFor(ctx context.Context, current *session, targetID string) (string, error) {
	coordinator.mu.Lock()
	if sessionID := current.targetSessions[targetID]; sessionID != "" {
		coordinator.mu.Unlock()
		return sessionID, nil
	}
	coordinator.mu.Unlock()
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
	coordinator.mu.Lock()
	current.targetSessions[targetID] = sessionID
	current.targetID = targetID
	coordinator.mu.Unlock()
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
	var lastErr error
	for index := len(targets) - 1; index >= 0; index-- {
		if targets[index].Type != "page" {
			continue
		}
		sessionID, attachErr := coordinator.targetSessionFor(ctx, current, targets[index].TargetID)
		if attachErr != nil {
			lastErr = attachErr
			continue
		}
		if firstSession == "" {
			firstSession = sessionID
		}
		cookies, cookieErr := current.cdp.Cookies(ctx, sessionID)
		if cookieErr != nil {
			lastErr = cookieErr
			continue
		}
		if len(cookies) > 0 {
			return sessionID, cookies, nil
		}
	}
	if firstSession != "" {
		return firstSession, nil, nil
	}
	if lastErr != nil {
		return "", nil, lastErr
	}
	return "", nil, errors.New("browser page target not found")
}

func (coordinator *Coordinator) Close(id string) error {
	coordinator.mu.Lock()
	current := coordinator.sessions[id]
	if current == nil {
		coordinator.mu.Unlock()
		return nil
	}
	if current.state.Lifecycle == LifecycleFailed {
		delete(coordinator.sessions, id)
		coordinator.bumpLocked()
		coordinator.mu.Unlock()
		coordinator.logger.Info("failed browser session dismissed", "session_id", id, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		return nil
	}
	if current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating || current.state.Lifecycle == LifecycleClosed {
		coordinator.mu.Unlock()
		return nil
	}
	current.state.Lifecycle = LifecycleClosing
	current.cancel()
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	coordinator.logger.Info("managed browser session closing", "session_id", id, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	go coordinator.closeGracefully(current)
	return nil
}

func (coordinator *Coordinator) closeGracefully(current *session) {
	coordinator.mu.Lock()
	if current.state.Mode == ModeLogin {
		current.state.Lifecycle = LifecycleValidating
		coordinator.bumpLocked()
	}
	coordinator.mu.Unlock()
	checkpointCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	coordinator.checkpoint(checkpointCtx, current, true)
	cancel()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := current.cdp.CloseBrowser(closeCtx); err != nil && !errors.Is(err, context.Canceled) {
		coordinator.logger.Debug("graceful browser close unavailable", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
	}
	closeCancel()
	if !current.process.Wait(2 * time.Second) {
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser process termination failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(2 * time.Second) {
			coordinator.logger.Warn("browser process did not exit before deadline", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
	}
	coordinator.finishSession(current, false, "")
}

func (coordinator *Coordinator) CloseAllLogin() {
	coordinator.mu.Lock()
	ids := []string{}
	for id, current := range coordinator.sessions {
		if current.state.Mode == ModeLogin && current.state.Lifecycle != LifecycleClosed {
			ids = append(ids, id)
		}
	}
	coordinator.mu.Unlock()
	for _, id := range ids {
		_ = coordinator.Close(id)
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
			coordinator.logger.Info("all managed browser windows closed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
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
						coordinator.logger.Info("all managed browser windows closed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
						_ = coordinator.Close(current.state.ID)
						return
					}
				default:
				}
			}
			coordinator.mu.Lock()
			expected := current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating
			coordinator.mu.Unlock()
			if !expected {
				coordinator.failSession(current.state.ID, err)
			}
			return
		}
	}
}

func (coordinator *Coordinator) failSession(id string, cause error) {
	coordinator.mu.Lock()
	current := coordinator.sessions[id]
	if current == nil {
		coordinator.mu.Unlock()
		return
	}
	if current.state.Lifecycle == LifecycleStarting {
		coordinator.mu.Unlock()
		coordinator.abortStartup(current, cause)
		return
	}
	if current.state.Lifecycle == LifecycleClosing || current.state.Lifecycle == LifecycleValidating || current.state.Lifecycle == LifecycleClosed || current.state.Lifecycle == LifecycleFailed {
		coordinator.mu.Unlock()
		return
	}
	coordinator.mu.Unlock()
	if !coordinator.finishSession(current, true, "The browser closed unexpectedly. The final cookie could not be checked.") {
		return
	}
	if cause == nil {
		coordinator.logger.Warn("browser session exited unexpectedly", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
	} else {
		coordinator.logger.Warn("browser session failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", cause)
	}
}

func (coordinator *Coordinator) finishSession(current *session, failed bool, warning string) bool {
	finished := false
	current.closeOnce.Do(func() {
		finished = true
		current.cancel()
		if err := current.process.Terminate(); err != nil {
			coordinator.logger.Warn("browser process containment termination failed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID, "error", err)
		}
		if !current.process.Wait(time.Second) {
			coordinator.logger.Warn("browser process containment did not stop before cleanup", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
		_ = current.cdp.Close()
		_ = current.process.Close()
		coordinator.mu.Lock()
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
		coordinator.mu.Unlock()
		coordinator.cleanup.enqueue(current.state.ID)
		if !failed {
			coordinator.logger.Info("managed browser session closed", "session_id", current.state.ID, "session_mode", current.state.Mode, "account_id", current.state.AccountID)
		}
	})
	return finished
}

func (coordinator *Coordinator) setCheckpointWarning(current *session, message string) {
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] != current {
		coordinator.mu.Unlock()
		return
	}
	current.state.CheckpointWarning = message
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}
func (coordinator *Coordinator) detach(current *session, message string) {
	coordinator.mu.Lock()
	if coordinator.sessions[current.state.ID] == current && current.state.Attached {
		current.state.Attached = false
		current.state.Error = message
		coordinator.bumpLocked()
	}
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) DetachAccount(accountID int64, message string) {
	coordinator.mu.Lock()
	for _, current := range coordinator.sessions {
		if current.state.AccountID == accountID && current.state.Attached {
			current.state.Attached = false
			current.state.Error = message
		}
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) SaveCandidates(ctx context.Context, selected []string) (SaveResult, error) {
	coordinator.mu.Lock()
	if len(selected) == 0 {
		coordinator.mu.Unlock()
		return SaveResult{}, errors.New("select at least one account candidate")
	}
	seen := map[int64]struct{}{}
	chosen := make([]*candidate, 0, len(selected))
	chosenIDs := map[string]struct{}{}
	for _, id := range selected {
		item := coordinator.candidates[id]
		if item == nil {
			coordinator.mu.Unlock()
			return SaveResult{}, errors.New("an account candidate is no longer available")
		}
		if _, duplicate := seen[item.state.RobloxUserID]; duplicate {
			coordinator.mu.Unlock()
			return SaveResult{}, errors.New("select only one candidate for each account")
		}
		seen[item.state.RobloxUserID] = struct{}{}
		chosenIDs[id] = struct{}{}
		chosen = append(chosen, item)
	}
	for id, item := range coordinator.candidates {
		if _, keep := chosenIDs[id]; !keep {
			item.validation.Cookie = ""
		}
		delete(coordinator.candidates, id)
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	result := SaveResult{Items: make([]SaveResultItem, 0, len(chosen))}
	for _, item := range chosen {
		account, status, err := coordinator.backend.CommitBrowserCandidate(ctx, item.validation)
		item.validation.Cookie = ""
		resultItem := SaveResultItem{CandidateID: item.state.ID, Status: status, Account: account}
		if err != nil {
			resultItem.Status, resultItem.Error = SaveFailed, "The account could not be saved."
		}
		result.Items = append(result.Items, resultItem)
	}
	return result, nil
}

func (coordinator *Coordinator) DiscardCandidates() {
	coordinator.mu.Lock()
	for id, item := range coordinator.candidates {
		item.validation.Cookie = ""
		delete(coordinator.candidates, id)
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) ForceShutdown() {
	coordinator.mu.Lock()
	coordinator.launchesBlocked = true
	sessions := make([]*session, 0, len(coordinator.sessions))
	for _, current := range coordinator.sessions {
		if current.state.Lifecycle != LifecycleClosed && current.state.Lifecycle != LifecycleFailed {
			current.state.Lifecycle = LifecycleClosing
		}
		sessions = append(sessions, current)
	}
	for id, item := range coordinator.candidates {
		item.validation.Cookie = ""
		delete(coordinator.candidates, id)
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
	for _, current := range sessions {
		current.cancel()
		coordinator.finishSession(current, false, "")
	}
	_ = coordinator.runtime.StopDownload(30 * time.Second)
	coordinator.mu.Lock()
	for id := range coordinator.sessions {
		delete(coordinator.sessions, id)
	}
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}

func (coordinator *Coordinator) AllowLaunches() {
	coordinator.mu.Lock()
	coordinator.launchesBlocked = false
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}
func (coordinator *Coordinator) Shutdown() {
	coordinator.ForceShutdown()
	coordinator.cancel()
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
func (coordinator *Coordinator) runtimeChanged() {
	coordinator.mu.Lock()
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
}
func (coordinator *Coordinator) cleanupChanged() {
	coordinator.mu.Lock()
	coordinator.bumpLocked()
	coordinator.mu.Unlock()
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
	mu      sync.Mutex
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
func (queue *cleanupQueue) createDirectory(id string) (string, error) {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	// The owner can write. Everyone can read and enter these directories.
	if err := os.MkdirAll(queue.root, 0o755); err != nil {
		return "", fmt.Errorf("create browser temporary root: %w", err)
	}
	directory := filepath.Join(queue.root, id)
	if err := os.Mkdir(directory, 0o755); err != nil {
		return "", fmt.Errorf("create browser session directory: %w", err)
	}
	return directory, nil
}
func (queue *cleanupQueue) enqueue(id string) {
	queue.mu.Lock()
	if _, exists := queue.pending[id]; !exists {
		queue.pending[id] = ""
	}
	queue.mu.Unlock()
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
	queue.mu.Lock()
	defer queue.mu.Unlock()
	result := make([]CleanupState, 0, len(queue.pending))
	for id, err := range queue.pending {
		if err != "" {
			result = append(result, CleanupState{ID: id, Error: err})
		}
	}
	return result
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
	queue.mu.Lock()
	ids := make([]string, 0, len(queue.pending))
	for id := range queue.pending {
		ids = append(ids, id)
	}
	queue.mu.Unlock()
	for _, id := range ids {
		var err error
		if !validRandomID(id, 24) {
			err = errors.New("browser cleanup target has an unexpected name")
		} else {
			err = removeManagedDirectory(queue.root, filepath.Join(queue.root, id))
		}
		queue.mu.Lock()
		if err == nil || errors.Is(err, os.ErrNotExist) {
			delete(queue.pending, id)
		} else {
			queue.pending[id] = "A temporary browser profile could not be removed."
			queue.logger.Warn("browser profile cleanup failed", "error", err)
		}
		queue.mu.Unlock()
	}
	queue.mu.Lock()
	err := removeEmptyDirectory(queue.root)
	queue.mu.Unlock()
	if err != nil {
		queue.logger.Warn("browser temporary root cleanup failed", "error", err)
	}
	if queue.changed != nil {
		queue.changed()
	}
}
func (queue *cleanupQueue) close() {
	close(queue.done)
	<-queue.stopped
}
