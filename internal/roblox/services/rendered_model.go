package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxRenderedModelManifestBytes = 64 << 10
	maxRenderedModelMeshBytes     = 16 << 20
	maxRenderedModelMaterialBytes = 256 << 10
	maxRenderedModelTextureBytes  = 4 << 20
	maxRenderedModelTextureCount  = 32
	maxRenderedModelFileHashBytes = 64
	renderedModelCDNHostCount     = 8
)

// RenderedModelVector is a point or a direction in the space of the model.
type RenderedModelVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// RenderedModelCamera is where Roblox places the camera for the model.
type RenderedModelCamera struct {
	Position           RenderedModelVector `json:"position"`
	FieldOfViewDegrees float64             `json:"fieldOfViewDegrees"`
}

// RenderedModelBounds is the axis-aligned box around the model.
type RenderedModelBounds struct {
	Minimum RenderedModelVector `json:"minimum"`
	Maximum RenderedModelVector `json:"maximum"`
}

// RenderedModel is a 3D model that Roblox renders, such as an avatar or an
// item, as Wavefront OBJ and MTL text. The MTL names each texture by its file
// hash, and TextureDataURLByFileHash holds every texture as a data URL.
type RenderedModel struct {
	Camera                   RenderedModelCamera `json:"camera"`
	Bounds                   RenderedModelBounds `json:"bounds"`
	Mesh                     string              `json:"mesh"`
	Materials                string              `json:"materials"`
	TextureDataURLByFileHash map[string]string   `json:"textureDataUrlByFileHash"`
}

// renderedModelManifestURL reads the address of a 3D model manifest from a
// thumbnails endpoint, or an empty address while Roblox renders the model.
// These endpoints require an authenticated session.
func (service *Users) renderedModelManifestURL(ctx context.Context, accountID, version int64, endpoint, address string) (string, error) {
	var payload struct {
		State    string `json:"state"`
		ImageURL string `json:"imageUrl"`
	}
	if err := service.readAccountDetails(ctx, accountID, version, endpoint, address, &payload); err != nil {
		return "", err
	}
	switch {
	case strings.EqualFold(payload.State, "Pending"):
		return "", nil
	case !strings.EqualFold(payload.State, "Completed"):
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: endpoint, Message: "Roblox can't show this in 3D."}
	case !safeThumbnailURL(payload.ImageURL):
		return "", invalidRenderedModel()
	}
	return payload.ImageURL, nil
}

// AssetModelManifestURL reads the address of the 3D model manifest of the
// asset, or an empty address while Roblox renders the model.
func (service *Users) AssetModelManifestURL(ctx context.Context, accountID, version, assetID int64) (string, error) {
	return service.renderedModelManifestURL(ctx, accountID, version, "asset-model", "https://thumbnails.roblox.com/v1/assets-thumbnail-3d?assetId="+strconv.FormatInt(assetID, 10))
}

// renderedModelManifest names the files of the model by hash, such as
// 30DAY- followed by hex digits.
type renderedModelManifest struct {
	Camera *struct {
		Position    RenderedModelVector `json:"position"`
		FieldOfView float64             `json:"fov"`
	} `json:"camera"`
	BoundingBox *struct {
		Minimum RenderedModelVector `json:"min"`
		Maximum RenderedModelVector `json:"max"`
	} `json:"aabb"`
	MeshFileHash      string   `json:"obj"`
	MaterialFileHash  string   `json:"mtl"`
	TextureFileHashes []string `json:"textures"`
}

func decodeRenderedModelManifest(manifestBody []byte) (renderedModelManifest, error) {
	var manifest renderedModelManifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil || manifest.Camera == nil || manifest.BoundingBox == nil {
		return renderedModelManifest{}, invalidRenderedModel()
	}
	if !isValidRenderedModelFileHash(manifest.MeshFileHash) || !isValidRenderedModelFileHash(manifest.MaterialFileHash) || len(manifest.TextureFileHashes) > maxRenderedModelTextureCount {
		return renderedModelManifest{}, invalidRenderedModel()
	}
	for _, textureFileHash := range manifest.TextureFileHashes {
		if !isValidRenderedModelFileHash(textureFileHash) {
			return renderedModelManifest{}, invalidRenderedModel()
		}
	}
	return manifest, nil
}

// RenderedModel downloads the manifest and the files of a model from the
// Roblox CDN, which serves them without credentials.
func (service *Users) RenderedModel(ctx context.Context, manifestURL string) (RenderedModel, error) {
	manifestBody, err := service.downloadCDNFile(ctx, manifestURL, maxRenderedModelManifestBytes)
	if err != nil {
		return RenderedModel{}, err
	}
	manifest, err := decodeRenderedModelManifest(manifestBody)
	if err != nil {
		return RenderedModel{}, err
	}
	model := RenderedModel{
		Camera:                   RenderedModelCamera{Position: manifest.Camera.Position, FieldOfViewDegrees: manifest.Camera.FieldOfView},
		Bounds:                   RenderedModelBounds{Minimum: manifest.BoundingBox.Minimum, Maximum: manifest.BoundingBox.Maximum},
		TextureDataURLByFileHash: make(map[string]string, len(manifest.TextureFileHashes)),
	}
	if !model.hasFiniteGeometry() {
		return RenderedModel{}, invalidRenderedModel()
	}
	if err := service.downloadRenderedModelFiles(ctx, manifest, &model); err != nil {
		return RenderedModel{}, err
	}
	return model, nil
}

// downloadRenderedModelFiles downloads the mesh, the materials, and the textures
// at once, and stops the other downloads when one fails.
func (service *Users) downloadRenderedModelFiles(ctx context.Context, manifest renderedModelManifest, model *RenderedModel) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		group      sync.WaitGroup
		modelMutex sync.Mutex
		firstError error
	)
	download := func(fileHash string, maxResponseSizeBytes int64, addToModel func([]byte) error) {
		group.Go(func() {
			fileBody, err := service.downloadCDNFile(ctx, renderedModelFileURL(fileHash), maxResponseSizeBytes)
			modelMutex.Lock()
			defer modelMutex.Unlock()
			if err == nil {
				err = addToModel(fileBody)
			}
			if err != nil && firstError == nil {
				firstError = err
				cancel()
			}
		})
	}
	download(manifest.MeshFileHash, maxRenderedModelMeshBytes, func(meshBody []byte) error {
		model.Mesh = string(meshBody)
		return nil
	})
	download(manifest.MaterialFileHash, maxRenderedModelMaterialBytes, func(materialBody []byte) error {
		model.Materials = string(materialBody)
		return nil
	})
	for _, textureFileHash := range manifest.TextureFileHashes {
		download(textureFileHash, maxRenderedModelTextureBytes, func(textureBody []byte) error {
			contentType := http.DetectContentType(textureBody)
			if contentType != "image/png" && contentType != "image/jpeg" {
				return invalidRenderedModel()
			}
			model.TextureDataURLByFileHash[textureFileHash] = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(textureBody)
			return nil
		})
	}
	group.Wait()
	return firstError
}

func (model RenderedModel) hasFiniteGeometry() bool {
	for _, number := range []float64{
		model.Camera.Position.X, model.Camera.Position.Y, model.Camera.Position.Z, model.Camera.FieldOfViewDegrees,
		model.Bounds.Minimum.X, model.Bounds.Minimum.Y, model.Bounds.Minimum.Z,
		model.Bounds.Maximum.X, model.Bounds.Maximum.Y, model.Bounds.Maximum.Z,
	} {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return false
		}
	}
	return true
}

// downloadCDNFile downloads a file from the Roblox CDN, or from the
// content delivery host on roblox.com that the asset delivery API names, both
// of which serve files without credentials.
func (service *Users) downloadCDNFile(ctx context.Context, address string, maxResponseSizeBytes int64) ([]byte, error) {
	fileURL, err := url.Parse(address)
	if err != nil || (!safeThumbnailURL(address) && !roblox.IsRobloxURL(fileURL)) {
		return nil, &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "cdn-file", Message: "Roblox returned an invalid download address."}
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "cdn-file", Method: http.MethodGet, URL: fileURL, MaxResponseSize: maxResponseSizeBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

// renderedModelFileURL names the CDN host that serves a file. Roblox spreads the
// files over the hosts t0 to t7 by an XOR of the characters of the hash,
// starting from 31.
func renderedModelFileURL(fileHash string) string {
	hostIndex := 31
	for _, character := range []byte(fileHash) {
		hostIndex ^= int(character)
	}
	return "https://t" + strconv.Itoa(hostIndex%renderedModelCDNHostCount) + ".rbxcdn.com/" + fileHash
}

// isValidRenderedModelFileHash accepts only letters, digits, and hyphens, so
// that a hash can only name a file at the root of a CDN host.
func isValidRenderedModelFileHash(fileHash string) bool {
	if fileHash == "" || len(fileHash) > maxRenderedModelFileHashBytes {
		return false
	}
	for _, character := range fileHash {
		isLetterOrDigit := (character >= '0' && character <= '9') || (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
		if !isLetterOrDigit && character != '-' {
			return false
		}
	}
	return true
}

func invalidRenderedModel() error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "rendered-model", Message: "Roblox returned an invalid 3D model. Try again later."}
}
