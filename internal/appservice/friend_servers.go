package appservice

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListFriendServers lists the place's servers that the account's friends are
// in, most friends first. Roblox omits friends whose privacy hides their server.
func (service *Service) ListFriendServers(ctx context.Context, accountID, placeID int64) ([]games.FriendServer, error) {
	if err := validGameID(placeID); err != nil {
		return nil, err
	}
	ctx, version, logger, done, err := service.beginFeatureOperation(ctx, accountID, "games", "friend-servers")
	if err != nil {
		return nil, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return nil, mapRepositoryError(err)
	}
	friendIDs, err := service.users.FriendIDs(ctx, accountID, version, account.RobloxUserID)
	if err != nil {
		return nil, service.gamesError(ctx, "friend-servers", err)
	}
	presences, err := service.presence.UserPresences(ctx, accountID, version, friendIDs)
	if err != nil {
		return nil, service.gamesError(ctx, "friend-servers", err)
	}
	friendIDsByJobID := groupFriendsByJobID(presences, placeID)
	if len(friendIDsByJobID) == 0 {
		return []games.FriendServer{}, nil
	}
	playingFriendIDs := slices.Concat(slices.Collect(maps.Values(friendIDsByJobID))...)
	users, err := service.users.Names(ctx, playingFriendIDs)
	if err != nil {
		return nil, service.gamesError(ctx, "friend-servers", err)
	}
	images, err := service.headshotImages(ctx, playingFriendIDs)
	if ctx.Err() != nil {
		return nil, service.gamesError(ctx, "friend-servers", ctx.Err())
	}
	if err != nil {
		logger.WarnContext(ctx, "friend avatars unavailable", "error", err)
	}
	friendByUserID := make(map[int64]games.FriendInServer, len(users))
	for _, user := range users {
		friendByUserID[user.UserID] = games.FriendInServer{
			UserID: user.UserID, Username: user.Username, DisplayName: cmp.Or(user.DisplayName, user.Username), ImageURL: images[user.UserID],
		}
	}
	servers := friendServers(friendIDsByJobID, friendByUserID)
	logger.DebugContext(ctx, "friend servers read", "count", len(servers))
	return servers, nil
}

func groupFriendsByJobID(presences []robloxservices.UserPresence, placeID int64) map[string][]int64 {
	friendIDsByJobID := map[string][]int64{}
	for _, presence := range presences {
		if presence.UserPresenceType != robloxservices.PresenceTypeInGame || presence.PlaceID == nil || *presence.PlaceID != placeID || !games.ValidJobID(presence.GameID) {
			continue
		}
		friendIDsByJobID[presence.GameID] = append(friendIDsByJobID[presence.GameID], presence.UserID)
	}
	return friendIDsByJobID
}

// friendServers sorts servers by friend count and drops unnamed friends.
func friendServers(friendIDsByJobID map[string][]int64, friendByUserID map[int64]games.FriendInServer) []games.FriendServer {
	servers := make([]games.FriendServer, 0, len(friendIDsByJobID))
	for jobID, userIDs := range friendIDsByJobID {
		server := games.FriendServer{JobID: jobID, Friends: make([]games.FriendInServer, 0, len(userIDs))}
		for _, userID := range userIDs {
			if friend, found := friendByUserID[userID]; found {
				server.Friends = append(server.Friends, friend)
			}
		}
		if len(server.Friends) == 0 {
			continue
		}
		slices.SortFunc(server.Friends, func(left, right games.FriendInServer) int {
			return strings.Compare(strings.ToLower(left.DisplayName), strings.ToLower(right.DisplayName))
		})
		servers = append(servers, server)
	}
	slices.SortFunc(servers, func(left, right games.FriendServer) int {
		return cmp.Or(cmp.Compare(len(right.Friends), len(left.Friends)), strings.Compare(left.JobID, right.JobID))
	})
	return servers
}
