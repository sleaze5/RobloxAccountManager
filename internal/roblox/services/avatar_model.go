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
	maxAvatarModelManifestBytes = 64 << 10
	maxAvatarModelMeshBytes     = 16 << 20
	maxAvatarModelMaterialBytes = 256 << 10
	maxAvatarModelTextureBytes  = 4 << 20
	maxAvatarModelTextureCount  = 32
	maxAvatarModelFileHashBytes = 64
	avatarModelCDNHostCount     = 8
)

// AvatarModelVector is a point or a direction in the space of the model.
type AvatarModelVector struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// AvatarModelCamera is where Roblox places the camera for the 3D avatar.
type AvatarModelCamera struct {
	Position           AvatarModelVector `json:"position"`
	FieldOfViewDegrees float64           `json:"fieldOfViewDegrees"`
}

// AvatarModelBounds is the axis-aligned box around the model.
type AvatarModelBounds struct {
	Minimum AvatarModelVector `json:"minimum"`
	Maximum AvatarModelVector `json:"maximum"`
}

// AvatarModel is the 3D avatar that Roblox renders, as Wavefront OBJ and MTL
// text. The MTL names each texture by its file hash, and
// TextureDataURLByFileHash holds every texture as a data URL.
type AvatarModel struct {
	Camera                   AvatarModelCamera `json:"camera"`
	Bounds                   AvatarModelBounds `json:"bounds"`
	Mesh                     string            `json:"mesh"`
	Materials                string            `json:"materials"`
	TextureDataURLByFileHash map[string]string `json:"textureDataUrlByFileHash"`
}

// AvatarModelManifestURL reads the address of the 3D avatar manifest of the
// user, or an empty address while Roblox renders the model. The endpoint
// requires an authenticated session.
func (service *Users) AvatarModelManifestURL(ctx context.Context, accountID, version, userID int64) (string, error) {
	var payload struct {
		State    string `json:"state"`
		ImageURL string `json:"imageUrl"`
	}
	address := "https://thumbnails.roblox.com/v1/users/avatar-3d?userId=" + strconv.FormatInt(userID, 10)
	if err := service.readAccountDetails(ctx, accountID, version, "avatar-model", address, &payload); err != nil {
		return "", err
	}
	if !strings.EqualFold(payload.State, "Completed") {
		return "", nil
	}
	if !safeThumbnailURL(payload.ImageURL) {
		return "", invalidAvatarModel()
	}
	return payload.ImageURL, nil
}

// avatarModelManifest names the files of the 3D avatar by hash, such as
// 30DAY- followed by hex digits.
type avatarModelManifest struct {
	Camera *struct {
		Position    AvatarModelVector `json:"position"`
		FieldOfView float64           `json:"fov"`
	} `json:"camera"`
	BoundingBox *struct {
		Minimum AvatarModelVector `json:"min"`
		Maximum AvatarModelVector `json:"max"`
	} `json:"aabb"`
	MeshFileHash      string   `json:"obj"`
	MaterialFileHash  string   `json:"mtl"`
	TextureFileHashes []string `json:"textures"`
}

func decodeAvatarModelManifest(manifestBody []byte) (avatarModelManifest, error) {
	var manifest avatarModelManifest
	if err := json.Unmarshal(manifestBody, &manifest); err != nil || manifest.Camera == nil || manifest.BoundingBox == nil {
		return avatarModelManifest{}, invalidAvatarModel()
	}
	if !isValidAvatarModelFileHash(manifest.MeshFileHash) || !isValidAvatarModelFileHash(manifest.MaterialFileHash) || len(manifest.TextureFileHashes) > maxAvatarModelTextureCount {
		return avatarModelManifest{}, invalidAvatarModel()
	}
	for _, textureFileHash := range manifest.TextureFileHashes {
		if !isValidAvatarModelFileHash(textureFileHash) {
			return avatarModelManifest{}, invalidAvatarModel()
		}
	}
	return manifest, nil
}

// AvatarModel downloads the manifest and the files of the 3D avatar from the
// Roblox CDN, which serves them without credentials.
func (service *Users) AvatarModel(ctx context.Context, manifestURL string) (AvatarModel, error) {
	manifestBody, err := service.downloadAvatarModelFile(ctx, manifestURL, maxAvatarModelManifestBytes)
	if err != nil {
		return AvatarModel{}, err
	}
	manifest, err := decodeAvatarModelManifest(manifestBody)
	if err != nil {
		return AvatarModel{}, err
	}
	model := AvatarModel{
		Camera:                   AvatarModelCamera{Position: manifest.Camera.Position, FieldOfViewDegrees: manifest.Camera.FieldOfView},
		Bounds:                   AvatarModelBounds{Minimum: manifest.BoundingBox.Minimum, Maximum: manifest.BoundingBox.Maximum},
		TextureDataURLByFileHash: make(map[string]string, len(manifest.TextureFileHashes)),
	}
	if !model.hasFiniteGeometry() {
		return AvatarModel{}, invalidAvatarModel()
	}
	if err := service.downloadAvatarModelFiles(ctx, manifest, &model); err != nil {
		return AvatarModel{}, err
	}
	return model, nil
}

// downloadAvatarModelFiles downloads the mesh, the materials, and the textures
// at once, and stops the other downloads when one fails.
func (service *Users) downloadAvatarModelFiles(ctx context.Context, manifest avatarModelManifest, model *AvatarModel) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		group      sync.WaitGroup
		modelMutex sync.Mutex
		firstError error
	)
	download := func(fileHash string, maxResponseSizeBytes int64, addToModel func([]byte) error) {
		group.Go(func() {
			fileBody, err := service.downloadAvatarModelFile(ctx, avatarModelFileURL(fileHash), maxResponseSizeBytes)
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
	download(manifest.MeshFileHash, maxAvatarModelMeshBytes, func(meshBody []byte) error {
		model.Mesh = string(meshBody)
		return nil
	})
	download(manifest.MaterialFileHash, maxAvatarModelMaterialBytes, func(materialBody []byte) error {
		model.Materials = string(materialBody)
		return nil
	})
	for _, textureFileHash := range manifest.TextureFileHashes {
		download(textureFileHash, maxAvatarModelTextureBytes, func(textureBody []byte) error {
			contentType := http.DetectContentType(textureBody)
			if contentType != "image/png" && contentType != "image/jpeg" {
				return invalidAvatarModel()
			}
			model.TextureDataURLByFileHash[textureFileHash] = "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(textureBody)
			return nil
		})
	}
	group.Wait()
	return firstError
}

func (model AvatarModel) hasFiniteGeometry() bool {
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

func (service *Users) downloadAvatarModelFile(ctx context.Context, address string, maxResponseSizeBytes int64) ([]byte, error) {
	fileURL, err := url.Parse(address)
	if err != nil || !safeThumbnailURL(address) {
		return nil, invalidAvatarModel()
	}
	response, err := service.client.Do(ctx, roblox.Request{
		Endpoint: "avatar-model-file", Method: http.MethodGet, URL: fileURL, MaxResponseSize: maxResponseSizeBytes,
		Retry: roblox.RetryPolicy{MaxAttempts: 3, Idempotent: true, RetryServerErrors: true, RetryRateLimit: true},
	})
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

// avatarModelFileURL names the CDN host that serves a file. Roblox spreads the
// files over the hosts t0 to t7 by an XOR of the characters of the hash,
// starting from 31.
func avatarModelFileURL(fileHash string) string {
	hostIndex := 31
	for _, character := range []byte(fileHash) {
		hostIndex ^= int(character)
	}
	return "https://t" + strconv.Itoa(hostIndex%avatarModelCDNHostCount) + ".rbxcdn.com/" + fileHash
}

// isValidAvatarModelFileHash accepts only letters, digits, and hyphens, so
// that a hash can only name a file at the root of a CDN host.
func isValidAvatarModelFileHash(fileHash string) bool {
	if fileHash == "" || len(fileHash) > maxAvatarModelFileHashBytes {
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

func invalidAvatarModel() error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: "avatar-model", Message: "Roblox returned an invalid 3D avatar. Try again later."}
}
