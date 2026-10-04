package appservice

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

var accountInfoOperationID atomic.Uint64

// AccountInfoSnapshot holds the personal details shown on the Account info settings page.
// It is never persisted or logged.
type AccountInfoSnapshot struct {
	AccountID        int64           `json:"accountId"`
	FetchedAtMs      int64           `json:"fetchedAtMs"`
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
	// Unavailable lists the sources that failed: email, phone, gender, birthdate, ageGroup, ageVerification, firstAccount, locale, location.
	Unavailable []string `json:"unavailable"`
}

func (service *Service) GetAccountInfo(ctx context.Context, accountID int64) (AccountInfoSnapshot, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "account-info")
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	defer done()
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return AccountInfoSnapshot{}, mapRepositoryError(err)
	}
	logger := service.logger.With("operation_id", fmt.Sprintf("account-info-%d", accountInfoOperationID.Add(1)), "operation", "account-info-read")
	started := time.Now()
	info := AccountInfoSnapshot{AccountID: accountID}
	var age accountAge
	reads := append(service.accountInfoReads(ctx, accountID, version, account.RobloxUserID, &info), service.ageReads(ctx, accountID, version, &age)...)
	info.Unavailable, err = service.runAccountReads(ctx, accountID, version, "account-info", logger, reads)
	if err != nil {
		return AccountInfoSnapshot{}, err
	}
	info.AgeGroup, info.AgeVerification = age.group, age.verification()
	info.FetchedAtMs = time.Now().UnixMilli()
	logger.Debug("account info read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_sections", len(info.Unavailable))
	return info, nil
}

func (service *Service) accountInfoReads(ctx context.Context, accountID, version, userID int64, info *AccountInfoSnapshot) []accountRead {
	return []accountRead{
		{"email", func() error {
			email, err := service.users.Email(ctx, accountID, version)
			if err == nil {
				info.Email, info.EmailVerified = email.Address, &email.Verified
			}
			return err
		}},
		{"phone", func() error {
			phone, err := service.users.Phone(ctx, accountID, version)
			if err == nil {
				info.Phone, info.PhoneVerified = phone.Number, &phone.Verified
			}
			return err
		}},
		{"gender", func() error {
			gender, err := service.users.Gender(ctx, accountID, version)
			if err == nil {
				info.Gender = gender
			}
			return err
		}},
		{"birthdate", func() error {
			date, err := service.users.Birthdate(ctx, accountID, version)
			if err == nil {
				info.Birthdate = fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
			}
			return err
		}},
		{"firstAccount", func() error {
			first, err := service.users.FirstAccount(ctx, accountID, version, userID)
			if err == nil {
				info.FirstAccount = &first
			}
			return err
		}},
		{"locale", func() error {
			locale, err := service.users.Locale(ctx, accountID, version)
			if err == nil {
				info.Language, info.AutoTranslations = locale.Language, locale.AutoTranslations
			}
			return err
		}},
		{"location", func() error {
			location, err := service.users.AccountLocation(ctx, accountID, version)
			if err == nil {
				info.Location = location
			}
			return err
		}},
	}
}
