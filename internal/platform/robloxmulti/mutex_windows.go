package robloxmulti

import (
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

const supported = true

type mutexHolder struct {
	stop   windows.Handle
	done   chan struct{}
	owned  atomic.Bool
	failed atomic.Bool
}

func newMutexHolder(logger *slog.Logger) (*mutexHolder, error) {
	stop, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("create mutex stop event: %w", err)
	}
	holder := &mutexHolder{stop: stop, done: make(chan struct{})}
	ready := make(chan error, 1)
	go holder.run(ready, logger)
	if err := <-ready; err != nil {
		<-holder.done
		windows.CloseHandle(stop)
		return nil, err
	}
	return holder, nil
}

func (holder *mutexHolder) run(ready chan<- error, logger *slog.Logger) {
	// Windows mutex ownership belongs to a thread, not a goroutine. Acquisition,
	// waiting, and release must all happen on this same live OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(holder.done)
	mutex, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr("ROBLOX_singletonMutex"))
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		ready <- fmt.Errorf("open Roblox singleton mutex: %w", err)
		return
	}
	defer windows.CloseHandle(mutex)
	result, err := windows.WaitForSingleObject(mutex, 0)
	if err != nil {
		ready <- fmt.Errorf("acquire Roblox singleton mutex: %w", err)
		return
	}
	holder.owned.Store(result == windows.WAIT_OBJECT_0 || result == windows.WAIT_ABANDONED)
	ready <- nil
	if !holder.owned.Load() {
		logger.Info("waiting for Roblox singleton mutex ownership")
		result, err = windows.WaitForMultipleObjects([]windows.Handle{holder.stop, mutex}, false, windows.INFINITE)
		if err != nil {
			holder.failed.Store(true)
			logger.Warn("waiting for Roblox singleton mutex failed", "error", err)
			return
		}
		if result == windows.WAIT_OBJECT_0 {
			return
		}
		if result != windows.WAIT_OBJECT_0+1 && result != windows.WAIT_ABANDONED+1 {
			holder.failed.Store(true)
			logger.Warn("unexpected Roblox mutex wait result", "result", result)
			return
		}
		holder.owned.Store(true)
	}
	logger.Info("holding Roblox singleton mutex")
	if _, err := windows.WaitForSingleObject(holder.stop, windows.INFINITE); err != nil {
		holder.failed.Store(true)
		logger.Warn("waiting for multi-instance shutdown failed", "error", err)
	}
	holder.owned.Store(false)
	if err := windows.ReleaseMutex(mutex); err != nil {
		logger.Warn("releasing Roblox singleton mutex failed", "error", err)
	}
}

func (holder *mutexHolder) close() {
	windows.SetEvent(holder.stop)
	<-holder.done
	windows.CloseHandle(holder.stop)
}
