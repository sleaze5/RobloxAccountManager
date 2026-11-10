package appservice

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

type AccountProfileSnapshot struct {
	AccountID       int64                            `json:"accountId"`
	Robux           *int64                           `json:"robux"`
	PendingRobux    *int64                           `json:"pendingRobux"`
	AgeBracket      *int                             `json:"ageBracket"`
	AgeGroup        string                           `json:"ageGroup"`
	AgeVerification AgeVerification                  `json:"ageVerification"`
	CountryCode     string                           `json:"countryCode"`
	Premium         *bool                            `json:"premium"`
	Plus            *bool                            `json:"plus"`
	TwoStepEnabled  *bool                            `json:"twoStepEnabled"`
	TwoStepMethods  []string                         `json:"twoStepMethods"`
	VoiceChat       *VoiceChat                       `json:"voiceChat"`
	Description     string                           `json:"description"`
	SocialLinks     []robloxservices.UserSocialLink  `json:"socialLinks"`
	PreviousNames   []string                         `json:"previousNames"`
	VerifiedBadge   *bool                            `json:"verifiedBadge"`
	FriendCount     *int64                           `json:"friendCount"`
	FollowerCount   *int64                           `json:"followerCount"`
	FollowingCount  *int64                           `json:"followingCount"`
	PrimaryGroup    *robloxservices.UserPrimaryGroup `json:"primaryGroup"`
	Unavailable     []string                         `json:"unavailable"`
}

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64) (AccountProfileSnapshot, error) {
	if accountID <= 0 {
		return AccountProfileSnapshot{}, profileError(roblox.KindProtocol, "Select an account first.")
	}
	if !service.vault.Unlocked() {
		return AccountProfileSnapshot{}, profileError(roblox.KindVaultLocked, "The account vault is locked.")
	}
	account, err := service.accountRepository.GetView(ctx, accountID)
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
	logger := service.logger.With("operation_id", logging.NewOperationID("profile"), "operation", "profile-read")
	started := time.Now()
	profile := AccountProfileSnapshot{AccountID: accountID, TwoStepMethods: []string{}, Unavailable: []string{}}
	var age accountAge
	reads := append(service.profileReads(ctx, accountID, session.SecretVersion, account.RobloxUserID, &profile), service.ageReads(ctx, accountID, session.SecretVersion, &age)...)
	unavailable, err := service.runAccountReads(ctx, accountID, session.SecretVersion, "account-profile", logger, reads)
	if err != nil {
		return AccountProfileSnapshot{}, err
	}
	profile.Unavailable = unavailable
	profile.AgeGroup, profile.AgeVerification = age.group, age.verification()
	logger.Debug("account profile read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_sections", len(profile.Unavailable))
	return profile, nil
}

func (service *Service) profileReads(ctx context.Context, accountID, version, userID int64, profile *AccountProfileSnapshot) []accountRead {
	return []accountRead{
		{"account", func() error {
			return service.readProfileAccount(ctx, accountID, version, userID, profile)
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
		{"twoStep", func() error {
			methods, err := service.users.TwoStepVerification(ctx, accountID, version, userID)
			if err == nil {
				applyProfileVerification(profile, methods)
			}
			return err
		}},
		service.voiceChatRead(ctx, accountID, version, &profile.VoiceChat),
		{"about", func() error {
			about, err := service.users.About(ctx, accountID, version, userID)
			if err == nil {
				profile.Description, profile.VerifiedBadge = about.Description, &about.VerifiedBadge
			}
			return err
		}},
		{"socialLinks", func() error {
			links, err := service.users.SocialLinks(ctx, accountID, version)
			if err == nil {
				profile.SocialLinks = links
			}
			return err
		}},
		{"previousNames", func() error {
			names, err := service.users.PreviousNames(ctx, accountID, version, userID)
			if err == nil {
				profile.PreviousNames = names
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

func (service *Service) readProfileAccount(ctx context.Context, accountID, version, userID int64, profile *AccountProfileSnapshot) error {
	accountDetails, err := service.users.AccountInfo(ctx, accountID, version, userID)
	if err == nil {
		profile.Premium, profile.Plus = accountDetails.IsPremium, accountDetails.HasRobloxSubscription
		if accountDetails.AgeBracket != nil && (*accountDetails.AgeBracket == 0 || *accountDetails.AgeBracket == 1) {
			profile.AgeBracket = accountDetails.AgeBracket
		}
		country := strings.ToUpper(strings.TrimSpace(accountDetails.CountryCode))
		if len(country) == 2 && country[0] >= 'A' && country[0] <= 'Z' && country[1] >= 'A' && country[1] <= 'Z' {
			profile.CountryCode = country
		}
	}
	return err
}
