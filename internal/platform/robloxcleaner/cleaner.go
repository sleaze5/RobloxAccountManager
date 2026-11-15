package robloxcleaner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

type Stage string

const (
	StageClosingProcesses        Stage = "closing-processes"
	StageDeletingFiles           Stage = "deleting-files"
	StageDeletingRegistryEntries Stage = "deleting-registry-entries"
)

// Progress counts the planned deletions. While Roblox processes close, the
// total is not known yet and is 0.
type Progress struct {
	Stage          Stage `json:"stage"`
	CompletedCount int   `json:"completedCount"`
	TotalCount     int   `json:"totalCount"`
}

// Outcome counts what cleaning deleted and what it skipped because it was in
// use, needs administrator rights, or is a link.
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

// Clean deletes Roblox files and registry entries of the request scope.
func Clean(ctx context.Context, request CleanRequest, reportProgress func(Progress), logger *slog.Logger) (Outcome, error) {
	if !Supported {
		return Outcome{}, errors.New("cleaning Roblox is only available on Windows")
	}
	if !ValidScope(request.Scope) {
		return Outcome{}, fmt.Errorf("unknown Roblox cleaning scope %q", request.Scope)
	}
	if request.ShouldCloseProcesses {
		reportProgress(Progress{Stage: StageClosingProcesses})
		processes, err := robloxmulti.ListProcesses()
		if err != nil {
			return Outcome{}, fmt.Errorf("list Roblox processes before cleaning: %w", err)
		}
		if err := robloxmulti.KillProcesses(ctx, processes.Processes, logger); err != nil {
			return Outcome{}, fmt.Errorf("close Roblox processes before cleaning: %w", err)
		}
	}
	return clean(ctx, request.Scope, request.ShouldDeleteLogs || request.Scope == ScopeUninstall, reportProgress, logger)
}
