package appservice

func applyAccountSettingDependencies(snapshot *AccountSettingsSnapshot) {
	for index := range snapshot.Settings {
		view := &snapshot.Settings[index]
		if !view.Editable {
			continue
		}
		switch view.Key {
		case SettingDirectGameChat:
			requireAccountSetting(snapshot, view, SettingGameChat, "AllUsers", "Turn on Game chat before you change Direct chat.")
		case SettingPartyChat, SettingPartyVoice:
			requireAccountSetting(snapshot, view, SettingParty, "AllConnections", "Turn on Chat and party with friends first.")
			if view.Key == SettingPartyVoice {
				capAccountSettingAudience(snapshot, view, SettingPartyChat, "Voice chat with friends cannot include more people than Friends chat.")
			}
		case SettingGameJoins:
			capAccountSettingAudience(snapshot, view, SettingOnlineStatus, "Select a broader Online status audience first.")
		case SettingActivityUpdates:
			parent := findAccountSetting(*snapshot, SettingGameJoins)
			if settingAudienceRank(parent.StringValue) < 2 || !settingKnownValue(parent) || parent.UnavailableReason == conflictingSettingReason {
				view.UnavailableReason = "Show current game must include Friends before sharing activity updates."
			}
		default:
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
		view.UnavailableReason = "Refresh to check the related setting before you change this one."
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

func settingAudienceRank(audience *string) int {
	if audience == nil {
		return -1
	}
	switch *audience {
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
