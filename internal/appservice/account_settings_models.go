package appservice

type AccountSettingKey string

const (
	SettingContentMaturity         AccountSettingKey = "contentAgeRestriction"
	SettingSensitiveIssues         AccountSettingKey = "allowSensitiveIssues"
	SettingExperienceChat          AccountSettingKey = "whoCanChatWithMeInExperiences"
	SettingDirectExperienceChat    AccountSettingKey = "whoCanWhisperChatWithMeInExperiences"
	SettingParty                   AccountSettingKey = "whoCanPartyWithMe"
	SettingPartyChat               AccountSettingKey = "whoCanUsePartyChatWithMe"
	SettingPartyVoice              AccountSettingKey = "whoCanUsePartyVoiceWithMe"
	SettingCamera                  AccountSettingKey = "isAvatarVideoOptIn"
	SettingQuickWords              AccountSettingKey = "allowPresetChat"
	SettingVoiceOptIn              AccountSettingKey = "isUserOptIn"
	SettingVoiceDataUsage          AccountSettingKey = "allowVoiceDataUsage"
	SettingOnlineStatus            AccountSettingKey = "whoCanSeeMyOnlineStatus"
	SettingExperienceJoins         AccountSettingKey = "whoCanJoinMeInExperiences"
	SettingPrivateServerAdditions  AccountSettingKey = "privateServerPrivacy"
	SettingPhoneDiscovery          AccountSettingKey = "phoneNumberDiscoverability"
	SettingContactPermission       AccountSettingKey = "canUploadContacts"
	SettingFriendSuggestions       AccountSettingKey = "friendSuggestions"
	SettingActivityUpdates         AccountSettingKey = "updateFriendsAboutMyActivity"
	SettingTradeAudience           AccountSettingKey = "whoCanTradeWithMe"
	SettingTradeQuality            AccountSettingKey = "tradeQualityFilter"
	SettingInventoryVisibility     AccountSettingKey = "whoCanSeeMyInventory"
	SettingPersonalizedAdvertising AccountSettingKey = "allowPersonalizedAdvertising"
	SettingDataSharing             AccountSettingKey = "allowSellShareData"
	SettingSocialNetworks          AccountSettingKey = "whoCanSeeMySocialNetworks"
)

type AccountSettingChange struct {
	Key                 AccountSettingKey `json:"key"`
	StringValue         *string           `json:"stringValue,omitempty"`
	BoolValue           *bool             `json:"boolValue,omitempty"`
	ExpectedStringValue *string           `json:"expectedStringValue,omitempty"`
	ExpectedBoolValue   *bool             `json:"expectedBoolValue,omitempty"`
}

type AccountSettingOption struct {
	StringValue *string `json:"stringValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
	Label       string  `json:"label"`
	Enabled     bool    `json:"enabled"`
	Reason      string  `json:"reason"`
}

type AccountSettingView struct {
	Key               AccountSettingKey      `json:"key"`
	ValueState        string                 `json:"valueState"`
	StringValue       *string                `json:"stringValue,omitempty"`
	BoolValue         *bool                  `json:"boolValue,omitempty"`
	ValueLabel        string                 `json:"valueLabel"`
	Options           []AccountSettingOption `json:"options"`
	Editable          bool                   `json:"editable"`
	UnavailableReason string                 `json:"unavailableReason"`
}

type AccountSettingsSectionError struct {
	Source  string `json:"source"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type AccountSettingsSnapshot struct {
	AccountID     int64                         `json:"accountId"`
	FetchedAtMs   int64                         `json:"fetchedAtMs"`
	Settings      []AccountSettingView          `json:"settings"`
	SectionErrors []AccountSettingsSectionError `json:"sectionErrors"`
}

type AccountSettingUpdateResult struct {
	Snapshot AccountSettingsSnapshot `json:"snapshot"`
	Outcome  string                  `json:"outcome"`
	Message  string                  `json:"message"`
}

type accountSettingDefinition struct {
	key           AccountSettingKey
	boolean       bool
	verifiedWrite bool
	optionsV2     bool
}

var accountSettingDefinitions = []accountSettingDefinition{
	{key: SettingContentMaturity, verifiedWrite: true},
	{key: SettingSensitiveIssues, verifiedWrite: true},
	{key: SettingExperienceChat, verifiedWrite: true, optionsV2: true},
	{key: SettingDirectExperienceChat, verifiedWrite: true, optionsV2: true},
	{key: SettingParty, verifiedWrite: true, optionsV2: true},
	{key: SettingPartyChat, verifiedWrite: true, optionsV2: true},
	{key: SettingPartyVoice, verifiedWrite: true, optionsV2: true},
	{key: SettingCamera, boolean: true, verifiedWrite: true},
	{key: SettingQuickWords, verifiedWrite: true, optionsV2: true},
	{key: SettingVoiceOptIn, boolean: true, verifiedWrite: true},
	{key: SettingVoiceDataUsage, verifiedWrite: true},
	{key: SettingOnlineStatus, verifiedWrite: true},
	{key: SettingExperienceJoins, verifiedWrite: true},
	{key: SettingPrivateServerAdditions, verifiedWrite: true},
	{key: SettingPhoneDiscovery, verifiedWrite: true},
	{key: SettingContactPermission, boolean: true, verifiedWrite: true},
	{key: SettingFriendSuggestions, verifiedWrite: true},
	{key: SettingActivityUpdates, verifiedWrite: true},
	{key: SettingTradeAudience, verifiedWrite: true},
	{key: SettingTradeQuality, verifiedWrite: true},
	{key: SettingInventoryVisibility, verifiedWrite: true},
	{key: SettingPersonalizedAdvertising, verifiedWrite: true},
	{key: SettingDataSharing, verifiedWrite: true},
	{key: SettingSocialNetworks, verifiedWrite: true},
}

func accountSettingDefinitionFor(key AccountSettingKey) (accountSettingDefinition, bool) {
	for _, definition := range accountSettingDefinitions {
		if definition.key == key {
			return definition, true
		}
	}
	return accountSettingDefinition{}, false
}
