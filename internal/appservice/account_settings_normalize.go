package appservice

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const unverifiedSettingReason = "This setting cannot be changed in this app yet."
const conflictingSettingReason = "Roblox returned conflicting values. Refresh settings before editing."

func normalizeAccountSetting(definition accountSettingDefinition, reads accountSettingsReads) (AccountSettingView, bool) {
	plain, optionsRaw, present := accountSettingDocuments(definition.key, reads)
	if definition.optionsV2 {
		v2, found := reads.optionsV2[string(definition.key)]
		plain = nil
		optionsRaw = v2
		present = present || found
	}
	if !present {
		return AccountSettingView{}, false
	}
	var metadata robloxservices.SettingOptions
	metadataOK := json.Unmarshal(optionsRaw, &metadata) == nil && !isNullSetting(optionsRaw)
	current := plain
	if metadataOK && len(metadata.CurrentValue) > 0 {
		current = metadata.CurrentValue
	}
	view := settingValueView(definition, current)
	view.Options = []AccountSettingOption{}
	view.UnavailableReason = settingReadOnlyReason(definition, view)
	if definition.verifiedWrite && !definition.optionsV2 && !settingHasIndependentChoices(definition.key) && reads.optionsErr != nil {
		view.UnavailableReason = "Some settings could not be refreshed. Refresh settings before editing."
	}
	if definition.optionsV2 && reads.optionsV2Err != nil {
		view.UnavailableReason = "Communication permissions could not be refreshed. Refresh settings before editing."
	}
	if metadataOK {
		for _, choice := range metadata.Options {
			value := settingValueView(definition, choice.Option.Value)
			reason := settingOptionReason(definition, value, choice.Requirement)
			if definition.optionsV2 {
				reason = settingRequiredActionsReason(definition, value, choice.RequiredActions)
			}
			view.Options = append(view.Options, AccountSettingOption{
				StringValue: value.StringValue, BoolValue: value.BoolValue, Label: value.ValueLabel,
				Enabled: reason == "", Reason: reason,
			})
		}
	}
	if settingValuesConflict(definition, plain, metadata.CurrentValue) {
		view.ValueState = "unavailable"
		view.UnavailableReason = conflictingSettingReason
	}
	completeSettingOptions(definition, reads, &view)
	slices.SortStableFunc(view.Options, func(a, b AccountSettingOption) int {
		return settingOptionOrder(definition.key, a) - settingOptionOrder(definition.key, b)
	})
	finishSettingCapability(&view)
	return view, true
}

func accountSettingDocuments(settingKey AccountSettingKey, reads accountSettingsReads) (json.RawMessage, json.RawMessage, bool) {
	key := string(settingKey)
	plain, plainPresent := reads.settings[key]
	optionsRaw, optionsPresent := reads.options[key]
	if settingKey == SettingVoiceOptIn || settingKey == SettingCamera {
		plain, plainPresent = reads.voice[key]
		return plain, nil, plainPresent
	}
	return plain, optionsRaw, plainPresent || optionsPresent
}

func isNullSetting(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func settingValueView(definition accountSettingDefinition, raw json.RawMessage) AccountSettingView {
	view := AccountSettingView{Key: definition.key, ValueState: "unavailable", ValueLabel: "Unavailable"}
	if len(raw) == 0 {
		return view
	}
	if isNullSetting(raw) {
		view.ValueState, view.ValueLabel = "unset", "Not set"
		return view
	}
	if definition.boolean {
		var value bool
		if json.Unmarshal(raw, &value) == nil {
			view.ValueState, view.BoolValue, view.ValueLabel = "known", &value, "Off"
			if value {
				view.ValueLabel = "On"
			}
		}
		return view
	}
	var value string
	if json.Unmarshal(raw, &value) == nil && len(value) <= 128 && !strings.ContainsAny(value, "\r\n\x00") {
		view.ValueState, view.StringValue = "known", &value
		view.ValueLabel, _ = settingValueLabel(definition.key, value)
	}
	return view
}

var accountSettingLabels = map[string]string{
	"NoOne": "No one", "Friends": "Friends", "AllUsers": "Everyone", "All": "Everyone",
	"Following": "Following", "Followers": "Followers", "FriendsAndFollowing": "Friends & people I follow",
	"FriendsFollowingAndFollowers": "Friends, followers & people I follow",
	"TrustedFriends":               "Trusted friends", "AllConnections": "All friends", "TrustedConnectionsOnly": "Trusted friends",
	"AllAuthenticatedUsers": "All signed-in users",
	"AllAges":               "Minimal", "NinePlus": "Mild", "ThirteenPlus": "Moderate", "SeventeenPlus": "Restricted", "EighteenPlus": "Restricted",
	"None": "None", "Low": "Low", "Medium": "Medium", "High": "High",
	"Unset": "Unset", "NotDiscoverable": "Not discoverable", "Discoverable": "Discoverable",
	"Enabled": "On", "Disabled": "Off", "Yes": "On", "No": "Off", "Default": "Default",
}

func settingValueLabel(key AccountSettingKey, value string) (string, bool) {
	if !supportedSettingToken(key, value) {
		return value + " (unrecognized value)", false
	}
	if key == SettingExperienceChat || key == SettingDirectExperienceChat || key == SettingParty {
		if value == "NoOne" {
			return "Off", true
		}
		return "On", true
	}
	if key == SettingExperienceJoins || key == SettingTradeAudience {
		if value == "Followers" {
			return "Friends, followers & people I follow", true
		}
		if value == "Following" {
			return "Friends & people I follow", true
		}
	}
	label, known := accountSettingLabels[value]
	if !known {
		return value + " (unrecognized value)", false
	}
	return label, true
}

func supportedSettingToken(key AccountSettingKey, value string) bool {
	var tokens []string
	switch key {
	case SettingContentMaturity:
		tokens = []string{"AllAges", "NinePlus", "ThirteenPlus", "SeventeenPlus", "EighteenPlus"}
	case SettingParty:
		tokens = []string{"AllConnections", "NoOne"}
	case SettingPartyChat, SettingPartyVoice:
		tokens = []string{"AllConnections", "TrustedConnectionsOnly", "NoOne"}
	case SettingExperienceChat, SettingDirectExperienceChat:
		tokens = []string{"NoOne", "AllUsers"}
	case SettingTradeQuality:
		tokens = []string{"None", "Low", "Medium", "High"}
	case SettingPhoneDiscovery:
		tokens = []string{"Unset", "NotDiscoverable", "Discoverable"}
	case SettingSensitiveIssues, SettingVoiceDataUsage, SettingFriendSuggestions, SettingPersonalizedAdvertising, SettingDataSharing, SettingQuickWords:
		tokens = []string{"Enabled", "Disabled"}
	case SettingActivityUpdates:
		tokens = []string{"Yes", "No"}
	case SettingExperienceJoins:
		tokens = []string{"All", "Friends", "NoOne", "Following", "Followers", "FriendsAndFollowing", "FriendsFollowingAndFollowers", "TrustedFriends"}
	case SettingTradeAudience:
		tokens = []string{"All", "Friends", "NoOne", "Following", "Followers", "FriendsAndFollowing", "FriendsFollowingAndFollowers", "AllUsers"}
	case SettingPrivateServerAdditions:
		tokens = []string{"NoOne", "Friends", "FriendsAndFollowing", "FriendsFollowingAndFollowers", "AllUsers", "Default"}
	default:
		tokens = []string{"NoOne", "Friends", "FriendsAndFollowing", "FriendsFollowingAndFollowers", "AllUsers", "TrustedFriends"}
	}
	return slices.Contains(tokens, value)
}

func settingKnownValue(view AccountSettingView) bool {
	if view.ValueState != "known" {
		return false
	}
	if view.BoolValue != nil {
		return true
	}
	if view.StringValue == nil {
		return false
	}
	_, known := settingValueLabel(view.Key, *view.StringValue)
	return known
}

func settingReadOnlyReason(definition accountSettingDefinition, view AccountSettingView) string {
	if !definition.verifiedWrite {
		return unverifiedSettingReason
	}
	if !settingKnownValue(view) {
		return "The current value is unavailable or not supported for editing."
	}
	return ""
}

func settingOptionReason(definition accountSettingDefinition, value AccountSettingView, requirement *string) string {
	if requirement != nil {
		switch *requirement {
		case "Inherited":
			return inheritedSettingReason(definition.key)
		case "ParentConsentInherited", "ParentalConsent":
			return "A linked parent must approve this change on Roblox."
		case "ContentAgeRestrictionVerification":
			return "Complete age verification on Roblox to change this setting."
		}
	}
	if !definition.verifiedWrite {
		return unverifiedSettingReason
	}
	if !settingKnownValue(value) {
		return "This option is not supported for editing."
	}
	if requirement != nil && *requirement != "None" && *requirement != "SelfUpdateSetting" {
		return "Roblox has not confirmed that this option can be changed here."
	}
	return ""
}

// Options without a known order return 10, so they sort last.
func settingOptionOrder(key AccountSettingKey, option AccountSettingOption) int {
	if option.StringValue == nil {
		return 10
	}
	if rank := settingAudienceRank(option.StringValue); rank >= 0 {
		return 5 - rank
	}
	values := []string{"None", "Low", "Medium", "High"}
	if key == SettingContentMaturity {
		values = []string{"AllAges", "NinePlus", "ThirteenPlus", "SeventeenPlus", "EighteenPlus"}
	}
	for index, value := range values {
		if *option.StringValue == value {
			return index
		}
	}
	return 10
}

func settingRequiredActionsReason(definition accountSettingDefinition, value AccountSettingView, raw json.RawMessage) string {
	var actions []string
	if len(raw) == 0 || isNullSetting(raw) || json.Unmarshal(raw, &actions) != nil {
		return "Roblox did not provide change permissions for this option."
	}
	for _, action := range actions {
		switch action {
		case "Inherited":
			return inheritedSettingReason(definition.key)
		case "ParentalConsent", "ParentConsentInherited", "VpcForFae":
			return "A linked parent must approve this change on Roblox."
		case "FacialAgeEstimation", "IdVerification", "ContentAgeRestrictionVerification", "AgeCheckPending":
			return "Complete the required age check on Roblox to change this setting."
		default:
			return "Roblox requires an additional step before this option can be changed."
		}
	}
	requirement := "None"
	return settingOptionReason(definition, value, &requirement)
}

func inheritedSettingReason(key AccountSettingKey) string {
	switch key {
	case SettingExperienceJoins:
		return "Choose a broader Online status audience first."
	case SettingActivityUpdates:
		return "Show current game must include Friends before sharing activity updates."
	case SettingDirectExperienceChat:
		return "Enable Experience chat before changing Direct chat."
	case SettingPartyChat:
		return "Enable Chat and party with friends first."
	case SettingPartyVoice:
		return "Voice chat with friends cannot include more people than Friends chat."
	default:
		return "This setting is inherited and cannot be changed here."
	}
}

func sameSettingValue(a, b AccountSettingView) bool {
	if a.ValueState != b.ValueState {
		return false
	}
	if a.StringValue != nil && b.StringValue != nil {
		return *a.StringValue == *b.StringValue
	}
	return a.BoolValue != nil && b.BoolValue != nil && *a.BoolValue == *b.BoolValue
}

func settingValuesConflict(definition accountSettingDefinition, plain, options json.RawMessage) bool {
	if len(plain) == 0 || len(options) == 0 || isNullSetting(plain) || isNullSetting(options) {
		return false
	}
	left, right := settingValueView(definition, plain), settingValueView(definition, options)
	if definition.key == SettingTradeAudience || definition.key == SettingExperienceJoins {
		if rank := settingAudienceRank(left.StringValue); rank >= 0 && rank == settingAudienceRank(right.StringValue) {
			return false
		}
	}
	return !sameSettingValue(left, right)
}

func settingHasIndependentChoices(key AccountSettingKey) bool {
	return key == SettingSensitiveIssues || key == SettingPersonalizedAdvertising || key == SettingDataSharing || key == SettingContactPermission || key == SettingVoiceOptIn || key == SettingCamera
}

func completeSettingOptions(definition accountSettingDefinition, reads accountSettingsReads, view *AccountSettingView) {
	if !settingKnownValue(*view) || view.UnavailableReason != "" {
		return
	}
	if definition.key == SettingVoiceOptIn || definition.key == SettingCamera {
		view.UnavailableReason = ""
		view.Options = nil
		for _, value := range []bool{true, false} {
			reason := voiceOptionReason(definition.key, value, reads.voice)
			label := "Off"
			if value {
				label = "On"
			}
			view.Options = append(view.Options, AccountSettingOption{BoolValue: &value, Label: label, Enabled: reason == "", Reason: reason})
		}
		return
	}
	if len(view.Options) != 0 || !settingHasIndependentChoices(definition.key) {
		return
	}
	view.UnavailableReason = ""
	if definition.boolean {
		view.Options = []AccountSettingOption{
			{BoolValue: new(true), Label: "On", Enabled: true},
			{BoolValue: new(false), Label: "Off", Enabled: true},
		}
		return
	}
	tokens := []string{"Enabled", "Disabled"}
	for _, value := range tokens {
		label, _ := settingValueLabel(definition.key, value)
		valCopy := value
		view.Options = append(view.Options, AccountSettingOption{StringValue: &valCopy, Label: label, Enabled: true})
	}
}

func voiceOptionReason(key AccountSettingKey, desired bool, voice robloxservices.SettingsDocument) string {
	if !desired {
		return ""
	}
	eligible, disabled := "isUserEligible", "isOptInDisabled"
	if key == SettingCamera {
		eligible, disabled = "isAvatarVideoEligible", "isAvatarVideoOptInDisabled"
	}
	if !settingBool(voice[eligible]) {
		return "This account is not eligible to enable this setting."
	}
	if settingBool(voice[disabled]) {
		return "Roblox has disabled opt-in for this account."
	}
	if key == SettingVoiceOptIn && settingBool(voice["isBanned"]) {
		return "Voice is suspended for this account."
	}
	return ""
}

func settingBool(raw json.RawMessage) bool {
	var value bool
	return json.Unmarshal(raw, &value) == nil && value
}

func finishSettingCapability(view *AccountSettingView) {
	currentReturned, alternative := false, false
	for _, option := range view.Options {
		value := AccountSettingView{ValueState: "known", StringValue: option.StringValue, BoolValue: option.BoolValue}
		if sameSettingValue(*view, value) {
			currentReturned = true
		} else if option.Enabled {
			alternative = true
		}
	}
	if view.UnavailableReason == "" && !currentReturned {
		view.UnavailableReason = "The current value is not among Roblox's available choices."
	}
	if view.UnavailableReason == "" && !alternative {
		view.UnavailableReason = "Roblox has not provided an alternative that can be selected here."
		for _, option := range view.Options {
			value := AccountSettingView{ValueState: "known", StringValue: option.StringValue, BoolValue: option.BoolValue}
			if !sameSettingValue(*view, value) && option.Reason != "" {
				view.UnavailableReason = option.Reason
				break
			}
		}
	}
	view.Editable = view.UnavailableReason == ""
	if !view.Editable {
		for index := range view.Options {
			view.Options[index].Enabled = false
			if view.Options[index].Reason == "" {
				view.Options[index].Reason = view.UnavailableReason
			}
		}
	}
}
