package appservice

func applyAccountSettingDependencies(snapshot *AccountSettingsSnapshot) {
	for index := range snapshot.Settings {
		view := &snapshot.Settings[index]
		if !view.Editable {
			continue
		}
		switch view.Key {
		case SettingDirectExperienceChat:
			requireAccountSetting(snapshot, view, SettingExperienceChat, "AllUsers", "Enable Experience chat before changing Direct chat.")
		case SettingPartyChat, SettingPartyVoice:
			requireAccountSetting(snapshot, view, SettingParty, "AllConnections", "Enable Chat and party with friends first.")
			if view.Key == SettingPartyVoice {
				capAccountSettingAudience(snapshot, view, SettingPartyChat, "Voice chat with friends cannot include more people than Friends chat.")
			}
		case SettingExperienceJoins:
			capAccountSettingAudience(snapshot, view, SettingOnlineStatus, "Choose a broader Online status audience first.")
		case SettingActivityUpdates:
			parent := findAccountSetting(*snapshot, SettingExperienceJoins)
			if settingAudienceRank(parent.StringValue) < 2 || !settingKnownValue(parent) || parent.UnavailableReason == conflictingSettingReason {
				view.UnavailableReason = "Show current game must include Friends before sharing activity updates."
			}
		}
		finishSettingCapability(view)
	}
}

func requireAccountSetting(snapshot *AccountSettingsSnapshot, view *AccountSettingView, parentKey AccountSettingKey, required, reason string) {
	parent := findAccountSetting(*snapshot, parentKey)
	if !settingKnownValue(parent) || parent.StringValue == nil || *parent.StringValue != required || parent.UnavailableReason == conflictingSettingReason {
		view.UnavailableReason = reason
	}
}

func capAccountSettingAudience(snapshot *AccountSettingsSnapshot, view *AccountSettingView, parentKey AccountSettingKey, reason string) {
	parent := findAccountSetting(*snapshot, parentKey)
	maximum := settingAudienceRank(parent.StringValue)
	if !settingKnownValue(parent) || maximum < 0 || parent.UnavailableReason == conflictingSettingReason {
		view.UnavailableReason = "Refresh settings to check the related audience before editing."
		return
	}
	for index := range view.Options {
		option := &view.Options[index]
		if option.Enabled && settingAudienceRank(option.StringValue) > maximum {
			option.Enabled = false
			option.Reason = reason
		}
	}
}

func settingAudienceRank(value *string) int {
	if value == nil {
		return -1
	}
	switch *value {
	case "All", "AllUsers", "AllAuthenticatedUsers":
		return 5
	case "Followers", "FriendsFollowingAndFollowers":
		return 4
	case "Following", "FriendsAndFollowing":
		return 3
	case "Friends", "AllConnections":
		return 2
	case "TrustedFriends", "TrustedConnectionsOnly":
		return 1
	case "NoOne":
		return 0
	default:
		return -1
	}
}
