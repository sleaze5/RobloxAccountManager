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
	key       AccountSettingKey
	boolean   bool
	optionsV2 bool
}

var accountSettingDefinitions = []accountSettingDefinition{
	{key: SettingContentMaturity},
	{key: SettingSensitiveIssues},
	{key: SettingExperienceChat, optionsV2: true},
	{key: SettingDirectExperienceChat, optionsV2: true},
	{key: SettingParty, optionsV2: true},
	{key: SettingPartyChat, optionsV2: true},
	{key: SettingPartyVoice, optionsV2: true},
	{key: SettingCamera, boolean: true},
	{key: SettingQuickWords, optionsV2: true},
	{key: SettingVoiceOptIn, boolean: true},
	{key: SettingVoiceDataUsage},
	{key: SettingOnlineStatus},
	{key: SettingExperienceJoins},
	{key: SettingPrivateServerAdditions},
	{key: SettingPhoneDiscovery},
	{key: SettingContactPermission, boolean: true},
	{key: SettingFriendSuggestions},
	{key: SettingActivityUpdates},
	{key: SettingTradeAudience},
	{key: SettingTradeQuality},
	{key: SettingInventoryVisibility},
	{key: SettingPersonalizedAdvertising},
	{key: SettingDataSharing},
	{key: SettingSocialNetworks},
}

func accountSettingDefinitionFor(key AccountSettingKey) (accountSettingDefinition, bool) {
	for _, definition := range accountSettingDefinitions {
		if definition.key == key {
			return definition, true
		}
	}
	return accountSettingDefinition{}, false
}
