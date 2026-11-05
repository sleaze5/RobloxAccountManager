package appservice

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
	gamestore "github.com/sleaze5/RobloxAccountManager/internal/storage/games"
)

const maxPrivateServerNameLength = 50

var privateServerUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,20}$`)

func (service *Service) ListOwnedPrivateServers(ctx context.Context, accountID int64, cursor string) (games.OwnedPrivateServerPage, error) {
	if len(cursor) > 2048 {
		return games.OwnedPrivateServerPage{}, privateServerInputError("This page has expired. Refresh the private servers.")
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "private-servers-owned")
	if err != nil {
		return games.OwnedPrivateServerPage{}, err
	}
	defer done()
	return service.privateServers.Owned(ctx, accountID, version, cursor)
}

func (service *Service) ListPlacePrivateServers(ctx context.Context, query games.PrivateServerQuery) (games.PrivateServerPage, error) {
	if err := validGameID(query.PlaceID); err != nil {
		return games.PrivateServerPage{}, err
	}
	if len(query.Cursor) > 2048 {
		return games.PrivateServerPage{}, privateServerInputError("This page has expired. Refresh the private servers.")
	}
	ctx, version, done, err := service.beginAccountSession(ctx, query.AccountID, "private-servers-place")
	if err != nil {
		return games.PrivateServerPage{}, err
	}
	defer done()
	account, err := service.repo.GetView(ctx, query.AccountID)
	if err != nil {
		return games.PrivateServerPage{}, mapRepositoryError(err)
	}
	return service.privateServers.ForPlace(ctx, query.AccountID, version, account.RobloxUserID, query)
}

func (service *Service) GetPrivateServer(ctx context.Context, accountID, serverID int64) (games.PrivateServerSettings, error) {
	if err := validPrivateServerID(serverID); err != nil {
		return games.PrivateServerSettings{}, err
	}
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "private-server")
	if err != nil {
		return games.PrivateServerSettings{}, err
	}
	defer done()
	return service.privateServerSettings(ctx, accountID, version, serverID)
}

func (service *Service) RenamePrivateServer(ctx context.Context, accountID, serverID int64, name string) (games.PrivateServerSettings, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxPrivateServerNameLength || strings.ContainsFunc(name, unicode.IsControl) {
		return games.PrivateServerSettings{}, privateServerInputError("Enter a name of up to 50 characters.")
	}
	return service.updatePrivateServer(ctx, accountID, serverID, "private-server-rename", func(context.Context) (robloxservices.PrivateServerUpdate, error) {
		return robloxservices.PrivateServerUpdate{Name: &name}, nil
	})
}

func (service *Service) SetPrivateServerOption(ctx context.Context, accountID, serverID int64, option games.PrivateServerOption, enabled bool) (games.PrivateServerSettings, error) {
	var update robloxservices.PrivateServerUpdate
	switch option {
	case games.PrivateServerActive:
		update.Active = &enabled
	case games.PrivateServerFriendsAllowed:
		update.FriendsAllowed = &enabled
	case games.PrivateServerRenews:
		update.Renews = &enabled
	default:
		return games.PrivateServerSettings{}, privateServerInputError("Choose a private server setting.")
	}
	return service.updatePrivateServer(ctx, accountID, serverID, "private-server-"+string(option), func(context.Context) (robloxservices.PrivateServerUpdate, error) {
		return update, nil
	})
}

func (service *Service) ResetPrivateServerLink(ctx context.Context, accountID, serverID int64) (games.PrivateServerSettings, error) {
	reset := true
	return service.updatePrivateServer(ctx, accountID, serverID, "private-server-link-reset", func(context.Context) (robloxservices.PrivateServerUpdate, error) {
		return robloxservices.PrivateServerUpdate{NewJoinCode: &reset}, nil
	})
}

func (service *Service) AddPrivateServerUser(ctx context.Context, accountID, serverID int64, user string) (games.PrivateServerSettings, error) {
	userID, username, err := privateServerUserInput(user)
	if err != nil {
		return games.PrivateServerSettings{}, err
	}
	return service.updatePrivateServer(ctx, accountID, serverID, "private-server-user-add", func(ctx context.Context) (robloxservices.PrivateServerUpdate, error) {
		if username != "" {
			id, err := service.users.UserIDFromUsername(ctx, username)
			if err != nil {
				return robloxservices.PrivateServerUpdate{}, err
			}
			userID = id
		}
		return robloxservices.PrivateServerUpdate{UsersToAdd: []int64{userID}}, nil
	})
}

func privateServerUserInput(user string) (int64, string, error) {
	user = strings.TrimPrefix(strings.TrimSpace(user), "@")
	if len(user) > 3 && strings.EqualFold(user[:3], "id=") {
		id, err := strconv.ParseInt(user[3:], 10, 64)
		if err != nil || !games.ValidID(id) {
			return 0, "", privateServerInputError("Enter a valid user ID after id=, such as id=1.")
		}
		return id, "", nil
	}
	if !privateServerUsernamePattern.MatchString(user) {
		return 0, "", privateServerInputError("Enter a username, or id= and a user ID.")
	}
	return 0, user, nil
}

func (service *Service) RemovePrivateServerUser(ctx context.Context, accountID, serverID, userID int64) (games.PrivateServerSettings, error) {
	if !games.ValidID(userID) {
		return games.PrivateServerSettings{}, privateServerInputError("The user is invalid. Refresh the private server.")
	}
	return service.updatePrivateServer(ctx, accountID, serverID, "private-server-user-remove", func(context.Context) (robloxservices.PrivateServerUpdate, error) {
		return robloxservices.PrivateServerUpdate{UsersToRemove: []int64{userID}}, nil
	})
}

func (service *Service) ListRBXServersPrivateServers(ctx context.Context, placeID int64) (games.ListedPrivateServerPage, error) {
	if err := validGameID(placeID); err != nil {
		return games.ListedPrivateServerPage{}, err
	}
	if !service.settings.Integrations().RBXServers {
		return games.ListedPrivateServerPage{}, gamesInputError("Turn on the RBXServers integration in Settings to use RBXServers servers.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.ListedPrivateServerPage{}, err
	}
	defer done()
	page, err := service.rbxservers.Servers(ctx, placeID)
	if err == nil {
		return page, nil
	}
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return games.ListedPrivateServerPage{}, &roblox.Error{Kind: roblox.KindCancelled, Endpoint: "rbxservers-servers", Message: "The RBXServers request was cancelled."}
	}
	service.logs.Module("integration.rbxservers").WarnContext(ctx, "rbxservers request failed", "operation", "rbxservers-servers", "place_id", placeID, "error", err)
	return games.ListedPrivateServerPage{}, &roblox.Error{Kind: roblox.KindServer, Endpoint: "rbxservers-servers", Message: "RBXServers could not be reached. Try again later.", Cause: err}
}

func (service *Service) updatePrivateServer(ctx context.Context, accountID, serverID int64, operation string, change func(context.Context) (robloxservices.PrivateServerUpdate, error)) (games.PrivateServerSettings, error) {
	if err := validPrivateServerID(serverID); err != nil {
		return games.PrivateServerSettings{}, err
	}
	var settings games.PrivateServerSettings
	err := service.editAccount(ctx, accountID, operation, "Roblox rejected the change. Only the owner can change this private server.", func(ctx context.Context, version int64) error {
		update, err := change(ctx)
		if err != nil {
			return err
		}
		if err := service.privateServers.Update(ctx, accountID, version, serverID, update); err != nil {
			return err
		}
		if settings, err = service.privateServerSettings(ctx, accountID, version, serverID); err != nil {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Roblox saved the change, but the private server could not be reloaded. Refresh to check.", Cause: err}
		}
		return nil
	})
	return settings, err
}

func (service *Service) privateServerSettings(ctx context.Context, accountID, version, serverID int64) (games.PrivateServerSettings, error) {
	settings, err := service.privateServers.Settings(ctx, accountID, version, serverID)
	if err != nil || len(settings.Users) == 0 {
		return settings, err
	}
	ids := make([]int64, len(settings.Users))
	for index, user := range settings.Users {
		ids[index] = user.UserID
	}
	headshots, err := service.avatars.AvatarHeadshots(ctx, ids)
	if err != nil {
		service.logger.Warn("private server user avatars unavailable", "operation", "private-server", "account_id", accountID, "error", err)
		return settings, nil
	}
	images := make(map[int64]string, len(headshots))
	for _, headshot := range headshots {
		images[headshot.RobloxUserID] = headshot.ImageURL
	}
	for index := range settings.Users {
		settings.Users[index].ImageURL = images[settings.Users[index].UserID]
	}
	return settings, nil
}

func validPrivateServerID(serverID int64) error {
	if !games.ValidID(serverID) {
		return privateServerInputError("The private server is invalid. Refresh the private servers.")
	}
	return nil
}

func privateServerInputError(message string) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "private-servers", Message: message}
}

func (service *Service) ListSavedPrivateServers(ctx context.Context, placeID int64) ([]games.SavedPrivateServer, error) {
	if err := validGameID(placeID); err != nil {
		return nil, err
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	servers, err := service.favoritePlaces.PrivateServers(ctx, placeID)
	return servers, service.gamesError(ctx, "private-server-links", err)
}

func (service *Service) SavePrivateServerLink(ctx context.Context, placeID, accountID int64, name, link string) (games.SavedPrivateServer, error) {
	if err := validGameID(placeID); err != nil {
		return games.SavedPrivateServer{}, err
	}
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > maxPrivateServerNameLength || strings.ContainsFunc(name, unicode.IsControl) {
		return games.SavedPrivateServer{}, privateServerInputError("Enter a name of up to 50 characters.")
	}
	target, err := service.privateServerTarget(ctx, accountID, link)
	if err != nil {
		return games.SavedPrivateServer{}, err
	}
	if target.PlaceID != placeID {
		return games.SavedPrivateServer{}, privateServerInputError("This link is for a different game.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return games.SavedPrivateServer{}, err
	}
	defer done()
	server, err := service.favoritePlaces.SavePrivateServer(ctx, games.SavedPrivateServer{PlaceID: placeID, Name: name, AccessCode: target.AccessCode, LinkCode: target.LinkCode})
	if errors.Is(err, gamestore.ErrPrivateServerSaved) {
		return games.SavedPrivateServer{}, privateServerInputError("This private server is already saved for this game.")
	}
	return server, service.gamesError(ctx, "private-server-link-save", err)
}

// privateServerTarget reads the place ID and access code from a private server link.
// Share links and link codes carry no access code, so Roblox resolves them for the account.
func (service *Service) privateServerTarget(ctx context.Context, accountID int64, link string) (gamelaunch.Request, error) {
	link = strings.TrimSpace(link)
	invalid := privateServerInputError("Paste a private server link, a server share link, or a link with an access code.")
	if len(link) > 2048 {
		return gamelaunch.Request{}, invalid
	}
	target, err := gamelaunch.Parse(gamelaunch.Input{Method: gamelaunch.MethodLink, Link: link})
	if err != nil || target.LinkCode == "" && target.AccessCode == "" && target.ShareType != gamelaunch.ShareServer {
		return gamelaunch.Request{}, invalid
	}
	if target.AccessCode == "" {
		if accountID <= 0 {
			return gamelaunch.Request{}, privateServerInputError("Add an account first. Roblox reads the access code of this link for an account.")
		}
		ctx, version, done, err := service.beginAccountSession(ctx, accountID, "private-server-link-resolve")
		if err != nil {
			return gamelaunch.Request{}, err
		}
		defer done()
		if target, err = service.gameLinks.Resolve(ctx, accountID, version, target); err != nil {
			return gamelaunch.Request{}, err
		}
	}
	if target.PlaceID <= 0 || target.AccessCode == "" {
		return gamelaunch.Request{}, invalid
	}
	return target, nil
}

func (service *Service) RemoveSavedPrivateServer(ctx context.Context, id int64) error {
	if !games.ValidID(id) {
		return privateServerInputError("The saved link is invalid. Refresh the private servers.")
	}
	ctx, done, err := service.beginGames(ctx)
	if err != nil {
		return err
	}
	defer done()
	if err := service.favoritePlaces.RemovePrivateServer(ctx, id); errors.Is(err, gamestore.ErrPrivateServerNotSaved) {
		return privateServerInputError("This link was already removed. Refresh the private servers.")
	} else if err != nil {
		return service.gamesError(ctx, "private-server-link-remove", err)
	}
	return nil
}
