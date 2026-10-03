package appservice

import (
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
