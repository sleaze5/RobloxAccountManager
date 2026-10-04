package services

import (
	"context"
	"strconv"
	"strings"
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
	var result struct {
		EmailAddress *string `json:"emailAddress"`
		Verified     bool    `json:"verified"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-email", "https://accountsettings.roblox.com/v1/email", &result); err != nil {
		return UserEmail{}, err
	}
	if result.EmailAddress != nil && len(*result.EmailAddress) > 320 {
		return UserEmail{}, invalidProfileDetails("account-info-email")
	}
	email := UserEmail{Verified: result.Verified}
	if result.EmailAddress != nil {
		email.Address = strings.TrimSpace(*result.EmailAddress)
	}
	return email, nil
}

func (service *Users) Phone(ctx context.Context, accountID, version int64) (UserPhone, error) {
	var result struct {
		Prefix     *string `json:"prefix"`
		Phone      *string `json:"phone"`
		IsVerified bool    `json:"isVerified"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-phone", "https://accountinformation.roblox.com/v1/phone", &result); err != nil {
		return UserPhone{}, err
	}
	if result.Phone == nil || strings.TrimSpace(*result.Phone) == "" {
		return UserPhone{}, nil
	}
	number := strings.TrimSpace(*result.Phone)
	if result.Prefix != nil && strings.TrimSpace(*result.Prefix) != "" {
		number = "+" + strings.TrimPrefix(strings.TrimSpace(*result.Prefix), "+") + " " + number
	}
	if len(number) > 64 {
		return UserPhone{}, invalidProfileDetails("account-info-phone")
	}
	return UserPhone{Number: number, Verified: result.IsVerified}, nil
}

// Gender returns "male", "female", or "" when the account has not set one.
func (service *Users) Gender(ctx context.Context, accountID, version int64) (string, error) {
	var result struct {
		Gender *int `json:"gender"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-gender", "https://users.roblox.com/v1/gender", &result); err != nil {
		return "", err
	}
	if result.Gender == nil {
		return "", invalidProfileDetails("account-info-gender")
	}
	switch *result.Gender {
	case 2:
		return "male", nil
	case 3:
		return "female", nil
	default:
		return "", nil
	}
}

func (service *Users) Birthdate(ctx context.Context, accountID, version int64) (UserBirthdate, error) {
	var result struct {
		Year  int `json:"birthYear"`
		Month int `json:"birthMonth"`
		Day   int `json:"birthDay"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-birthdate", "https://users.roblox.com/v1/birthdate", &result); err != nil {
		return UserBirthdate{}, err
	}
	if result.Year < 1900 || result.Year > 9999 || result.Month < 1 || result.Month > 12 || result.Day < 1 || result.Day > 31 {
		return UserBirthdate{}, invalidProfileDetails("account-info-birthdate")
	}
	return UserBirthdate{Year: result.Year, Month: result.Month, Day: result.Day}, nil
}

// Labels from Roblox's account settings translations for Label.AgeGroup* keys.
var ageGroupLabels = map[string]string{
	"Label.AgeGroupUnder9": "5–8",
	"Label.AgeGroup9To12":  "9–12",
	"Label.AgeGroup13To15": "13–15",
	"Label.AgeGroup16To17": "16–17",
	"Label.AgeGroup18To20": "18–20",
	"Label.AgeGroup18To24": "18–24",
	"Label.AgeGroupOver21": "21+",
	"Label.AgeGroupOver25": "25+",
}

// CheckedAgeGroup returns the account's age group and whether Roblox checked it with a face scan or an ID.
// The age group can differ from the birthday unless the birthday was verified with an ID.
func (service *Users) CheckedAgeGroup(ctx context.Context, accountID, version int64) (UserAgeGroup, error) {
	var result struct {
		Key     *string `json:"ageGroupTranslationKey"`
		Checked bool    `json:"isChecked"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-age-group", "https://apis.roblox.com/user-settings-api/v1/account-insights/age-group", &result); err != nil {
		return UserAgeGroup{}, err
	}
	if result.Key == nil {
		return UserAgeGroup{}, invalidProfileDetails("account-info-age-group")
	}
	return UserAgeGroup{Label: ageGroupLabels[*result.Key], Checked: result.Checked}, nil
}

func (service *Users) FirstAccount(ctx context.Context, accountID, version, userID int64) (bool, error) {
	var result struct {
		PlayerInfo *struct {
			UserID         string `json:"userId"`
			IsOriginalUser *bool  `json:"isOriginalUser"`
		} `json:"playerInfo"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-first-account", "https://apis.roblox.com/player-hydration-service/v1/players/signed", &result); err != nil {
		return false, err
	}
	if result.PlayerInfo == nil || result.PlayerInfo.IsOriginalUser == nil {
		return false, invalidProfileDetails("account-info-first-account")
	}
	if result.PlayerInfo.UserID != strconv.FormatInt(userID, 10) {
		return false, profileIdentityError()
	}
	return *result.PlayerInfo.IsOriginalUser, nil
}

func (service *Users) Locale(ctx context.Context, accountID, version int64) (UserLocale, error) {
	var result struct {
		GeneralExperience *struct {
			Name string `json:"name"`
		} `json:"generalExperience"`
		ShowRobloxTranslations *bool `json:"showRobloxTranslations"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-locale", "https://locale.roblox.com/v1/locales/user-localization-locus-supported-locales", &result); err != nil {
		return UserLocale{}, err
	}
	locale := UserLocale{AutoTranslations: result.ShowRobloxTranslations}
	if result.GeneralExperience != nil {
		locale.Language = strings.TrimSpace(result.GeneralExperience.Name)
	}
	if len(locale.Language) > 128 {
		return UserLocale{}, invalidProfileDetails("account-info-locale")
	}
	return locale, nil
}

func (service *Users) AccountLocation(ctx context.Context, accountID, version int64) (string, error) {
	var result struct {
		Value *struct {
			LocalizedName        string `json:"localizedName"`
			LocalizedSubdivision string `json:"localizedSubdivision"`
		} `json:"value"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, "account-info-location", "https://accountsettings.roblox.com/v1/account/settings/account-country", &result); err != nil {
		return "", err
	}
	if result.Value == nil {
		return "", nil
	}
	location := strings.TrimSpace(result.Value.LocalizedSubdivision)
	if location == "" {
		location = strings.TrimSpace(result.Value.LocalizedName)
	}
	if len(location) > 256 {
		return "", invalidProfileDetails("account-info-location")
	}
	return location, nil
}
