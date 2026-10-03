package gamelaunch

import (
	"errors"
	"net/url"
	"strings"
)

// Browser launch URLs carry the browser account's ticket. Validate the handoff
// without rebuilding it or substituting the account stored in the vault.
func ValidateBrowserProtocolURL(value string) error {
	const prefix = "roblox-player:1+"
	if len(value) > 30000 || !strings.HasPrefix(strings.ToLower(value), prefix) || strings.IndexFunc(value, func(character rune) bool {
		return character <= ' ' || character > '~' || character == '"' || character == '\\'
	}) >= 0 {
		return errors.New("the browser supplied an invalid Roblox launch URL")
	}
	parameters := make(map[string]string)
	for _, part := range strings.Split(value[len(prefix):], "+") {
		name, raw, found := strings.Cut(part, ":")
		name = strings.ToLower(name)
		if _, duplicate := parameters[name]; !found || name == "" || duplicate {
			return errors.New("the browser supplied invalid Roblox launch parameters")
		}
		decoded, err := url.QueryUnescape(raw)
		if err != nil || strings.ContainsAny(decoded, "\x00\r\n") {
			return errors.New("the browser supplied invalid Roblox launch parameters")
		}
		parameters[name] = decoded
	}
	if parameters["launchmode"] != "play" || parameters["gameinfo"] == "" {
		return errors.New("the browser launch is not an authenticated Roblox game launch")
	}
	launcher, err := url.Parse(parameters["placelauncherurl"])
	if err != nil || launcher.Scheme != "https" || launcher.User != nil || launcher.Fragment != "" ||
		(launcher.Host != "www.roblox.com" && launcher.Host != "assetgame.roblox.com") ||
		!strings.EqualFold(launcher.Path, "/game/PlaceLauncher.ashx") {
		return errors.New("the browser supplied an invalid Roblox game launcher address")
	}
	return nil
}
