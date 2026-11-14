package appservice

import (
	"context"
	"fmt"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// AccountInfoSection names one part of the account info. The frontend reads
// each part on its own so that it shows every part as soon as it arrives.
type AccountInfoSection string

const (
	InfoSectionEmail         AccountInfoSection = "email"
	InfoSectionPhone         AccountInfoSection = "phone"
	InfoSectionGender        AccountInfoSection = "gender"
	InfoSectionBirthdate     AccountInfoSection = "birthdate"
	InfoSectionAge           AccountInfoSection = "age"
	InfoSectionFirstAccount  AccountInfoSection = "firstAccount"
	InfoSectionLocale        AccountInfoSection = "locale"
	InfoSectionLocation      AccountInfoSection = "location"
	InfoSectionPin           AccountInfoSection = "pin"
	InfoSectionVoiceChat     AccountInfoSection = "voiceChat"
	InfoSectionTwoStep       AccountInfoSection = "twoStep"
	InfoSectionRecoveryCodes AccountInfoSection = "recoveryCodes"
	InfoSectionPasskeys      AccountInfoSection = "passkeys"
)

// AccountInfoSnapshot holds the fields of the one section that was read. It
// is never persisted or logged.
type AccountInfoSnapshot struct {
	AccountID        int64              `json:"accountId"`
	Section          AccountInfoSection `json:"section"`
	IsComplete       bool               `json:"isComplete"`
	Email            string             `json:"email"`
	EmailVerified    *bool              `json:"emailVerified"`
	Phone            string             `json:"phone"`
	PhoneVerified    *bool              `json:"phoneVerified"`
	Gender           string             `json:"gender"`
	Birthdate        string             `json:"birthdate"`
	AgeGroup         string             `json:"ageGroup"`
	AgeVerification  AgeVerification    `json:"ageVerification"`
	FirstAccount     *bool              `json:"firstAccount"`
	Language         string             `json:"language"`
	AutoTranslations *bool              `json:"autoTranslations"`
	Location         string             `json:"location"`
	Pin              PinState           `json:"pin"`
	VoiceChat        *VoiceChat         `json:"voiceChat"`
	TwoStepMethods   []string           `json:"twoStepMethods"`
	// RecoveryCodeCount counts the unused recovery codes.
	RecoveryCodeCount        *int64   `json:"recoveryCodeCount"`
	RecoveryCodesCreatedAtMs int64    `json:"recoveryCodesCreatedAtMs"`
	PasskeyNames             []string `json:"passkeyNames"`
}

func (service *Service) GetAccountInfo(ctx context.Context, accountID int64, section AccountInfoSection) (AccountInfoSnapshot, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-info")
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return AccountInfoSnapshot{}, mapRepositoryError(err)
	}
	logger := service.logger.With("operation_id", logging.NewOperationID("account-info"), "operation", "account-info-read", "section", section)
	started := time.Now()
	snapshot := AccountInfoSnapshot{AccountID: accountID, Section: section}
	var age accountAge
	reads := service.accountInfoSectionReads(ctx, accountID, version, account.RobloxUserID, section, &snapshot, &age)
	if len(reads) == 0 {
		return AccountInfoSnapshot{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "account-info", Message: "This account info section does not exist."}
	}
	unavailable, err := service.runAccountReads(ctx, accountID, version, "account-info", logger, reads)
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	snapshot.IsComplete = len(unavailable) == 0
	if section == InfoSectionAge {
		snapshot.AgeGroup, snapshot.AgeVerification = age.group, age.verification()
	}
	logger.Debug("account info section read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_reads", len(unavailable))
	return snapshot, nil
}

func (service *Service) accountInfoSectionReads(ctx context.Context, accountID, version, userID int64, section AccountInfoSection, snapshot *AccountInfoSnapshot, age *accountAge) []accountRead {
	users := service.users
	readsBySection := map[AccountInfoSection]func() []accountRead{
		InfoSectionEmail: func() []accountRead {
			return []accountRead{applyRead("email", func() (robloxservices.UserEmail, error) { return users.Email(ctx, accountID, version) }, func(email robloxservices.UserEmail) {
				snapshot.Email, snapshot.EmailVerified = email.Address, &email.Verified
			})}
		},
		InfoSectionPhone: func() []accountRead {
			return []accountRead{applyRead("phone", func() (robloxservices.UserPhone, error) { return users.Phone(ctx, accountID, version) }, func(phone robloxservices.UserPhone) {
				snapshot.Phone, snapshot.PhoneVerified = phone.Number, &phone.Verified
			})}
		},
		InfoSectionGender: func() []accountRead {
			return []accountRead{applyRead("gender", func() (string, error) { return users.Gender(ctx, accountID, version) }, func(gender string) { snapshot.Gender = gender })}
		},
		InfoSectionBirthdate: func() []accountRead {
			return []accountRead{applyRead("birthdate", func() (robloxservices.UserBirthdate, error) { return users.Birthdate(ctx, accountID, version) }, func(date robloxservices.UserBirthdate) {
				snapshot.Birthdate = fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
			})}
		},
		InfoSectionAge: func() []accountRead { return service.ageReads(ctx, accountID, version, age) },
		InfoSectionFirstAccount: func() []accountRead {
			return []accountRead{applyRead("firstAccount", func() (bool, error) { return users.FirstAccount(ctx, accountID, version, userID) }, func(first bool) { snapshot.FirstAccount = &first })}
		},
		InfoSectionLocale: func() []accountRead {
			return []accountRead{applyRead("locale", func() (robloxservices.UserLocale, error) { return users.Locale(ctx, accountID, version) }, func(locale robloxservices.UserLocale) {
				snapshot.Language, snapshot.AutoTranslations = locale.Language, locale.AutoTranslations
			})}
		},
		InfoSectionLocation: func() []accountRead {
			return []accountRead{applyRead("location", func() (string, error) { return users.AccountLocation(ctx, accountID, version) }, func(location string) { snapshot.Location = location })}
		},
		InfoSectionPin: func() []accountRead {
			return []accountRead{applyRead("pin", func() (robloxservices.AccountPin, error) { return users.Pin(ctx, accountID, version) }, func(pin robloxservices.AccountPin) { snapshot.Pin = pinState(pin) })}
		},
		InfoSectionVoiceChat: func() []accountRead {
			return []accountRead{service.voiceChatRead(ctx, accountID, version, userID, &snapshot.VoiceChat)}
		},
		InfoSectionTwoStep: func() []accountRead {
			return []accountRead{applyRead("twoStep", func() ([]robloxservices.UserVerificationMethod, error) {
				return users.TwoStepVerification(ctx, accountID, version, userID)
			}, func(methods []robloxservices.UserVerificationMethod) {
				snapshot.TwoStepMethods = enabledVerificationMethods(methods)
			})}
		},
		InfoSectionRecoveryCodes: func() []accountRead {
			return []accountRead{applyRead("recoveryCodes", func() (robloxservices.UserRecoveryCodes, error) {
				return users.RecoveryCodes(ctx, accountID, version, userID)
			}, func(codes robloxservices.UserRecoveryCodes) {
				snapshot.RecoveryCodeCount = &codes.ActiveCount
				if !codes.CreatedAt.IsZero() {
					snapshot.RecoveryCodesCreatedAtMs = codes.CreatedAt.UnixMilli()
				}
			})}
		},
		InfoSectionPasskeys: func() []accountRead {
			return []accountRead{applyRead("passkeys", func() ([]string, error) { return users.Passkeys(ctx, accountID, version) }, func(names []string) { snapshot.PasskeyNames = names })}
		},
	}
	sectionReads, ok := readsBySection[section]
	if !ok {
		return nil
	}
	return sectionReads()
}
