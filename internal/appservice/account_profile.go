package appservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

var accountProfileOperationID atomic.Uint64

type AccountProfileSnapshot struct {
	AccountID      int64                            `json:"accountId"`
	FetchedAtMs    int64                            `json:"fetchedAtMs"`
	Robux          *int64                           `json:"robux"`
	PendingRobux   *int64                           `json:"pendingRobux"`
	AgeBracket     *int                             `json:"ageBracket"`
	AgeGroup       string                           `json:"ageGroup"`
	AgeVerified    *bool                            `json:"ageVerified"`
	CountryCode    string                           `json:"countryCode"`
	Premium        *bool                            `json:"premium"`
	Plus           *bool                            `json:"plus"`
	TwoStepEnabled *bool                            `json:"twoStepEnabled"`
	TwoStepMethods []string                         `json:"twoStepMethods"`
	Description    string                           `json:"description"`
	VerifiedBadge  *bool                            `json:"verifiedBadge"`
	FriendCount    *int64                           `json:"friendCount"`
	FollowerCount  *int64                           `json:"followerCount"`
	FollowingCount *int64                           `json:"followingCount"`
	PrimaryGroup   *robloxservices.UserPrimaryGroup `json:"primaryGroup"`
	Unavailable    []string                         `json:"unavailable"`
}

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64) (AccountProfileSnapshot, error) {
	if accountID <= 0 {
		return AccountProfileSnapshot{}, profileError(roblox.KindProtocol, "Select an account first.")
	}
	if !service.vault.Unlocked() {
		return AccountProfileSnapshot{}, profileError(roblox.KindVaultLocked, "The account vault is locked.")
	}
	account, err := service.repo.GetView(ctx, accountID)
	if err != nil {
		return AccountProfileSnapshot{}, mapRepositoryError(err)
	}
	session, err := service.sessions.Snapshot(accountID)
	if err != nil {
		return AccountProfileSnapshot{}, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(session.Context, cancel)
	defer stop()
	if session.Context.Err() != nil {
		cancel()
	}
	logger := service.logger.With("operation_id", fmt.Sprintf("profile-%d", accountProfileOperationID.Add(1)), "operation", "profile-read")
	started := time.Now()
	profile := AccountProfileSnapshot{AccountID: accountID, TwoStepMethods: []string{}, Unavailable: []string{}}
	reads := service.profileReads(ctx, accountID, session.SecretVersion, account.RobloxUserID, &profile)
	failures := make([]error, len(reads))
	var group sync.WaitGroup
	for index, read := range reads {
		group.Go(func() { failures[index] = read.fetch() })
	}
	group.Wait()
	if ctx.Err() != nil || !service.vault.Unlocked() || !service.sessions.VersionCurrent(accountID, session.SecretVersion) {
		return AccountProfileSnapshot{}, profileError(roblox.KindCancelled, "The profile request ended or the account session changed. Refresh to retry.")
	}
	for index, failure := range failures {
		if failure == nil {
			continue
		}
		var remote *roblox.Error
		kind, status := roblox.KindProtocol, 0
		if errors.As(failure, &remote) {
			kind, status = remote.Kind, remote.Status
		}
		logger.Warn("account profile section unavailable", "source", reads[index].source, "error_kind", kind, "status", status)
		if kind == roblox.KindReauthRequired || kind == roblox.KindInvalidSession || kind == roblox.KindVaultLocked {
			return AccountProfileSnapshot{}, failure
		}
		profile.Unavailable = append(profile.Unavailable, reads[index].source)
	}
	profile.FetchedAtMs = time.Now().UnixMilli()
	logger.Debug("account profile read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_sections", len(profile.Unavailable))
	return profile, nil
}

type accountProfileRead struct {
	source string
	fetch  func() error
}

func (service *Service) profileReads(ctx context.Context, accountID, version, userID int64, profile *AccountProfileSnapshot) []accountProfileRead {
	return []accountProfileRead{
		{"account", func() error {
			info, err := service.users.AccountInfo(ctx, accountID, version, userID)
			if err == nil {
				profile.Premium, profile.Plus = info.IsPremium, info.HasRobloxSubscription
				if info.AgeBracket != nil && (*info.AgeBracket == 0 || *info.AgeBracket == 1) {
					profile.AgeBracket = info.AgeBracket
				}
				country := strings.ToUpper(strings.TrimSpace(info.CountryCode))
				if len(country) == 2 && country[0] >= 'A' && country[0] <= 'Z' && country[1] >= 'A' && country[1] <= 'Z' {
					profile.CountryCode = country
				}
			}
			return err
		}},
		{"robux", func() error {
			value, err := service.users.Robux(ctx, accountID, version)
			if err == nil {
				profile.Robux = value
			}
			return err
		}},
		{"pendingRobux", func() error {
			value, err := service.users.PendingRobux(ctx, accountID, version, userID)
			if err == nil {
				profile.PendingRobux = value
			}
			return err
		}},
		{"ageGroup", func() error {
			value, err := service.users.AgeGroup(ctx, accountID, version, userID)
			if err == nil {
				profile.AgeGroup = value
			}
			return err
		}},
		{"ageVerification", func() error {
			value, err := service.users.AgeVerified(ctx, accountID, version)
			if err == nil {
				profile.AgeVerified = value
			}
			return err
		}},
		{"twoStep", func() error {
			methods, err := service.users.TwoStepVerification(ctx, accountID, version, userID)
			if err == nil {
				applyProfileVerification(profile, methods)
			}
			return err
		}},
		{"about", func() error {
			about, err := service.users.About(ctx, accountID, version, userID)
			if err == nil {
				profile.Description, profile.VerifiedBadge = about.Description, &about.VerifiedBadge
			}
			return err
		}},
		{"friends", func() error {
			value, err := service.users.SocialCount(ctx, accountID, version, userID, "friends")
			if err == nil {
				profile.FriendCount = value
			}
			return err
		}},
		{"followers", func() error {
			value, err := service.users.SocialCount(ctx, accountID, version, userID, "followers")
			if err == nil {
				profile.FollowerCount = value
			}
			return err
		}},
		{"following", func() error {
			value, err := service.users.SocialCount(ctx, accountID, version, userID, "followings")
			if err == nil {
				profile.FollowingCount = value
			}
			return err
		}},
		{"primaryGroup", func() error {
			group, err := service.users.PrimaryGroup(ctx, accountID, version, userID)
			if err == nil {
				profile.PrimaryGroup = group
			}
			return err
		}},
	}
}

func applyProfileVerification(profile *AccountProfileSnapshot, methods []robloxservices.UserVerificationMethod) {
	for _, method := range methods {
		if method.Enabled != nil && *method.Enabled && !slices.Contains(profile.TwoStepMethods, method.MediaType) {
			profile.TwoStepMethods = append(profile.TwoStepMethods, method.MediaType)
		}
	}
	profile.TwoStepEnabled = new(len(profile.TwoStepMethods) > 0)
}

func profileError(kind roblox.ErrorKind, message string) error {
	return &roblox.Error{Kind: kind, Endpoint: "account-profile", Message: message}
}
