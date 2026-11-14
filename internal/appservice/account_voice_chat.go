package appservice

import (
	"context"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

type VoiceChatState string

const (
	VoiceChatOn         VoiceChatState = "on"
	VoiceChatOff        VoiceChatState = "off"
	VoiceChatIneligible VoiceChatState = "ineligible"
	VoiceChatSuspended  VoiceChatState = "suspended"
)

type VoiceChat struct {
	State              VoiceChatState `json:"state"`
	SuspendedUntilMs   int64          `json:"suspendedUntilMs"`
	OptedIn            bool           `json:"optedIn"`
	EnrollmentDisabled bool           `json:"enrollmentDisabled"`
	Verified           bool           `json:"verified"`
	CanVerifyAge       bool           `json:"canVerifyAge"`
}

func (service *Service) voiceChatRead(ctx context.Context, accountID, version, userID int64, target **VoiceChat) accountRead {
	return accountRead{"voiceChat", func() error {
		voice, err := service.users.VoiceChat(ctx, accountID, version)
		if err != nil {
			return err
		}
		*target = voiceChat(voice)
		service.recordVoiceSuspension(ctx, accountID, userID, voiceSuspension(voice))
		return nil
	}}
}

// recordVoiceSuspension caches the voice chat suspension of an account, and
// refreshes the account list when the suspension changed.
func (service *Service) recordVoiceSuspension(ctx context.Context, accountID, userID int64, suspension *accounts.VoiceSuspension) {
	previous := service.users.CachedVoiceSuspension(ctx, userID)
	service.users.SaveVoiceSuspension(ctx, userID, suspension)
	if isSameVoiceSuspension(previous, suspension) {
		return
	}
	view, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		if ctx.Err() == nil {
			service.logger.Warn("account list could not show the voice chat suspension until the next refresh", "operation", "voice-suspension-record", "account_id", accountID, "error", err)
		}
		return
	}
	service.attachModeration(ctx, &view)
	service.attachVoiceSuspension(ctx, &view)
	service.events.send("account:updated", view)
}

// attachVoiceSuspension adds the cached voice chat suspension of an account to
// its view, unless the suspension already ended.
func (service *Service) attachVoiceSuspension(ctx context.Context, view *accounts.AccountView) {
	suspension := service.users.CachedVoiceSuspension(ctx, view.RobloxUserID)
	if suspension != nil && suspension.EndsAtMs != nil && *suspension.EndsAtMs <= time.Now().UnixMilli() {
		return
	}
	view.VoiceSuspension = suspension
}

func voiceSuspension(voice robloxservices.UserVoiceChat) *accounts.VoiceSuspension {
	if !voice.Suspended {
		return nil
	}
	suspension := &accounts.VoiceSuspension{}
	if !voice.SuspendedUntil.IsZero() {
		suspension.EndsAtMs = new(voice.SuspendedUntil.UnixMilli())
	}
	return suspension
}

func isSameVoiceSuspension(previous, next *accounts.VoiceSuspension) bool {
	if previous == nil || next == nil {
		return previous == next
	}
	if previous.EndsAtMs == nil || next.EndsAtMs == nil {
		return previous.EndsAtMs == next.EndsAtMs
	}
	return *previous.EndsAtMs == *next.EndsAtMs
}

// Availability follows the Roblox privacy settings page, which hides the toggle for unverified users
// and disables it while enrollment is closed.
func voiceChat(voice robloxservices.UserVoiceChat) *VoiceChat {
	voiceChat := &VoiceChat{
		State: VoiceChatOff, OptedIn: voice.OptedIn, EnrollmentDisabled: voice.OptInDisabled,
		Verified: voice.Verified, CanVerifyAge: voice.CanVerifyAge,
	}
	switch {
	case voice.Suspended:
		voiceChat.State = VoiceChatSuspended
		if !voice.SuspendedUntil.IsZero() {
			voiceChat.SuspendedUntilMs = voice.SuspendedUntil.UnixMilli()
		}
	case !voice.Verified || (voice.OptInDisabled && !voice.OptedIn):
		voiceChat.State = VoiceChatIneligible
	case voice.OptedIn:
		voiceChat.State = VoiceChatOn
	}
	return voiceChat
}
