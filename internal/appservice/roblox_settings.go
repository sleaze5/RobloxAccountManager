package appservice

import (
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) GetMultiClientState() robloxmulti.Snapshot {
	return service.multiClient.Snapshot()
}

func (service *Service) SetMultiClientEnabled(enabled bool) (robloxmulti.Snapshot, error) {
	service.launchMutex.Lock()
	defer service.launchMutex.Unlock()
	if state := service.multiClient.Snapshot(); !state.Supported {
		return state, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "multi-client", Message: "Multi-client is only available on Windows."}
	}
	if err := service.settings.SetMultiClient(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "multi-client", "error", err)
		return service.multiClient.Snapshot(), &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "multi-client", Message: "Could not save the Multi-client setting. Try again."}
	}
	service.multiClient.SetEnabled(enabled)
	return service.multiClient.Snapshot(), nil
}

type RobloxPlayers struct {
	Versions   []gamelaunch.PlayerVersion `json:"versions"`
	CustomPath string                     `json:"customPath"`
}

func (service *Service) GetRobloxPlayers() RobloxPlayers {
	return RobloxPlayers{Versions: gamelaunch.PlayerVersions(), CustomPath: service.settings.Roblox().PlayerPath}
}

type RobloxPlayerPathCheck struct {
	Path    string `json:"path"`
	Found   bool   `json:"found"`
	Problem string `json:"problem"`
}

func (service *Service) CheckRobloxPlayerPath(path string) RobloxPlayerPathCheck {
	parsed, found, err := gamelaunch.CheckPlayerPath(path)
	if err != nil {
		return RobloxPlayerPathCheck{Problem: userLaunchMessage(err, "Could not read the custom path.")}
	}
	return RobloxPlayerPathCheck{Path: parsed, Found: found}
}

func (service *Service) SetRobloxPlayerPath(path string) error {
	path = strings.TrimSpace(path)
	if path != "" {
		if _, _, err := gamelaunch.CheckPlayerPath(path); err != nil {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-player-path", Message: userLaunchMessage(err, "Could not read the custom path."), Cause: err}
		}
	}
	if err := service.settings.SetPlayerPath(path); err != nil {
		service.logger.Warn("could not save Roblox Player path", "operation", "roblox-player-path", "error", err)
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-player-path", Message: "Could not save the custom Roblox Player path. Try again."}
	}
	return nil
}

func (service *Service) GetRobloxClients() gamelaunch.ClientState {
	return gamelaunch.CurrentClients(service.settings.Roblox().LinuxClient)
}

func (service *Service) SetLinuxClient(client string) (gamelaunch.ClientState, error) {
	state := service.GetRobloxClients()
	if !state.ChoiceSupported {
		return state, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "linux-client", Message: "Linux clients are only available on Linux."}
	}
	if err := service.settings.SetLinuxClient(client); err != nil {
		service.logger.Warn("could not save Roblox client", "operation", "linux-client", "error", err)
		return service.GetRobloxClients(), &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "linux-client", Message: "Could not save the Linux client. Try again."}
	}
	service.logger.Info("Roblox client saved", "operation", "linux-client", "client", client)
	return service.GetRobloxClients(), nil
}
