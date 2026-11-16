package bindings

import (
	"context"

	"github.com/sleaze5/RobloxAccountManager/internal/appservice"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	"github.com/sleaze5/RobloxAccountManager/internal/storage/vault"
)

func (service *Service) ExportVault(ctx context.Context, password, confirmation, hint string) (vault.ExportFile, error) {
	if err := service.core.ValidateExportPassword(password, confirmation, hint); err != nil {
		return vault.ExportFile{}, err
	}
	path, err := chooseSaveFile(service.core.ExportsDirectory(), service.core.DefaultExportName(), exportFilterName, exportFilterPattern)
	if err != nil {
		return vault.ExportFile{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "The save dialog could not be opened.", Cause: err}
	}
	if path == "" {
		return vault.ExportFile{}, nil
	}
	return service.core.ExportVault(ctx, path, password, confirmation, hint)
}

func (service *Service) ChooseVaultExport() (string, error) {
	path, err := chooseOpenFile(service.core.ExportsDirectory())
	if err != nil {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "The open dialog could not be opened.", Cause: err}
	}
	return path, nil
}

func (service *Service) GetExportDetails(path string) (vault.ExportDetails, error) {
	return service.core.ExportDetails(path)
}

func (service *Service) OpenVaultExport(ctx context.Context, path, password string) (appservice.TransferPreview, error) {
	return service.core.OpenVaultExport(ctx, path, password)
}

func (service *Service) ImportVaultExport(ctx context.Context, transferID string, request appservice.TransferRequest, confirmation string) (appservice.TransferResult, error) {
	if request.Mode == appservice.TransferOverwrite && confirmation != "CONFIRM" {
		return appservice.TransferResult{}, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "vault-transfer", Message: "Type CONFIRM to overwrite the vault."}
	}
	return service.core.ImportVaultExport(ctx, transferID, request)
}

func (service *Service) DiscardVaultExport(transferID string) {
	service.core.DiscardVaultExport(transferID)
}
