package robloxcleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
)

type Scope string

const (
	// ScopeData deletes caches, logs, cookies, and other account traces, and
	// keeps the installed Roblox Player and Roblox Studio with their settings.
	ScopeData Scope = "data"
	// ScopeUninstall deletes every Roblox installation, setting, shortcut,
	// file association, and registry entry.
	ScopeUninstall Scope = "uninstall"
)

const CleanTaskName = "clean-roblox"

// Phase is one group of work that cleaning runs in order.
type Phase string

const (
	PhaseClosingProcesses Phase = "closing-processes"
	// PhaseRemovingApplications removes the install folders and shortcuts.
	PhaseRemovingApplications    Phase = "removing-applications"
	PhaseRemovingRegistryEntries Phase = "removing-registry-entries"
	PhaseDeletingUserContent     Phase = "deleting-user-content"
	// PhaseDeletingCaches deletes caches, cookies, the Roblox Studio sign-in,
	// and temporary files.
	PhaseDeletingCaches Phase = "deleting-caches"
	PhaseDeletingLogs   Phase = "deleting-logs"
)

type PhaseState string

const (
	PhaseStateWaiting PhaseState = "waiting"
	PhaseStateRunning PhaseState = "running"
	PhaseStateDone    PhaseState = "done"
)

// PhaseProgress counts the items of one phase. The totals are known when
// cleaning starts, except that the total of closing processes grows while they
// close, because closing one can start another, such as the crash handler.
type PhaseProgress struct {
	Phase          Phase      `json:"phase"`
	State          PhaseState `json:"state"`
	CompletedCount int        `json:"completedCount"`
	TotalCount     int        `json:"totalCount"`
	SkippedCount   int        `json:"skippedCount"`
}

type ActivityAction string

const (
	ActivityActionClosed  ActivityAction = "closed"
	ActivityActionDeleted ActivityAction = "deleted"
	ActivityActionSkipped ActivityAction = "skipped"
)

type SkipReason string

const (
	SkipReasonInUse                    SkipReason = "in-use"
	SkipReasonNeedsAdministratorRights SkipReason = "needs-administrator-rights"
	SkipReasonLink                     SkipReason = "link"
	SkipReasonFailed                   SkipReason = "failed"
)

// Activity is one process that was closed, or one item that was deleted or
// skipped. Items that did not exist are not reported. Item is the full path,
// registry key, or process name, which the user sees but logs never contain.
type Activity struct {
	Action     ActivityAction `json:"action"`
	Item       string         `json:"item"`
	SkipReason SkipReason     `json:"skipReason,omitempty"`
}

// Progress lists every phase of a cleaning in the order they run. Activity is
// set when the report was caused by a closed, deleted, or skipped item.
type Progress struct {
	Phases   []PhaseProgress `json:"phases"`
	Activity *Activity       `json:"activity,omitempty"`
}

// Outcome counts what cleaning deleted and what it skipped because it was in
// use, needs administrator rights, or is a link. Roblox processes that could
// not be closed count as skipped.
type Outcome struct {
	DeletedCount int `json:"deletedCount"`
	SkippedCount int `json:"skippedCount"`
}

type CleanRequest struct {
	Scope                Scope
	ShouldCloseProcesses bool
	// ShouldDeleteLogs applies to ScopeData. ScopeUninstall always deletes logs.
	ShouldDeleteLogs bool
}

func ValidScope(scope Scope) bool {
	return scope == ScopeData || scope == ScopeUninstall
}

// Arguments encodes the request for CleanTask in the elevated helper.
func (request CleanRequest) Arguments() []string {
	return []string{string(request.Scope), strconv.FormatBool(request.ShouldCloseProcesses), strconv.FormatBool(request.ShouldDeleteLogs)}
}

func (request CleanRequest) shouldDeleteLogs() bool {
	return request.ShouldDeleteLogs || request.Scope == ScopeUninstall
}

// deletionPhases lists the phases that delete files and registry entries.
// Uninstalling removes the applications and their shortcuts first, so that
// nothing can start Roblox while its remaining files are deleted.
func (request CleanRequest) deletionPhases() []Phase {
	var phases []Phase
	if request.Scope == ScopeUninstall {
		phases = append(phases, PhaseRemovingApplications, PhaseRemovingRegistryEntries, PhaseDeletingUserContent)
	}
	phases = append(phases, PhaseDeletingCaches)
	if request.shouldDeleteLogs() {
		phases = append(phases, PhaseDeletingLogs)
	}
	return phases
}

func (request CleanRequest) phases() []Phase {
	var phases []Phase
	if request.ShouldCloseProcesses {
		phases = append(phases, PhaseClosingProcesses)
	}
	return append(phases, request.deletionPhases()...)
}

// InitialProgress lists every phase of the request as waiting. It is shown
// before cleaning starts, such as while Windows asks for administrator rights.
func InitialProgress(request CleanRequest) Progress {
	phases := request.phases()
	progress := Progress{Phases: make([]PhaseProgress, 0, len(phases))}
	for _, phase := range phases {
		progress.Phases = append(progress.Phases, PhaseProgress{Phase: phase, State: PhaseStateWaiting})
	}
	return progress
}

func parseCleanRequest(arguments []string) (CleanRequest, error) {
	if len(arguments) != 3 || !ValidScope(Scope(arguments[0])) {
		return CleanRequest{}, errors.New("invalid Roblox cleaning arguments")
	}
	shouldCloseProcesses, closeError := strconv.ParseBool(arguments[1])
	shouldDeleteLogs, logsError := strconv.ParseBool(arguments[2])
	if err := errors.Join(closeError, logsError); err != nil {
		return CleanRequest{}, fmt.Errorf("parse Roblox cleaning arguments: %w", err)
	}
	return CleanRequest{Scope: Scope(arguments[0]), ShouldCloseProcesses: shouldCloseProcesses, ShouldDeleteLogs: shouldDeleteLogs}, nil
}

// CleanTask runs a request that Arguments encoded. The elevated helper runs it.
func CleanTask(ctx context.Context, arguments []string, reportProgress func(any), logger *slog.Logger) (any, error) {
	request, err := parseCleanRequest(arguments)
	if err != nil {
		return nil, err
	}
	return Clean(ctx, request, func(progress Progress) { reportProgress(progress) }, logger)
}

// Clean runs the phases of the request in order. An item that cannot be
// closed or deleted is skipped, and the remaining items are still cleaned.
func Clean(ctx context.Context, request CleanRequest, reportProgress func(Progress), logger *slog.Logger) (Outcome, error) {
	if !Supported {
		return Outcome{}, errors.New("cleaning Roblox is only available on Windows")
	}
	if !ValidScope(request.Scope) {
		return Outcome{}, fmt.Errorf("unknown Roblox cleaning scope %q", request.Scope)
	}
	logger = logger.With("scope", request.Scope, "should_delete_logs", request.shouldDeleteLogs())
	tracker := &progressTracker{progress: InitialProgress(request), reportProgress: reportProgress, logger: logger}
	// Every target is listed first, so that the totals are known from the
	// start. A target that an earlier phase deleted counts as done.
	deletionPhases := request.deletionPhases()
	targetsByPhase := make(map[Phase][]cleanupTarget, len(deletionPhases))
	for _, phase := range deletionPhases {
		targetsByPhase[phase] = listPhaseTargets(phase)
		tracker.setTotal(phase, len(targetsByPhase[phase]))
	}
	tracker.report(nil)
	if request.ShouldCloseProcesses {
		tracker.beginPhase(PhaseClosingProcesses)
		if err := closeProcesses(ctx, tracker, logger); err != nil {
			return Outcome{}, err
		}
		tracker.endPhase()
	}
	for _, phase := range deletionPhases {
		tracker.beginPhase(phase)
		deleteTargets(ctx, targetsByPhase[phase], tracker)
		// A cancelled phase stays running in the last report, so that it does
		// not look finished.
		if err := ctx.Err(); err != nil {
			return Outcome{}, fmt.Errorf("clean Roblox: %w", err)
		}
		tracker.endPhase()
	}
	outcome := tracker.outcome()
	logger.Info("Roblox cleaned", "deleted_count", outcome.DeletedCount, "skipped_count", outcome.SkippedCount)
	return outcome, nil
}

// closingRoundLimit bounds how often the process list is read again. Closing
// the player can start the crash handler, and a launcher can start the player.
const closingRoundLimit = 3

// closeProcesses closes every Roblox process, then reads the process list again
// to close processes that started meanwhile. A process that cannot be closed,
// such as one that runs with administrator rights, is skipped, and its files
// are skipped later because they are in use.
func closeProcesses(ctx context.Context, tracker *progressTracker, logger *slog.Logger) error {
	attemptedProcesses := map[uint32]string{}
	for range closingRoundLimit {
		snapshot, err := robloxmulti.ListProcesses()
		if err != nil {
			return fmt.Errorf("list Roblox processes before cleaning: %w", err)
		}
		newProcesses := slices.DeleteFunc(snapshot.Processes, func(process robloxmulti.Process) bool {
			return attemptedProcesses[process.PID] == process.StartTime
		})
		if len(newProcesses) == 0 {
			return nil
		}
		tracker.addToTotal(len(newProcesses))
		for _, process := range newProcesses {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("close Roblox processes before cleaning: %w", err)
			}
			attemptedProcesses[process.PID] = process.StartTime
			displayName := fmt.Sprintf("%s (process %d)", process.Name, process.PID)
			if err := robloxmulti.KillProcesses(ctx, []robloxmulti.Process{process}, logger); err != nil {
				tracker.completeItem(itemSkipped, process.Name, displayName, err)
				continue
			}
			tracker.completeItem(itemClosed, process.Name, displayName, nil)
		}
	}
	return nil
}

type itemOutcome int

const (
	itemClosed itemOutcome = iota
	itemDeleted
	itemMissing
	itemSkipped
)

// progressTracker keeps the state of every phase and reports all of them
// after each change, so that a listener renders the whole list from one report.
type progressTracker struct {
	progress       Progress
	reportProgress func(Progress)
	logger         *slog.Logger
	phaseIndex     int
	deletedCount   int
}

func (tracker *progressTracker) indexOf(phase Phase) int {
	return slices.IndexFunc(tracker.progress.Phases, func(phaseProgress PhaseProgress) bool {
		return phaseProgress.Phase == phase
	})
}

func (tracker *progressTracker) currentPhase() *PhaseProgress {
	return &tracker.progress.Phases[tracker.phaseIndex]
}

func (tracker *progressTracker) setTotal(phase Phase, totalCount int) {
	tracker.progress.Phases[tracker.indexOf(phase)].TotalCount = totalCount
}

func (tracker *progressTracker) beginPhase(phase Phase) {
	tracker.phaseIndex = tracker.indexOf(phase)
	tracker.currentPhase().State = PhaseStateRunning
	tracker.report(nil)
}

func (tracker *progressTracker) addToTotal(count int) {
	tracker.currentPhase().TotalCount += count
	tracker.report(nil)
}

// completeItem counts an item of the current phase. The label goes into logs
// and contains no user name. The display name is what the user sees.
func (tracker *progressTracker) completeItem(outcome itemOutcome, label, displayName string, err error) {
	current := tracker.currentPhase()
	current.CompletedCount++
	var activity *Activity
	switch outcome {
	case itemClosed:
		activity = &Activity{Action: ActivityActionClosed, Item: displayName}
	case itemDeleted:
		tracker.deletedCount++
		activity = &Activity{Action: ActivityActionDeleted, Item: displayName}
	case itemSkipped:
		current.SkippedCount++
		activity = &Activity{Action: ActivityActionSkipped, Item: displayName, SkipReason: skipReasonOf(err)}
		tracker.logger.Debug("Roblox item skipped", "phase", current.Phase, "item", label, "error", err)
	case itemMissing:
	}
	tracker.report(activity)
}

func (tracker *progressTracker) endPhase() {
	current := tracker.currentPhase()
	current.State = PhaseStateDone
	current.CompletedCount = current.TotalCount
	tracker.report(nil)
}

// report sends a copy, because the listener can encode it after the tracker
// changed again.
func (tracker *progressTracker) report(activity *Activity) {
	tracker.reportProgress(Progress{Phases: slices.Clone(tracker.progress.Phases), Activity: activity})
}

func (tracker *progressTracker) outcome() Outcome {
	outcome := Outcome{DeletedCount: tracker.deletedCount}
	for _, phase := range tracker.progress.Phases {
		outcome.SkippedCount += phase.SkippedCount
	}
	return outcome
}
