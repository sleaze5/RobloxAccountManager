package appservice

import (
	"context"
	"fmt"
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
		voice, err := service.readVoiceChat(ctx, accountID, version, userID)
		if err == nil {
			*target = voiceChat(voice)
		}
		return err
	}}
}

func (service *Service) readVoiceChat(ctx context.Context, accountID, version, userID int64) (robloxservices.UserVoiceChat, error) {
	voice, err := service.users.VoiceChat(ctx, accountID, version)
	if err != nil {
		return robloxservices.UserVoiceChat{}, err
	}
	service.recordVoiceSuspension(ctx, accountID, userID, voiceSuspension(voice))
	return voice, nil
}

// refreshOnlineVoiceSuspensions reads the voice chat status of each online
// account in the background. A suspension that has not ended yet is read again
// only when the user opens the account's details.
func (service *Service) refreshOnlineVoiceSuspensions(ctx context.Context, targets []robloxservices.PresenceTarget, presences []robloxservices.UserPresence) {
	accountIDByRobloxUserID := make(map[int64]int64, len(targets))
	for _, target := range targets {
		accountIDByRobloxUserID[target.RobloxUserID] = target.AccountID
	}
	nowMs := time.Now().UnixMilli()
	for _, presence := range presences {
		accountID, found := accountIDByRobloxUserID[presence.UserID]
		if !found || presence.UserPresenceType == robloxservices.PresenceTypeOffline {
			continue
		}
		suspension := service.users.CachedVoiceSuspension(ctx, presence.UserID)
		if suspension != nil && (suspension.EndsAtMs == nil || *suspension.EndsAtMs > nowMs) {
			continue
		}
		service.refreshAccountVoiceSuspension(accountID, presence.UserID)
	}
}

func (service *Service) refreshAccountVoiceSuspension(accountID, userID int64) {
	service.refreshMetadata(fmt.Sprintf("voice-suspension-%d", userID), func(parent context.Context) {
		ctx, version, done, err := service.beginAccountSession(parent, accountID, "voice-suspension-refresh")
		if err != nil {
			return
		}
		defer done()
		if _, err := service.readVoiceChat(ctx, accountID, version, userID); err != nil && ctx.Err() == nil {
			service.logger.WarnContext(ctx, "voice chat suspension could not be refreshed; retrying with the next presence update", "operation", "voice-suspension-refresh", "account_id", accountID, "error", err)
		}
	})
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
// its view. A suspension past its end stays until a new read shows that Roblox
// lifted it.
func (service *Service) attachVoiceSuspension(ctx context.Context, view *accounts.AccountView) {
	view.VoiceSuspension = service.users.CachedVoiceSuspension(ctx, view.RobloxUserID)
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
