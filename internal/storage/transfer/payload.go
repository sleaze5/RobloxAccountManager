package transferstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	accountdomain "github.com/sleaze5/RobloxAccountManager/internal/accounts"
)

const (
	FormatVersion             = 2
	maxFavoriteNicknameLen    = 50
	maxPrivateServerNameLen   = 50
	maxPrivateServerLinkBytes = 2048
)

var (
	ErrUnsupportedVersion = errors.New("export data is from a newer version")
	ErrInvalidPayload     = errors.New("export data is damaged")
)

type object = map[string]json.RawMessage

var migrations = [...]func(object) (object, error){addSavedPrivateServers}

// addSavedPrivateServers gives a version 1 export the empty list of saved private servers that version 2 requires.
func addSavedPrivateServers(document object) (object, error) {
	document["version"] = json.RawMessage("2")
	document["savedPrivateServers"] = json.RawMessage("[]")
	return document, nil
}

var _ = [1]struct{}{}[len(migrations)-(FormatVersion-1)]

type Payload struct {
	Version             int                  `json:"version"`
	Accounts            []Account            `json:"accounts"`
	Tags                []string             `json:"tags"`
	FavoritePlaces      []FavoritePlace      `json:"favoritePlaces"`
	SavedPrivateServers []SavedPrivateServer `json:"savedPrivateServers"`
	Settings            json.RawMessage      `json:"settings"`
}

type SavedPrivateServer struct {
	PlaceID int64  `json:"placeId"`
	Name    string `json:"name"`
	Link    string `json:"link"`
}

type Account struct {
	RobloxUserID      int64    `json:"robloxUserId"`
	Username          string   `json:"username"`
	DisplayName       string   `json:"displayName"`
	CreatedAtMS       int64    `json:"createdAtMs"`
	Cookie            string   `json:"cookie"`
	CookieExpiresAtMS *int64   `json:"cookieExpiresAtMs"`
	Password          string   `json:"password"`
	Favorite          bool     `json:"favorite"`
	Tags              []string `json:"tags"`
}

type FavoritePlace struct {
	PlaceID  int64  `json:"placeId"`
	Nickname string `json:"nickname"`
}

func Encode(payload Payload) ([]byte, error) {
	payload.Version = FormatVersion
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode export data: %w", err)
	}
	return data, nil
}

func Decode(data []byte) (Payload, error) {
	var document object
	if err := json.Unmarshal(data, &document); err != nil || document == nil {
		return Payload{}, ErrInvalidPayload
	}
	var version int
	if err := json.Unmarshal(document["version"], &version); err != nil || version < 1 {
		return Payload{}, ErrInvalidPayload
	}
	if version > FormatVersion {
		return Payload{}, fmt.Errorf("%w: export data version %d", ErrUnsupportedVersion, version)
	}
	for from := version; from < FormatVersion; from++ {
		var err error
		if document, err = migrations[from-1](document); err != nil {
			return Payload{}, fmt.Errorf("%w: migrate export data from version %d: %w", ErrInvalidPayload, from, err)
		}
	}
	migrated, err := json.Marshal(document)
	if err != nil {
		return Payload{}, fmt.Errorf("encode migrated export data: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(migrated))
	decoder.DisallowUnknownFields()
	var payload Payload
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, fmt.Errorf("%w: %w", ErrInvalidPayload, err)
	}
	if err := payload.validate(); err != nil {
		return Payload{}, fmt.Errorf("%w: %w", ErrInvalidPayload, err)
	}
	return payload, nil
}

func (payload Payload) validate() error {
	tags, err := validateTags(payload.Tags)
	if err != nil {
		return err
	}
	if err := validateAccounts(payload.Accounts, tags); err != nil {
		return err
	}
	if err := validateFavoritePlaces(payload.FavoritePlaces); err != nil {
		return err
	}
	return validateSavedPrivateServers(payload.SavedPrivateServers)
}

func validateSavedPrivateServers(servers []SavedPrivateServer) error {
	type key struct {
		placeID int64
		link    string
	}
	seen := make(map[key]struct{}, len(servers))
	for _, server := range servers {
		if server.PlaceID <= 0 || server.Link == "" || len(server.Link) > maxPrivateServerLinkBytes || !utf8.ValidString(server.Link) ||
			!utf8.ValidString(server.Name) || utf8.RuneCountInString(server.Name) > maxPrivateServerNameLen {
			return errors.New("invalid saved private server")
		}
		if _, duplicate := seen[key{server.PlaceID, server.Link}]; duplicate {
			return fmt.Errorf("duplicate saved private server for place %d", server.PlaceID)
		}
		seen[key{server.PlaceID, server.Link}] = struct{}{}
	}
	return nil
}

func validateTags(names []string) (map[string]struct{}, error) {
	tags := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, err := accountdomain.ValidateTagName(name); err != nil {
			return nil, err
		}
		if _, duplicate := tags[name]; duplicate {
			return nil, fmt.Errorf("duplicate tag %q", name)
		}
		tags[name] = struct{}{}
	}
	return tags, nil
}

func validateAccounts(accounts []Account, tags map[string]struct{}) error {
	users := make(map[int64]struct{}, len(accounts))
	for _, account := range accounts {
		if account.RobloxUserID <= 0 || account.Cookie == "" {
			return errors.New("invalid account")
		}
		if !utf8.ValidString(account.Username) || !utf8.ValidString(account.DisplayName) || !utf8.ValidString(account.Password) {
			return errors.New("invalid account text")
		}
		if _, duplicate := users[account.RobloxUserID]; duplicate {
			return fmt.Errorf("duplicate account %d", account.RobloxUserID)
		}
		users[account.RobloxUserID] = struct{}{}
		for _, name := range account.Tags {
			if _, known := tags[name]; !known {
				return fmt.Errorf("unknown account tag %q", name)
			}
		}
	}
	return nil
}

func validateFavoritePlaces(places []FavoritePlace) error {
	seen := make(map[int64]struct{}, len(places))
	for _, place := range places {
		if place.PlaceID <= 0 || !utf8.ValidString(place.Nickname) || utf8.RuneCountInString(place.Nickname) > maxFavoriteNicknameLen {
			return errors.New("invalid favorite place")
		}
		if _, duplicate := seen[place.PlaceID]; duplicate {
			return fmt.Errorf("duplicate favorite place %d", place.PlaceID)
		}
		seen[place.PlaceID] = struct{}{}
	}
	return nil
}
