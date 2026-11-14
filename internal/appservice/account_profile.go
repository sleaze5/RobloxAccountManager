package appservice

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/logging"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// AccountProfileSection names one part of the profile. The frontend reads
// each part on its own so that it shows every part as soon as it arrives.
type AccountProfileSection string

const (
	ProfileSectionAccount       AccountProfileSection = "account"
	ProfileSectionRobux         AccountProfileSection = "robux"
	ProfileSectionTransactions  AccountProfileSection = "transactions"
	ProfileSectionAge           AccountProfileSection = "age"
	ProfileSectionTwoStep       AccountProfileSection = "twoStep"
	ProfileSectionVoiceChat     AccountProfileSection = "voiceChat"
	ProfileSectionAbout         AccountProfileSection = "about"
	ProfileSectionSocialLinks   AccountProfileSection = "socialLinks"
	ProfileSectionPreviousNames AccountProfileSection = "previousNames"
	ProfileSectionFriends       AccountProfileSection = "friends"
	ProfileSectionFollowers     AccountProfileSection = "followers"
	ProfileSectionFollowing     AccountProfileSection = "following"
	ProfileSectionPrimaryGroup  AccountProfileSection = "primaryGroup"
	ProfileSectionCommunities   AccountProfileSection = "communities"
	ProfileSectionRobloxBadges  AccountProfileSection = "robloxBadges"
	ProfileSectionCollectibles  AccountProfileSection = "collectibles"
	ProfileSectionActivity      AccountProfileSection = "activity"
	ProfileSectionPremium       AccountProfileSection = "premium"
	ProfileSectionTrading       AccountProfileSection = "trading"
	ProfileSectionConnections   AccountProfileSection = "connections"
)

type PremiumSubscription struct {
	Name         string `json:"name"`
	RobuxStipend int64  `json:"robuxStipend"`
	RenewsAtMs   int64  `json:"renewsAtMs"`
	ExpiresAtMs  int64  `json:"expiresAtMs"`
}

type TradeEligibility struct {
	CanTrade    bool                           `json:"canTrade"`
	Eligibility string                         `json:"eligibility"`
	FreeTrades  *robloxservices.UserFreeTrades `json:"freeTrades"`
}

// AccountProfileSnapshot holds the fields of the one section that was read.
type AccountProfileSnapshot struct {
	AccountID                          int64                            `json:"accountId"`
	Section                            AccountProfileSection            `json:"section"`
	IsComplete                         bool                             `json:"isComplete"`
	Robux                              *int64                           `json:"robux"`
	PendingRobux                       *int64                           `json:"pendingRobux"`
	IncomingRobuxPastMonth             *int64                           `json:"incomingRobuxPastMonth"`
	OutgoingRobuxPastMonth             *int64                           `json:"outgoingRobuxPastMonth"`
	AgeBracket                         *int                             `json:"ageBracket"`
	AgeGroup                           string                           `json:"ageGroup"`
	AgeVerification                    AgeVerification                  `json:"ageVerification"`
	CountryCode                        string                           `json:"countryCode"`
	Premium                            *bool                            `json:"premium"`
	Plus                               *bool                            `json:"plus"`
	TwoStepEnabled                     *bool                            `json:"twoStepEnabled"`
	TwoStepMethods                     []string                         `json:"twoStepMethods"`
	VoiceChat                          *VoiceChat                       `json:"voiceChat"`
	Description                        string                           `json:"description"`
	SocialLinks                        []robloxservices.UserSocialLink  `json:"socialLinks"`
	PreviousNames                      []string                         `json:"previousNames"`
	VerifiedBadge                      *bool                            `json:"verifiedBadge"`
	FriendCount                        *int64                           `json:"friendCount"`
	FollowerCount                      *int64                           `json:"followerCount"`
	FollowingCount                     *int64                           `json:"followingCount"`
	PrimaryGroup                       *robloxservices.UserPrimaryGroup `json:"primaryGroup"`
	CommunityCount                     *int64                           `json:"communityCount"`
	RobloxBadges                       []robloxservices.RobloxBadge     `json:"robloxBadges"`
	CollectibleCount                   *int64                           `json:"collectibleCount"`
	CollectibleRecentAveragePriceTotal *int64                           `json:"collectibleRecentAveragePriceTotal"`
	UnreadNotificationCount            *int64                           `json:"unreadNotificationCount"`
	// PremiumSubscription is nil when the account has no Premium.
	PremiumSubscription *PremiumSubscription `json:"premiumSubscription"`
	TradeEligibility    *TradeEligibility    `json:"tradeEligibility"`
	// Connections lists how the other saved accounts are connected to the account.
	Connections []FriendConnection `json:"connections"`
}

func (service *Service) GetAccountProfile(ctx context.Context, accountID int64, section AccountProfileSection) (AccountProfileSnapshot, error) {
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
	logger := service.logger.With("operation_id", logging.NewOperationID("profile"), "operation", "profile-read", "section", section)
	started := time.Now()
	profile := AccountProfileSnapshot{AccountID: accountID, Section: section}
	var age accountAge
	reads := service.profileSectionReads(ctx, accountID, session.SecretVersion, account.RobloxUserID, section, logger, &profile, &age)
	if len(reads) == 0 {
		return AccountProfileSnapshot{}, profileError(roblox.KindProtocol, "This profile section does not exist.")
	}
	unavailable, err := service.runAccountReads(ctx, accountID, session.SecretVersion, "account-profile", logger, reads)
	if err != nil {
		return AccountProfileSnapshot{}, err
	}
	profile.IsComplete = len(unavailable) == 0
	if section == ProfileSectionAge {
		profile.AgeGroup, profile.AgeVerification = age.group, age.verification()
	}
	logger.Debug("account profile section read finished", "duration_ms", time.Since(started).Milliseconds(), "failed_reads", len(unavailable))
	return profile, nil
}

func (service *Service) profileSectionReads(ctx context.Context, accountID, version, userID int64, section AccountProfileSection, logger *slog.Logger, profile *AccountProfileSnapshot, age *accountAge) []accountRead {
	users := service.users
	readsBySection := map[AccountProfileSection]func() []accountRead{
		ProfileSectionAccount: func() []accountRead {
			return []accountRead{{"account", func() error { return service.readProfileAccount(ctx, accountID, version, userID, profile) }}}
		},
		ProfileSectionRobux: func() []accountRead {
			return []accountRead{applyRead("robux", func() (*int64, error) { return users.Robux(ctx, accountID, version) }, func(robux *int64) { profile.Robux = robux })}
		},
		ProfileSectionTransactions: func() []accountRead {
			return []accountRead{applyRead("transactions", func() (robloxservices.UserTransactionTotals, error) {
				return users.TransactionTotals(ctx, accountID, version, userID)
			}, func(totals robloxservices.UserTransactionTotals) {
				profile.PendingRobux, profile.IncomingRobuxPastMonth, profile.OutgoingRobuxPastMonth = &totals.PendingRobux, &totals.IncomingRobux, &totals.OutgoingRobux
			})}
		},
		ProfileSectionAge: func() []accountRead { return service.ageReads(ctx, accountID, version, age) },
		ProfileSectionTwoStep: func() []accountRead {
			return []accountRead{applyRead("twoStep", func() ([]robloxservices.UserVerificationMethod, error) {
				return users.TwoStepVerification(ctx, accountID, version, userID)
			}, func(methods []robloxservices.UserVerificationMethod) {
				profile.TwoStepMethods = enabledVerificationMethods(methods)
				profile.TwoStepEnabled = new(len(profile.TwoStepMethods) > 0)
			})}
		},
		ProfileSectionVoiceChat: func() []accountRead {
			return []accountRead{service.voiceChatRead(ctx, accountID, version, userID, &profile.VoiceChat)}
		},
		ProfileSectionAbout: func() []accountRead {
			return []accountRead{applyRead("about", func() (robloxservices.UserAbout, error) { return users.About(ctx, accountID, version, userID) }, func(about robloxservices.UserAbout) {
				profile.Description, profile.VerifiedBadge = about.Description, &about.VerifiedBadge
			})}
		},
		ProfileSectionSocialLinks: func() []accountRead {
			return []accountRead{applyRead("socialLinks", func() ([]robloxservices.UserSocialLink, error) { return users.SocialLinks(ctx, accountID, version) }, func(links []robloxservices.UserSocialLink) {
				profile.SocialLinks = links
			})}
		},
		ProfileSectionPreviousNames: func() []accountRead {
			return []accountRead{applyRead("previousNames", func() ([]string, error) { return users.PreviousNames(ctx, accountID, version, userID) }, func(names []string) {
				profile.PreviousNames = names
			})}
		},
		ProfileSectionFriends: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "friends", &profile.FriendCount)}
		},
		ProfileSectionFollowers: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "followers", &profile.FollowerCount)}
		},
		ProfileSectionFollowing: func() []accountRead {
			return []accountRead{service.socialCountRead(ctx, accountID, version, userID, "followings", &profile.FollowingCount)}
		},
		ProfileSectionPrimaryGroup: func() []accountRead {
			return []accountRead{applyRead("primaryGroup", func() (*robloxservices.UserPrimaryGroup, error) {
				return users.PrimaryGroup(ctx, accountID, version, userID)
			}, func(group *robloxservices.UserPrimaryGroup) {
				profile.PrimaryGroup = group
			})}
		},
		ProfileSectionCommunities: func() []accountRead {
			return []accountRead{applyRead("communities", func() (int64, error) { return users.CommunityCount(ctx, accountID, version, userID) }, func(count int64) {
				profile.CommunityCount = &count
			})}
		},
		ProfileSectionRobloxBadges: func() []accountRead {
			return []accountRead{applyRead("robloxBadges", func() ([]robloxservices.RobloxBadge, error) {
				return users.RobloxBadges(ctx, accountID, version, userID)
			}, func(badges []robloxservices.RobloxBadge) {
				profile.RobloxBadges = badges
			})}
		},
		ProfileSectionCollectibles: func() []accountRead {
			return []accountRead{applyRead("collectibles", func() (robloxservices.UserCollectibles, error) {
				return users.Collectibles(ctx, accountID, version, userID)
			}, func(collectibles robloxservices.UserCollectibles) {
				profile.CollectibleCount, profile.CollectibleRecentAveragePriceTotal = &collectibles.Count, &collectibles.RecentAveragePriceTotal
			})}
		},
		ProfileSectionActivity: func() []accountRead {
			return []accountRead{
				applyRead("unreadNotifications", func() (int64, error) { return users.UnreadNotificationCount(ctx, accountID, version) }, func(count int64) {
					profile.UnreadNotificationCount = &count
				}),
			}
		},
		ProfileSectionPremium: func() []accountRead {
			return []accountRead{applyRead("premium", func() (*robloxservices.UserPremiumSubscription, error) {
				return users.PremiumSubscription(ctx, accountID, version, userID)
			}, func(subscription *robloxservices.UserPremiumSubscription) {
				profile.PremiumSubscription = premiumSubscription(subscription)
			})}
		},
		ProfileSectionTrading: func() []accountRead {
			return []accountRead{applyRead("trading", func() (robloxservices.UserTradeEligibility, error) {
				return users.TradeEligibility(ctx, accountID, version)
			}, func(eligibility robloxservices.UserTradeEligibility) {
				profile.TradeEligibility = &TradeEligibility{CanTrade: eligibility.CanTrade, Eligibility: eligibility.Eligibility, FreeTrades: eligibility.FreeTrades}
			})}
		},
		ProfileSectionConnections: func() []accountRead {
			return []accountRead{{"connections", func() error {
				var readError error
				_, profile.Connections, readError = service.friendConnections(ctx, logger, accountID, userID)
				return readError
			}}}
		},
	}
	sectionReads, ok := readsBySection[section]
	if !ok {
		return nil
	}
	return sectionReads()
}

func premiumSubscription(subscription *robloxservices.UserPremiumSubscription) *PremiumSubscription {
	if subscription == nil {
		return nil
	}
	premium := &PremiumSubscription{Name: subscription.Name, RobuxStipend: subscription.RobuxStipend}
	if !subscription.RenewsAt.IsZero() {
		premium.RenewsAtMs = subscription.RenewsAt.UnixMilli()
	}
	if !subscription.ExpiresAt.IsZero() {
		premium.ExpiresAtMs = subscription.ExpiresAt.UnixMilli()
	}
	return premium
}

// applyRead runs read and passes its value to apply only when it succeeds.
func applyRead[Fetched any](source string, read func() (Fetched, error), apply func(Fetched)) accountRead {
	return accountRead{source, func() error {
		fetched, err := read()
		if err == nil {
			apply(fetched)
		}
		return err
	}}
}

func (service *Service) socialCountRead(ctx context.Context, accountID, version, userID int64, counter string, target **int64) accountRead {
	return applyRead(counter, func() (*int64, error) { return service.users.SocialCount(ctx, accountID, version, userID, counter) }, func(count *int64) {
		*target = count
	})
}

func enabledVerificationMethods(methods []robloxservices.UserVerificationMethod) []string {
	enabledMethods := []string{}
	for _, method := range methods {
		if method.Enabled != nil && *method.Enabled && !slices.Contains(enabledMethods, method.MediaType) {
			enabledMethods = append(enabledMethods, method.MediaType)
		}
	}
	return enabledMethods
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
