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

// ListFriendsInGame lists the account's friends who play any place of
// the universe, grouped by server with the most friends first. Roblox omits
// the universe of a friend whose privacy hides their game, and the server of a
// friend whose privacy hides joining them.
func (service *Service) ListFriendsInGame(ctx context.Context, accountID, universeID int64) (games.FriendsInGame, error) {
	const operation = "friends-in-game"
	if err := validGameID(universeID); err != nil {
		return games.FriendsInGame{}, err
	}
	ctx, version, logger, done, err := service.beginFeatureOperation(ctx, accountID, "games", operation)
	if err != nil {
		return games.FriendsInGame{}, err
	}
	defer done()
	account, err := service.accountRepository.GetView(ctx, accountID)
	if err != nil {
		return games.FriendsInGame{}, mapRepositoryError(err)
	}
	friendIDs, err := service.users.FriendIDs(ctx, accountID, version, account.RobloxUserID)
	if err != nil {
		return games.FriendsInGame{}, service.gamesError(ctx, operation, err)
	}
	presences, err := service.presence.UserPresences(ctx, accountID, version, friendIDs)
	if err != nil {
		return games.FriendsInGame{}, service.gamesError(ctx, operation, err)
	}
	playing := groupFriendsInGame(presences, universeID)
	if len(playing.friendIDsByServer) == 0 && len(playing.hiddenServerFriendIDs) == 0 {
		return games.FriendsInGame{Servers: []games.FriendServer{}, FriendsInHiddenServers: []games.FriendInServer{}}, nil
	}
	playingFriendIDs := slices.Concat(append(slices.Collect(maps.Values(playing.friendIDsByServer)), playing.hiddenServerFriendIDs)...)
	users, err := service.users.Names(ctx, playingFriendIDs)
	if err != nil {
		return games.FriendsInGame{}, service.gamesError(ctx, operation, err)
	}
	images, err := service.headshotImages(ctx, playingFriendIDs)
	if ctx.Err() != nil {
		return games.FriendsInGame{}, service.gamesError(ctx, operation, ctx.Err())
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
	friends := games.FriendsInGame{
		Servers:                friendServers(playing.friendIDsByServer, friendByUserID),
		FriendsInHiddenServers: namedFriends(playing.hiddenServerFriendIDs, friendByUserID),
	}
	logger.DebugContext(ctx, "friends in game read", "server_count", len(friends.Servers), "hidden_server_friend_count", len(friends.FriendsInHiddenServers))
	return friends, nil
}

type friendServerKey struct {
	jobID   string
	placeID int64
}

type friendsPlayingGame struct {
	friendIDsByServer     map[friendServerKey][]int64
	hiddenServerFriendIDs []int64
}

func groupFriendsInGame(presences []robloxservices.UserPresence, universeID int64) friendsPlayingGame {
	playing := friendsPlayingGame{friendIDsByServer: map[friendServerKey][]int64{}, hiddenServerFriendIDs: []int64{}}
	for _, presence := range presences {
		if presence.UserPresenceType != robloxservices.PresenceTypeInGame || presence.UniverseID == nil || *presence.UniverseID != universeID {
			continue
		}
		if presence.PlaceID == nil || !games.ValidID(*presence.PlaceID) || !games.ValidJobID(presence.GameID) {
			playing.hiddenServerFriendIDs = append(playing.hiddenServerFriendIDs, presence.UserID)
			continue
		}
		key := friendServerKey{jobID: presence.GameID, placeID: *presence.PlaceID}
		playing.friendIDsByServer[key] = append(playing.friendIDsByServer[key], presence.UserID)
	}
	return playing
}

// friendServers sorts servers by friend count and drops unnamed friends.
func friendServers(friendIDsByServer map[friendServerKey][]int64, friendByUserID map[int64]games.FriendInServer) []games.FriendServer {
	servers := make([]games.FriendServer, 0, len(friendIDsByServer))
	for key, userIDs := range friendIDsByServer {
		server := games.FriendServer{JobID: key.jobID, PlaceID: key.placeID, Friends: namedFriends(userIDs, friendByUserID)}
		if len(server.Friends) > 0 {
			servers = append(servers, server)
		}
	}
	slices.SortFunc(servers, func(left, right games.FriendServer) int {
		return cmp.Or(cmp.Compare(len(right.Friends), len(left.Friends)), strings.Compare(left.JobID, right.JobID))
	})
	return servers
}

// namedFriends sorts friends by display name and drops unnamed friends.
func namedFriends(userIDs []int64, friendByUserID map[int64]games.FriendInServer) []games.FriendInServer {
	friends := make([]games.FriendInServer, 0, len(userIDs))
	for _, userID := range userIDs {
		if friend, found := friendByUserID[userID]; found {
			friends = append(friends, friend)
		}
	}
	slices.SortFunc(friends, func(left, right games.FriendInServer) int {
		return strings.Compare(strings.ToLower(left.DisplayName), strings.ToLower(right.DisplayName))
	})
	return friends
}
