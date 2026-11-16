package appservice

import (
	"context"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

// readRenderedModel reads the manifest address with readManifestURL, then
// downloads the model. Roblox renders a model on the first request and
// answers Pending until it is ready, so the read repeats a bounded number of
// times.
func (service *Service) readRenderedModel(ctx context.Context, accountID int64, operation string, readManifestURL func(context.Context, int64) (string, error)) (robloxservices.RenderedModel, error) {
	const (
		maxAttempts = 8
		retryDelay  = 2 * time.Second
	)
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, operation)
	if err != nil {
		return robloxservices.RenderedModel{}, err
	}
	defer done()
	logger := service.logs.Module("application.rendered-model").With("operation", operation, "account_id", accountID)
	manifestURL := ""
	isRendered, err := pollUntilReady(ctx, operation, "The 3D model was not loaded.", maxAttempts, retryDelay, func() (bool, error) {
		var err error
		manifestURL, err = readManifestURL(ctx, version)
		if err != nil {
			logger.WarnContext(ctx, "rendered model unavailable", "error", err)
			return false, err
		}
		return manifestURL != "", nil
	})
	if err != nil {
		return robloxservices.RenderedModel{}, err
	}
	if !isRendered {
		return robloxservices.RenderedModel{}, &roblox.Error{Kind: roblox.KindTimeout, Endpoint: operation, Message: "Roblox is still rendering the 3D model. Try again in a minute."}
	}
	model, err := service.users.RenderedModel(ctx, manifestURL)
	if err != nil {
		logger.WarnContext(ctx, "rendered model files unavailable", "error", err)
		return robloxservices.RenderedModel{}, err
	}
	return model, nil
}
