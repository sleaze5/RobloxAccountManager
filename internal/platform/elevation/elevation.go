package elevation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	helperArgument = "--elevated-helper"
	// A deep cleanup deletes whole Roblox installations, which can take a while
	// on a slow disk, but a stuck task must not hold a launch forever.
	taskTimeout = 3 * time.Minute
)

var (
	ErrDeclined  = errors.New("administrator rights were declined")
	ErrOtherUser = errors.New("administrator rights were given by another Windows user")
	errStopped   = errors.New("the elevated helper stopped")
)

// Task runs inside the elevated helper process and reports whether it skipped
// items that it could not change.
type Task func(ctx context.Context, arguments []string, logger *slog.Logger) (hasSkippedItems bool, err error)

type responseStatus string

const (
	statusReady                     responseStatus = "ready"
	statusOtherUser                 responseStatus = "other-user"
	statusCompleted                 responseStatus = "completed"
	statusCompletedWithSkippedItems responseStatus = "completed-with-skipped-items"
	statusFailed                    responseStatus = "failed"
)

type taskRequest struct {
	Task      string   `json:"task"`
	Arguments []string `json:"arguments"`
}

type taskResponse struct {
	Status responseStatus `json:"status"`
}

// helperConnection is the application side of a running helper. Requests
// go out through requests, and a goroutine decodes the answers into responses.
// responses closes when the helper exits.
type helperConnection struct {
	requests  io.WriteCloser
	responses <-chan taskResponse
}

// Helper runs tasks in one elevated helper process. Windows shows its
// administrator prompt when the first task starts the helper. Later tasks
// reuse the helper until Close or until the application exits.
type Helper struct {
	mutex      sync.Mutex
	logger     *slog.Logger
	connection *helperConnection
	// finishing belongs to a stopped helper that may still run the task it
	// received last. It closes when that helper exits.
	finishing <-chan taskResponse
}

func NewHelper(logger *slog.Logger) *Helper {
	return &Helper{logger: logger}
}

func (helper *Helper) Run(ctx context.Context, taskName string, arguments []string) (hasSkippedItems bool, err error) {
	helper.mutex.Lock()
	defer helper.mutex.Unlock()
	ctx, cancel := context.WithTimeout(ctx, taskTimeout+30*time.Second)
	defer cancel()
	if err := helper.waitForStoppedLocked(ctx); err != nil {
		return false, err
	}
	encoded, err := json.Marshal(taskRequest{Task: taskName, Arguments: arguments})
	if err != nil {
		return false, fmt.Errorf("encode elevated task: %w", err)
	}
	if err := helper.sendLocked(ctx, append(encoded, '\n')); err != nil {
		return false, err
	}
	select {
	case response, isOpen := <-helper.connection.responses:
		if !isOpen {
			helper.stopLocked()
			return false, errStopped
		}
		switch response.Status {
		case statusCompleted:
			return false, nil
		case statusCompletedWithSkippedItems:
			return true, nil
		default:
			return false, fmt.Errorf("elevated task %s failed", taskName)
		}
	case <-ctx.Done():
		// The helper finishes the task by itself, then exits because its
		// request pipe is closed. The caller holds the launch lock, so waiting
		// here keeps games from starting while the task still deletes files.
		// The helper ends every task within taskTimeout.
		helper.stopLocked()
		stopContext, stopCancel := context.WithTimeout(context.Background(), taskTimeout+30*time.Second)
		defer stopCancel()
		return false, errors.Join(ctx.Err(), helper.waitForStoppedLocked(stopContext))
	}
}

// WaitForStoppedTask waits until a stopped helper has finished its last task.
// A launch calls it, so that a game never starts while Roblox is cleaned.
func (helper *Helper) WaitForStoppedTask(ctx context.Context) error {
	helper.mutex.Lock()
	defer helper.mutex.Unlock()
	return helper.waitForStoppedLocked(ctx)
}

func (helper *Helper) waitForStoppedLocked(ctx context.Context) error {
	for helper.finishing != nil {
		select {
		case _, isOpen := <-helper.finishing:
			if !isOpen {
				helper.finishing = nil
			}
		case <-ctx.Done():
			return fmt.Errorf("wait for the stopped elevated helper: %w", ctx.Err())
		}
	}
	return nil
}

// sendLocked writes a request to the running helper. It starts the helper if
// none runs, or if the previous one exited, for example because it was closed
// in Task Manager.
func (helper *Helper) sendLocked(ctx context.Context, encodedRequest []byte) error {
	if helper.connection != nil {
		if _, err := helper.connection.requests.Write(encodedRequest); err == nil {
			return nil
		}
		helper.stopLocked()
	}
	connection, err := startHelper(ctx)
	if err != nil {
		return err
	}
	helper.connection = connection
	helper.logger.Info("elevated helper started", "operation", "elevated-helper")
	if _, err := connection.requests.Write(encodedRequest); err != nil {
		helper.stopLocked()
		return fmt.Errorf("send task to the elevated helper: %w", err)
	}
	return nil
}

// Close stops the helper. It exits after its current task.
func (helper *Helper) Close() {
	helper.mutex.Lock()
	defer helper.mutex.Unlock()
	helper.stopLocked()
}

func (helper *Helper) stopLocked() {
	if helper.connection == nil {
		return
	}
	_ = helper.connection.requests.Close()
	helper.finishing = helper.connection.responses
	helper.connection = nil
	helper.logger.Info("elevated helper stopped", "operation", "elevated-helper")
}

// HandleHelperMode serves tasks when this process was started as the
// elevated helper. It reports whether it did, and the exit code.
func HandleHelperMode(tasks map[string]Task, logger *slog.Logger) (isHelper bool, exitCode int) {
	if len(os.Args) < 2 || os.Args[1] != helperArgument {
		return false, 0
	}
	logger = logger.With("operation", "elevated-helper")
	requests, responses, err := connectToApplication(os.Args[2:])
	defer closePipes(requests, responses)
	if errors.Is(err, ErrOtherUser) {
		err = errors.Join(err, json.NewEncoder(responses).Encode(taskResponse{Status: statusOtherUser}))
	}
	if err != nil {
		logger.Error("elevated helper could not connect to the application", "error", err)
		return true, 1
	}
	encoder := json.NewEncoder(responses)
	if err := encoder.Encode(taskResponse{Status: statusReady}); err != nil {
		logger.Error("elevated helper could not answer the application", "error", err)
		return true, 1
	}
	decoder := json.NewDecoder(requests)
	for {
		var request taskRequest
		// The application closes the request pipe when it stops the helper or
		// exits, and Windows closes it when the application crashes.
		if err := decoder.Decode(&request); err != nil {
			return true, 0
		}
		if err := encoder.Encode(taskResponse{Status: runTask(tasks, request, logger)}); err != nil {
			logger.Error("elevated helper could not answer the application", "error", err)
			return true, 1
		}
	}
}

func closePipes(requests, responses io.Closer) {
	for _, pipe := range []io.Closer{requests, responses} {
		if pipe != nil {
			// The helper exits right after, and Windows closes the pipe anyway.
			_ = pipe.Close()
		}
	}
}

func runTask(tasks map[string]Task, request taskRequest, logger *slog.Logger) responseStatus {
	task, exists := tasks[request.Task]
	if !exists {
		logger.Error("elevated helper received an unknown task", "task", request.Task)
		return statusFailed
	}
	ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)
	defer cancel()
	hasSkippedItems, err := task(ctx, request.Arguments, logger.With("task", request.Task))
	switch {
	case err != nil:
		logger.Error("elevated task failed", "task", request.Task, "error", err)
		return statusFailed
	case hasSkippedItems:
		return statusCompletedWithSkippedItems
	default:
		return statusCompleted
	}
}
