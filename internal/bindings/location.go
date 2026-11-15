package bindings

import "github.com/sleaze5/RobloxAccountManager/internal/appdata"

func (service *Service) GetAppLocation() appdata.Location { return service.location.State() }

func (service *Service) ConfirmAppLocation(mode appdata.Mode) (bool, error) {
	restarting, err := service.location.Confirm(mode)
	if err != nil || restarting {
		return restarting, err
	}
	return false, service.startCore()
}

func (service *Service) MoveAppData(mode appdata.Mode) error {
	return service.location.Move(mode)
}
