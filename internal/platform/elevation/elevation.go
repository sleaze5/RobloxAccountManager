package elevation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	taskArgumentPrefix = "--elevated-task="
	userArgumentPrefix = "--elevated-user="
	// A deep cleanup deletes whole Roblox installations, which can take a while
	// on a slow disk, but a stuck helper must not hold the launch forever.
	taskTimeout = 3 * time.Minute
)

// Exit codes of the elevated helper process. They start above 2 so that a
// crash or a runtime failure, which exits with 1 or 2, is never read as success.
const (
	exitCompleted                 = 0
	exitCompletedWithSkippedItems = 3
	exitFailed                    = 4
	exitOtherUser                 = 5
	exitInvalidArguments          = 6
)

var (
	ErrDeclined    = errors.New("administrator rights were declined")
	ErrOtherUser   = errors.New("administrator rights were given by another Windows user")
	ErrUnsupported = errors.New("administrator rights are only available on Windows")
)

// Task runs inside the elevated helper process and reports whether it skipped
// items that it could not change.
type Task func(ctx context.Context, arguments []string, logger *slog.Logger) (hasSkippedItems bool, err error)

// HandleHelperMode runs the task named on the command line when this process
// was started as an elevated helper. It reports whether it did, and the exit code.
func HandleHelperMode(tasks map[string]Task, logger *slog.Logger) (isHelper bool, exitCode int) {
	// Arguments: executable, --elevated-task=<name>, --elevated-user=<SID>, task arguments.
	if len(os.Args) < 3 || !strings.HasPrefix(os.Args[1], taskArgumentPrefix) {
		return false, 0
	}
	taskName := strings.TrimPrefix(os.Args[1], taskArgumentPrefix)
	task, taskExists := tasks[taskName]
	requestingUserSID, hasUser := strings.CutPrefix(os.Args[2], userArgumentPrefix)
	if !taskExists || !hasUser || requestingUserSID == "" {
		logger.Error("elevated helper received invalid arguments", "operation", "elevated-task")
		return true, exitInvalidArguments
	}
	logger = logger.With("operation", "elevated-task", "task", taskName)
	if err := requireCurrentUser(requestingUserSID); err != nil {
		logger.Error("elevated helper runs as another Windows user", "error", err)
		return true, exitOtherUser
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
	defer cancel()
	hasSkippedItems, err := task(ctx, os.Args[3:], logger)
	switch {
	case err != nil:
		logger.Error("elevated task failed", "error", err)
		return true, exitFailed
	case hasSkippedItems:
		return true, exitCompletedWithSkippedItems
	default:
		return true, exitCompleted
	}
}

// Run starts this executable as an elevated helper that runs the named task.
// Windows shows its administrator prompt unless this process is already elevated.
func Run(ctx context.Context, taskName string, arguments []string) (hasSkippedItems bool, err error) {
	requestingUserSID, err := currentUserSID()
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(ctx, taskTimeout+30*time.Second)
	defer cancel()
	helperArguments := append([]string{taskArgumentPrefix + taskName, userArgumentPrefix + requestingUserSID}, arguments...)
	exitCode, err := runElevated(ctx, helperArguments)
	if err != nil {
		return false, err
	}
	switch exitCode {
	case exitCompleted:
		return false, nil
	case exitCompletedWithSkippedItems:
		return true, nil
	case exitOtherUser:
		return false, ErrOtherUser
	default:
		return false, fmt.Errorf("elevated task %s exited with code %d", taskName, exitCode)
	}
}
