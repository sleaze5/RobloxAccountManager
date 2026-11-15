package appservice

import (
	"context"
	"strconv"
)

type LaunchConfirmation struct {
	ID                   string `json:"id"`
	ProcessCount         int    `json:"processCount"`
	MultiInstanceEnabled bool   `json:"multiInstanceEnabled"`
	AccountCount         int    `json:"accountCount"`
	MultiInstanceMessage string `json:"multiInstanceMessage"`
	// IsCleaningBeforeLaunch asks a launch of several accounts to skip
	// cleaning, and tells a single launch that Roblox is cleaned first.
	IsCleaningBeforeLaunch bool `json:"isCleaningBeforeLaunch"`
}

type pendingLaunchConfirmation struct {
	state LaunchConfirmation
	reply chan bool
}

func (service *Service) GetLaunchConfirmation() LaunchConfirmation {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.launchConfirmation == nil {
		return LaunchConfirmation{}
	}
	return service.launchConfirmation.state
}

func (service *Service) ResolveLaunchConfirmation(confirmationID string, approved bool) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	pending := service.launchConfirmation
	if pending == nil || pending.state.ID != confirmationID {
		return
	}
	select {
	case pending.reply <- approved:
	default:
	}
}

func (service *Service) cancelLaunchConfirmation() {
	service.ResolveLaunchConfirmation(service.GetLaunchConfirmation().ID, false)
}

// replacementConfirmation asks before running Roblox games close for a launch.
func (service *Service) replacementConfirmation(shouldClean bool) func(context.Context, int, bool) (bool, error) {
	return func(ctx context.Context, count int, multiInstance bool) (bool, error) {
		return service.confirmLaunch(ctx, LaunchConfirmation{ProcessCount: count, MultiInstanceEnabled: multiInstance, IsCleaningBeforeLaunch: shouldClean})
	}
}

func (service *Service) confirmLaunch(ctx context.Context, state LaunchConfirmation) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	service.mutex.Lock()
	service.launchConfirmationNext++
	state.ID = strconv.FormatUint(service.launchConfirmationNext, 10)
	pending := &pendingLaunchConfirmation{
		state: state,
		reply: make(chan bool, 1),
	}
	service.launchConfirmation = pending
	service.mutex.Unlock()
	service.events.launchConfirmationRequested()
	defer func() {
		service.mutex.Lock()
		service.launchConfirmation = nil
		service.mutex.Unlock()
		service.events.launchConfirmationChanged()
	}()
	select {
	case approved := <-pending.reply:
		return approved, ctx.Err()
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
