package appservice

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/games"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// ListFriendsInGame lists the account's friends who play any place of the
// universe, grouped by place and then by server. Roblox omits the universe of
// a friend whose privacy hides their game, the server of a friend whose
// privacy hides joining them, and sometimes the place too.
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
	if len(playing.playingFriendIDs) == 0 {
		return games.FriendsInGame{Places: []games.FriendPlace{}, FriendsInHiddenPlaces: []games.FriendInServer{}}, nil
	}
	users, err := service.users.Names(ctx, playing.playingFriendIDs)
	if err != nil {
		return games.FriendsInGame{}, service.gamesError(ctx, operation, err)
	}
	images, err := service.headshotImages(ctx, playing.playingFriendIDs)
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
	placeNames := service.friendPlaceNames(ctx, logger, universeID, playing.placeIDs())
	friends := games.FriendsInGame{
		Places:                friendPlaces(playing, placeNames, friendByUserID),
		FriendsInHiddenPlaces: namedFriends(playing.hiddenPlaceFriendIDs, friendByUserID),
	}
	logger.DebugContext(ctx, "friends in game read", "place_count", len(friends.Places), "hidden_place_friend_count", len(friends.FriendsInHiddenPlaces))
	return friends, nil
}

// friendPlaceNames names the places from the cache, and lists the universe's
// places once when the cache lacks one of them. A place that stays unnamed
// keeps the cache's placeholder name.
func (service *Service) friendPlaceNames(ctx context.Context, logger *slog.Logger, universeID int64, placeIDs []int64) map[int64]string {
	namesByPlaceID := make(map[int64]string, len(placeIDs))
	isEveryPlaceCached := true
	for _, placeID := range placeIDs {
		place, isFresh := service.games.CachedPlace(ctx, placeID)
		namesByPlaceID[placeID] = place.Name
		isEveryPlaceCached = isEveryPlaceCached && isFresh
	}
	if isEveryPlaceCached {
		return namesByPlaceID
	}
	places, _, err := service.games.UniversePlaces(ctx, universeID)
	if err != nil {
		logger.WarnContext(ctx, "friend place names unavailable", "universe_id", universeID, "error", err)
		return namesByPlaceID
	}
	for _, place := range places {
		if _, isFriendPlace := namesByPlaceID[place.PlaceID]; isFriendPlace {
			namesByPlaceID[place.PlaceID] = place.Name
		}
	}
	return namesByPlaceID
}

type friendsPlayingPlace struct {
	friendIDsByJobID      map[string][]int64
	hiddenServerFriendIDs []int64
}

type friendsPlayingGame struct {
	rootPlaceID          int64
	placeByID            map[int64]*friendsPlayingPlace
	hiddenPlaceFriendIDs []int64
	playingFriendIDs     []int64
}

func (playing friendsPlayingGame) placeIDs() []int64 {
	placeIDs := make([]int64, 0, len(playing.placeByID))
	for placeID := range playing.placeByID {
		placeIDs = append(placeIDs, placeID)
	}
	return placeIDs
}

func groupFriendsInGame(presences []robloxservices.UserPresence, universeID int64) friendsPlayingGame {
	playing := friendsPlayingGame{placeByID: map[int64]*friendsPlayingPlace{}, hiddenPlaceFriendIDs: []int64{}, playingFriendIDs: []int64{}}
	for _, presence := range presences {
		if presence.UserPresenceType != robloxservices.PresenceTypeInGame || presence.UniverseID == nil || *presence.UniverseID != universeID {
			continue
		}
		playing.playingFriendIDs = append(playing.playingFriendIDs, presence.UserID)
		if presence.RootPlaceID != nil && games.ValidID(*presence.RootPlaceID) {
			playing.rootPlaceID = *presence.RootPlaceID
		}
		if presence.PlaceID == nil || !games.ValidID(*presence.PlaceID) {
			playing.hiddenPlaceFriendIDs = append(playing.hiddenPlaceFriendIDs, presence.UserID)
			continue
		}
		place := playing.placeByID[*presence.PlaceID]
		if place == nil {
			place = &friendsPlayingPlace{friendIDsByJobID: map[string][]int64{}}
			playing.placeByID[*presence.PlaceID] = place
		}
		if games.ValidJobID(presence.GameID) {
			place.friendIDsByJobID[presence.GameID] = append(place.friendIDsByJobID[presence.GameID], presence.UserID)
		} else {
			place.hiddenServerFriendIDs = append(place.hiddenServerFriendIDs, presence.UserID)
		}
	}
	return playing
}

// friendPlaces lists the root place first and the subplaces by friend count,
// and drops unnamed friends and the places left without friends.
func friendPlaces(playing friendsPlayingGame, namesByPlaceID map[int64]string, friendByUserID map[int64]games.FriendInServer) []games.FriendPlace {
	places := make([]games.FriendPlace, 0, len(playing.placeByID))
	friendCountByPlaceID := make(map[int64]int, len(playing.placeByID))
	for placeID, playingPlace := range playing.placeByID {
		place := games.FriendPlace{
			PlaceID: placeID, Name: namesByPlaceID[placeID], IsRootPlace: placeID == playing.rootPlaceID,
			Servers:                friendServers(playingPlace.friendIDsByJobID, friendByUserID),
			FriendsInHiddenServers: namedFriends(playingPlace.hiddenServerFriendIDs, friendByUserID),
		}
		friendCount := len(place.FriendsInHiddenServers)
		for _, server := range place.Servers {
			friendCount += len(server.Friends)
		}
		if friendCount > 0 {
			friendCountByPlaceID[placeID] = friendCount
			places = append(places, place)
		}
	}
	slices.SortFunc(places, func(left, right games.FriendPlace) int {
		if left.IsRootPlace != right.IsRootPlace {
			if left.IsRootPlace {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(friendCountByPlaceID[right.PlaceID], friendCountByPlaceID[left.PlaceID]), strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)), cmp.Compare(left.PlaceID, right.PlaceID))
	})
	return places
}

// friendServers sorts servers by friend count and drops unnamed friends.
func friendServers(friendIDsByJobID map[string][]int64, friendByUserID map[int64]games.FriendInServer) []games.FriendServer {
	servers := make([]games.FriendServer, 0, len(friendIDsByJobID))
	for jobID, userIDs := range friendIDsByJobID {
		server := games.FriendServer{JobID: jobID, Friends: namedFriends(userIDs, friendByUserID)}
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
