package roblox

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

const ModerationURL = "https://usermoderation.roblox.com/v1/not-approved"

// Roblox answers most requests of a moderated account with 403 and this
// message. Verified against a banned account on 2026-11-09.
const moderatedMessage = "User is moderated"

func isModeratedResponse(response *Response) bool {
	if response.Status != http.StatusForbidden {
		return false
	}
	var payload struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	return json.Unmarshal(response.Body, &payload) == nil && len(payload.Errors) > 0 && payload.Errors[0].Message == moderatedMessage
}

func moderatedError(endpoint string, status int) *Error {
	return &Error{Kind: KindModerated, Endpoint: endpoint, Status: status, Message: "Roblox moderated this account. Check its status in Accounts."}
}

// maxModerationTextLength bounds the Roblox explanation that the vault keeps.
const maxModerationTextLength = 2000

// DecodeModeration reads the punishment of a moderated account and the
// punished user ID. punishmentTypeDescription is "Warn", "Delete" for a
// termination, or "Ban" and a length, such as "Ban 7 Days". messageToUser
// explains the punishment. Shape verified against a banned account on
// 2026-11-09. A moderated account without a punishment is locked until a human
// verification on the Roblox website. title and body may explain the lock.
func DecodeModeration(body []byte) (accounts.Moderation, int64, bool) {
	var payload struct {
		PunishedUserID            int64  `json:"punishedUserId"`
		PunishmentTypeDescription string `json:"punishmentTypeDescription"`
		BeginDate                 string `json:"beginDate"`
		EndDate                   string `json:"endDate"`
		MessageToUser             string `json:"messageToUser"`
		Title                     string `json:"title"`
		Body                      string `json:"body"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return accounts.Moderation{}, 0, false
	}
	moderation := accounts.Moderation{
		Kind:    accounts.ModerationBanned,
		Title:   moderationText(payload.Title),
		Message: moderationText(cmp.Or(strings.TrimSpace(payload.Body), payload.MessageToUser)),
	}
	switch payload.PunishmentTypeDescription {
	case "":
		moderation.Kind = accounts.ModerationLocked
	case "Warn":
		moderation.Kind = accounts.ModerationWarned
	case "Delete":
		moderation.Kind = accounts.ModerationTerminated
	}
	if begins, err := time.Parse(time.RFC3339, payload.BeginDate); err == nil {
		moderation.BeginsAtMs = begins.UnixMilli()
	}
	if ends, err := time.Parse(time.RFC3339, payload.EndDate); err == nil && moderation.Kind == accounts.ModerationBanned {
		endsAtMs := ends.UnixMilli()
		moderation.EndsAtMs = &endsAtMs
	}
	return moderation, payload.PunishedUserID, true
}

func moderationText(text string) string {
	text = strings.TrimSpace(text)
	if !utf8.ValidString(text) {
		return ""
	}
	if runes := []rune(text); len(runes) > maxModerationTextLength {
		return string(runes[:maxModerationTextLength])
	}
	return text
}

// moderatedCookieValidation reads who a moderated cookie belongs to, because
// Roblox rejects the authenticated-user request of a moderated account.
func (client *Client) moderatedCookieValidation(ctx context.Context, validation CookieValidation) (CookieValidation, error) {
	// The last attempt number turns off transport retries for this one read.
	response, _, err := client.sendCookieValidation(ctx, validation, ModerationURL, 3)
	if err != nil {
		return CookieValidation{}, err
	}
	moderation, userID, found := DecodeModeration(response.Body)
	if response.Status != http.StatusOK || !found || userID <= 0 {
		return CookieValidation{}, &Error{Kind: KindProtocol, Endpoint: "cookie-import", Status: response.Status, Message: "Roblox moderated this account, but its details could not be read. Try again."}
	}
	validation.Identity = accounts.Identity{RobloxUserID: userID}
	validation.Moderation = &moderation
	return validation, nil
}
