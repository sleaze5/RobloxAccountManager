package appservice

import (
	"strings"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/platform/robloxmulti"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

func (service *Service) GetMultiInstanceState() robloxmulti.Snapshot {
	return service.multiInstance.Snapshot()
}

func (service *Service) SetMultiInstanceEnabled(enabled bool) (robloxmulti.Snapshot, error) {
	service.launchMu.Lock()
	defer service.launchMu.Unlock()
	if state := service.multiInstance.Snapshot(); !state.Supported {
		return state, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "multi-instance", Message: "Multi-instance is only available on Windows."}
	}
	if err := service.settings.SetMultiInstance(enabled); err != nil {
		service.logger.Warn("could not save Roblox settings", "operation", "multi-instance", "error", err)
		return service.multiInstance.Snapshot(), &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "multi-instance", Message: "The multi-instance setting could not be saved. Try again."}
	}
	service.multiInstance.SetEnabled(enabled)
	return service.multiInstance.Snapshot(), nil
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
		return RobloxPlayerPathCheck{Problem: userLaunchMessage(err, "The custom path could not be read.")}
	}
	return RobloxPlayerPathCheck{Path: parsed, Found: found}
}

func (service *Service) SetRobloxPlayerPath(path string) error {
	path = strings.TrimSpace(path)
	if path != "" {
		if _, _, err := gamelaunch.CheckPlayerPath(path); err != nil {
			return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-player-path", Message: userLaunchMessage(err, "The custom path could not be read."), Cause: err}
		}
	}
	if err := service.settings.SetPlayerPath(path); err != nil {
		service.logger.Warn("could not save Roblox Player path", "operation", "roblox-player-path", "error", err)
		return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "roblox-player-path", Message: "The custom Roblox Player path could not be saved. Try again."}
	}
	return nil
}

func (service *Service) GetRobloxClients() gamelaunch.ClientState {
	return gamelaunch.CurrentClients(service.settings.Roblox().LinuxClient)
}

func (service *Service) SetLinuxClient(client string) (gamelaunch.ClientState, error) {
	state := service.GetRobloxClients()
	if !state.ChoiceSupported {
		return state, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "linux-client", Message: "Choosing a Linux Roblox client is only available on Linux."}
	}
	if err := service.settings.SetLinuxClient(client); err != nil {
		service.logger.Warn("could not save Roblox client", "operation", "linux-client", "error", err)
		return service.GetRobloxClients(), &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "linux-client", Message: "The Linux client could not be saved. Try again."}
	}
	service.logger.Info("Roblox client saved", "operation", "linux-client", "client", client)
	return service.GetRobloxClients(), nil
}
