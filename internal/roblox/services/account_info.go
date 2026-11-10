package services

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Shapes verified against signed-in responses on 2026-10-04.

type UserEmail struct {
	Address  string
	Verified bool
}

type UserPhone struct {
	Number   string
	Verified bool
}

type UserBirthdate struct {
	Year, Month, Day int
}

type UserAgeGroup struct {
	Label   string
	Checked bool
}

type UserLocale struct {
	Language         string
	AutoTranslations *bool
}

func (service *Users) Email(ctx context.Context, accountID, version int64) (UserEmail, error) {
	var payload struct {
		EmailAddress *string `json:"emailAddress"`
		Verified     bool    `json:"verified"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-email", "https://accountsettings.roblox.com/v1/email", &payload); err != nil {
		return UserEmail{}, err
	}
	// 320 bytes is the longest valid email address.
	if payload.EmailAddress != nil && len(*payload.EmailAddress) > 320 {
		return UserEmail{}, invalidProfileDetails("account-info-email")
	}
	email := UserEmail{Verified: payload.Verified}
	if payload.EmailAddress != nil {
		email.Address = strings.TrimSpace(*payload.EmailAddress)
	}
	return email, nil
}

func (service *Users) Phone(ctx context.Context, accountID, version int64) (UserPhone, error) {
	var payload struct {
		Prefix     *string `json:"prefix"`
		Phone      *string `json:"phone"`
		IsVerified bool    `json:"isVerified"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-phone", "https://accountinformation.roblox.com/v1/phone", &payload); err != nil {
		return UserPhone{}, err
	}
	if payload.Phone == nil || strings.TrimSpace(*payload.Phone) == "" {
		return UserPhone{}, nil
	}
	number := strings.TrimSpace(*payload.Phone)
	if payload.Prefix != nil && strings.TrimSpace(*payload.Prefix) != "" {
		number = "+" + strings.TrimPrefix(strings.TrimSpace(*payload.Prefix), "+") + " " + number
	}
	if len(number) > 64 {
		return UserPhone{}, invalidProfileDetails("account-info-phone")
	}
	return UserPhone{Number: number, Verified: payload.IsVerified}, nil
}

func (service *Users) Gender(ctx context.Context, accountID, version int64) (string, error) {
	var payload struct {
		Gender *int `json:"gender"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-gender", "https://users.roblox.com/v1/gender", &payload); err != nil {
		return "", err
	}
	if payload.Gender == nil {
		return "", invalidProfileDetails("account-info-gender")
	}
	switch *payload.Gender {
	case 2:
		return "male", nil
	case 3:
		return "female", nil
	default:
		return "", nil
	}
}

func (service *Users) Birthdate(ctx context.Context, accountID, version int64) (UserBirthdate, error) {
	var payload struct {
		Year  int `json:"birthYear"`
		Month int `json:"birthMonth"`
		Day   int `json:"birthDay"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-birthdate", "https://users.roblox.com/v1/birthdate", &payload); err != nil {
		return UserBirthdate{}, err
	}
	if payload.Year < 1900 || payload.Year > 9999 || payload.Month < 1 || payload.Month > 12 || payload.Day < 1 || payload.Day > 31 {
		return UserBirthdate{}, invalidProfileDetails("account-info-birthdate")
	}
	return UserBirthdate{Year: payload.Year, Month: payload.Month, Day: payload.Day}, nil
}

// Labels from Roblox's account settings translations for Label.AgeGroup* keys.
var ageGroupLabels = map[string]string{
	"Label.AgeGroupUnder9": "5-8",
	"Label.AgeGroup9To12":  "9-12",
	"Label.AgeGroup13To15": "13-15",
	"Label.AgeGroup16To17": "16-17",
	"Label.AgeGroup18To20": "18-20",
	"Label.AgeGroup18To24": "18-24",
	"Label.AgeGroupOver21": "21+",
	"Label.AgeGroupOver25": "25+",
}

// The age group can differ from the birthday unless the birthday was verified with an ID.
func (service *Users) CheckedAgeGroup(ctx context.Context, accountID, version int64) (UserAgeGroup, error) {
	var payload struct {
		Key     *string `json:"ageGroupTranslationKey"`
		Checked bool    `json:"isChecked"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-age-group", "https://apis.roblox.com/user-settings-api/v1/account-insights/age-group", &payload); err != nil {
		return UserAgeGroup{}, err
	}
	if payload.Key == nil {
		return UserAgeGroup{}, invalidProfileDetails("account-info-age-group")
	}
	return UserAgeGroup{Label: ageGroupLabels[*payload.Key], Checked: payload.Checked}, nil
}

func (service *Users) FirstAccount(ctx context.Context, accountID, version, userID int64) (bool, error) {
	var payload struct {
		PlayerInfo *struct {
			UserID         string `json:"userId"`
			IsOriginalUser *bool  `json:"isOriginalUser"`
		} `json:"playerInfo"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-first-account", "https://apis.roblox.com/player-hydration-service/v1/players/signed", &payload); err != nil {
		return false, err
	}
	if payload.PlayerInfo == nil || payload.PlayerInfo.IsOriginalUser == nil {
		return false, invalidProfileDetails("account-info-first-account")
	}
	if payload.PlayerInfo.UserID != strconv.FormatInt(userID, 10) {
		return false, profileIdentityError()
	}
	return *payload.PlayerInfo.IsOriginalUser, nil
}

func (service *Users) Locale(ctx context.Context, accountID, version int64) (UserLocale, error) {
	var payload struct {
		GeneralExperience *struct {
			Name string `json:"name"`
		} `json:"generalExperience"`
		ShowRobloxTranslations *bool `json:"showRobloxTranslations"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-locale", "https://locale.roblox.com/v1/locales/user-localization-locus-supported-locales", &payload); err != nil {
		return UserLocale{}, err
	}
	locale := UserLocale{AutoTranslations: payload.ShowRobloxTranslations}
	if payload.GeneralExperience != nil {
		locale.Language = strings.TrimSpace(payload.GeneralExperience.Name)
	}
	if len(locale.Language) > 128 {
		return UserLocale{}, invalidProfileDetails("account-info-locale")
	}
	return locale, nil
}

func (service *Users) AccountLocation(ctx context.Context, accountID, version int64) (string, error) {
	var payload struct {
		Value *struct {
			LocalizedName        string `json:"localizedName"`
			LocalizedSubdivision string `json:"localizedSubdivision"`
		} `json:"value"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-location", "https://accountsettings.roblox.com/v1/account/settings/account-country", &payload); err != nil {
		return "", err
	}
	if payload.Value == nil {
		return "", nil
	}
	location := strings.TrimSpace(payload.Value.LocalizedSubdivision)
	if location == "" {
		location = strings.TrimSpace(payload.Value.LocalizedName)
	}
	if len(location) > 256 {
		return "", invalidProfileDetails("account-info-location")
	}
	return location, nil
}

type UserRecoveryCodes struct {
	ActiveCount int64
	CreatedAt   time.Time
}

// RecoveryCodes follows the Roblox two-step verification API schema. Roblox
// sends no creation date when the account never generated codes.
func (service *Users) RecoveryCodes(ctx context.Context, accountID, version, userID int64) (UserRecoveryCodes, error) {
	var payload struct {
		ActiveCount *int64  `json:"activeCount"`
		Created     *string `json:"created"`
	}
	address := "https://twostepverification.roblox.com/v1/users/" + strconv.FormatInt(userID, 10) + "/recovery-codes"
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-recovery-codes", address, &payload); err != nil {
		return UserRecoveryCodes{}, err
	}
	if payload.ActiveCount == nil || *payload.ActiveCount < 0 {
		return UserRecoveryCodes{}, invalidProfileDetails("account-info-recovery-codes")
	}
	codes := UserRecoveryCodes{ActiveCount: *payload.ActiveCount}
	if payload.Created != nil {
		if createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(*payload.Created)); err == nil {
			codes.CreatedAt = createdAt.UTC()
		}
	}
	return codes, nil
}

// Passkeys lists the names of the account's passkeys. Roblox lists them with a
// POST request that changes nothing. Shape from the Roblox auth API schema.
func (service *Users) Passkeys(ctx context.Context, accountID, version int64) ([]string, error) {
	var payload struct {
		Credentials *[]struct {
			Nickname string `json:"nickname"`
		} `json:"credentials"`
	}
	if err := service.postRead(ctx, accountID, version, "account-info-passkeys", "https://auth.roblox.com/v1/passkey/ListCredentials", map[string]bool{"all": true}, &payload); err != nil {
		return nil, err
	}
	if payload.Credentials == nil || len(*payload.Credentials) > 100 {
		return nil, invalidProfileDetails("account-info-passkeys")
	}
	names := make([]string, 0, len(*payload.Credentials))
	for _, credential := range *payload.Credentials {
		name := strings.TrimSpace(credential.Nickname)
		if len(name) > 128 {
			return nil, invalidProfileDetails("account-info-passkeys")
		}
		names = append(names, name)
	}
	return names, nil
}
