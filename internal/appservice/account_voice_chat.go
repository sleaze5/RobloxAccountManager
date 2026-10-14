package appservice

import (
	"context"

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

func (service *Service) voiceChatRead(ctx context.Context, accountID, version int64, target **VoiceChat) accountRead {
	return accountRead{"voiceChat", func() error {
		voice, err := service.users.VoiceChat(ctx, accountID, version)
		if err == nil {
			*target = voiceChat(voice)
		}
		return err
	}}
}

// Availability follows the Roblox privacy settings page, which hides the toggle for unverified users
// and disables it while enrollment is closed.
func voiceChat(voice robloxservices.UserVoiceChat) *VoiceChat {
	result := &VoiceChat{
		State: VoiceChatOff, OptedIn: voice.OptedIn, EnrollmentDisabled: voice.OptInDisabled,
		Verified: voice.Verified, CanVerifyAge: voice.CanVerifyAge,
	}
	switch {
	case voice.Suspended:
		result.State = VoiceChatSuspended
		if !voice.SuspendedUntil.IsZero() {
			result.SuspendedUntilMs = voice.SuspendedUntil.UnixMilli()
		}
	case !voice.Verified || (voice.OptInDisabled && !voice.OptedIn):
		result.State = VoiceChatIneligible
	case voice.OptedIn:
		result.State = VoiceChatOn
	}
	return result
}
