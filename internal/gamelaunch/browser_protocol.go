package gamelaunch

import (
	"errors"
	"net/url"
	"strings"
)

func ValidateBrowserProtocolURL(protocolURL string) error {
	const prefix = "roblox-player:1+"
	if len(protocolURL) > 30000 || !strings.HasPrefix(strings.ToLower(protocolURL), prefix) || strings.IndexFunc(protocolURL, func(character rune) bool {
		return character <= ' ' || character > '~' || character == '"' || character == '\\'
	}) >= 0 {
		return errors.New("the browser supplied an invalid Roblox launch URL")
	}
	parameters, err := browserProtocolParameters(protocolURL[len(prefix):])
	if err != nil {
		return err
	}
	if parameters["launchmode"] != "play" || parameters["gameinfo"] == "" {
		return errors.New("the browser launch is not an authenticated Roblox game launch")
	}
	return validateBrowserLauncher(parameters["placelauncherurl"])
}

func browserProtocolParameters(encodedParameters string) (map[string]string, error) {
	parameters := make(map[string]string)
	for part := range strings.SplitSeq(encodedParameters, "+") {
		name, raw, found := strings.Cut(part, ":")
		name = strings.ToLower(name)
		if _, duplicate := parameters[name]; !found || name == "" || duplicate {
			return nil, errors.New("the browser supplied invalid Roblox launch parameters")
		}
		decoded, err := url.QueryUnescape(raw)
		if err != nil || strings.ContainsAny(decoded, "\x00\r\n") {
			return nil, errors.New("the browser supplied invalid Roblox launch parameters")
		}
		parameters[name] = decoded
	}
	return parameters, nil
}

func validateBrowserLauncher(launcherURL string) error {
	launcher, err := url.Parse(launcherURL)
	if err != nil || launcher.Scheme != "https" || launcher.User != nil || launcher.Fragment != "" ||
		launcher.Host != "www.roblox.com" ||
		!strings.EqualFold(launcher.Path, "/game/PlaceLauncher.ashx") {
		return errors.New("the browser supplied an invalid Roblox game launcher address")
	}
	return nil
}
