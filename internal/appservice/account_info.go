package appservice

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

var accountInfoOperationID atomic.Uint64

// AccountInfoSnapshot is never persisted or logged.
type AccountInfoSnapshot struct {
	AccountID        int64           `json:"accountId"`
	Email            string          `json:"email"`
	EmailVerified    *bool           `json:"emailVerified"`
	Phone            string          `json:"phone"`
	PhoneVerified    *bool           `json:"phoneVerified"`
	Gender           string          `json:"gender"`
	Birthdate        string          `json:"birthdate"`
	AgeGroup         string          `json:"ageGroup"`
	AgeVerification  AgeVerification `json:"ageVerification"`
	FirstAccount     *bool           `json:"firstAccount"`
	Language         string          `json:"language"`
	AutoTranslations *bool           `json:"autoTranslations"`
	Location         string          `json:"location"`
	Pin              PinState        `json:"pin"`
	VoiceChat        *VoiceChat      `json:"voiceChat"`
	Unavailable      []string        `json:"unavailable"`
}

func (service *Service) GetAccountInfo(ctx context.Context, accountID int64) (AccountInfoSnapshot, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-info")
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return AccountInfoSnapshot{}, mapRepositoryError(err)
	}
	logger := service.logger.With("operation_id", fmt.Sprintf("account-info-%d", accountInfoOperationID.Add(1)), "operation", "account-info-read")
	started := time.Now()
	snapshot := AccountInfoSnapshot{AccountID: accountID}
	var age accountAge
	reads := append(service.accountInfoReads(ctx, accountID, version, account.RobloxUserID, &snapshot), service.ageReads(ctx, accountID, version, &age)...)
	snapshot.Unavailable, err = service.runAccountReads(ctx, accountID, version, "account-info", logger, reads)
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	snapshot.AgeGroup, snapshot.AgeVerification = age.group, age.verification()
	logger.Debug("account info read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_sections", len(snapshot.Unavailable))
	return snapshot, nil
}

func (service *Service) accountInfoReads(ctx context.Context, accountID, version, userID int64, snapshot *AccountInfoSnapshot) []accountRead {
	return []accountRead{
		{"email", func() error {
			email, err := service.users.Email(ctx, accountID, version)
			if err == nil {
				snapshot.Email, snapshot.EmailVerified = email.Address, &email.Verified
			}
			return err
		}},
		{"phone", func() error {
			phone, err := service.users.Phone(ctx, accountID, version)
			if err == nil {
				snapshot.Phone, snapshot.PhoneVerified = phone.Number, &phone.Verified
			}
			return err
		}},
		{"gender", func() error {
			gender, err := service.users.Gender(ctx, accountID, version)
			if err == nil {
				snapshot.Gender = gender
			}
			return err
		}},
		{"birthdate", func() error {
			date, err := service.users.Birthdate(ctx, accountID, version)
			if err == nil {
				snapshot.Birthdate = fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
			}
			return err
		}},
		{"firstAccount", func() error {
			first, err := service.users.FirstAccount(ctx, accountID, version, userID)
			if err == nil {
				snapshot.FirstAccount = &first
			}
			return err
		}},
		{"locale", func() error {
			locale, err := service.users.Locale(ctx, accountID, version)
			if err == nil {
				snapshot.Language, snapshot.AutoTranslations = locale.Language, locale.AutoTranslations
			}
			return err
		}},
		{"location", func() error {
			location, err := service.users.AccountLocation(ctx, accountID, version)
			if err == nil {
				snapshot.Location = location
			}
			return err
		}},
		{"pin", func() error {
			pin, err := service.users.Pin(ctx, accountID, version)
			if err == nil {
				snapshot.Pin = pinState(pin)
			}
			return err
		}},
		service.voiceChatRead(ctx, accountID, version, &snapshot.VoiceChat),
	}
}
