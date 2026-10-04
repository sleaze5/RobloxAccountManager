package appservice

import (
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
