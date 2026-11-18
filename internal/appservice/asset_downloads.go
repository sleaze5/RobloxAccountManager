package appservice

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sleaze5/RobloxAccountManager/internal/appdata"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
	robloxservices "github.com/sleaze5/RobloxAccountManager/internal/roblox/services"
)

const maxDownloadNameLength = 100

// DownloadableFile is the file of an item in a format that other programs
// open, with the extension and the name of that format.
type DownloadableFile struct {
	Contents   []byte
	Extension  string
	FormatName string
}

var formatNameByAssetFileFormat = map[robloxservices.AssetFileFormat]string{
	robloxservices.AssetFilePNG:   "PNG image",
	robloxservices.AssetFileJPEG:  "JPEG image",
	robloxservices.AssetFileRBXM:  "Roblox model",
	robloxservices.AssetFileRBXMX: "Roblox XML model",
	robloxservices.AssetFileOGG:   "OGG audio",
	robloxservices.AssetFileMP3:   "MP3 audio",
}

// ReadInventoryItemFile downloads the file of the item. Other programs cannot
// open a mesh in the format of Roblox, so a mesh is saved as the OBJ model
// that Roblox renders of it.
func (service *Service) ReadInventoryItemFile(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (DownloadableFile, error) {
	assetFile, err := service.readAssetFile(ctx, accountID, category, itemID)
	if err != nil {
		return DownloadableFile{}, err
	}
	if assetFile.Format != robloxservices.AssetFileRobloxMesh {
		return DownloadableFile{Contents: assetFile.Contents, Extension: string(assetFile.Format), FormatName: formatNameByAssetFileFormat[assetFile.Format]}, nil
	}
	model, err := service.readRenderedModel(ctx, accountID, "asset-file-model", func(ctx context.Context, version int64) (string, error) {
		return service.users.AssetModelManifestURL(ctx, accountID, version, itemID)
	})
	if err != nil {
		return DownloadableFile{}, err
	}
	return DownloadableFile{Contents: []byte(model.Mesh), Extension: "obj", FormatName: "OBJ model"}, nil
}

func (service *Service) readAssetFile(ctx context.Context, accountID int64, category robloxservices.InventoryCategory, itemID int64) (robloxservices.AssetFile, error) {
	ctx, version, done, err := service.beginAccountSession(ctx, accountID, "asset-file")
	if err != nil {
		return robloxservices.AssetFile{}, err
	}
	defer done()
	assetFile, err := service.users.AssetFile(ctx, accountID, version, category, itemID)
	if err != nil {
		service.logs.Module("application.inventory").WarnContext(ctx, "asset file unavailable", "operation", "asset-file", "account_id", accountID, "category", category, "error", err)
		return robloxservices.AssetFile{}, err
	}
	return assetFile, nil
}

// DefaultDownloadDirectory is where the save dialog for a download opens.
func (service *Service) DefaultDownloadDirectory() string {
	return appdata.DefaultDownloadDirectory()
}

// DownloadFileName names the file of an item after the item. An item without
// a usable name is named after its ID.
func (service *Service) DownloadFileName(itemName string, itemID int64, extension string) string {
	return fileNameOf(itemName, fmt.Sprintf("Item %d", itemID), extension)
}

// fileNameOf names a file after a name, without the characters that a file
// system rejects. A name with no usable characters gives the fallback name.
func fileNameOf(name, fallbackName, extension string) string {
	cleanedName := strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || strings.ContainsRune(`<>:"/\|?*`, character) {
			return '_'
		}
		return character
	}, name)
	cleanedName = strings.Trim(strings.TrimSpace(cleanedName), ".")
	if utf8.RuneCountInString(cleanedName) > maxDownloadNameLength {
		cleanedName = strings.TrimSpace(string([]rune(cleanedName)[:maxDownloadNameLength]))
	}
	if cleanedName == "" {
		cleanedName = fallbackName
	}
	return cleanedName + "." + extension
}

// SaveDownloadedFile writes the file of an item where the user chose.
func (service *Service) SaveDownloadedFile(path, extension string, contents []byte) (string, error) {
	savedPath, err := service.saveChosenFile(path, extension, contents, "asset-download", "application.inventory")
	if err == nil {
		service.logs.Module("application.inventory").Info("item file saved", "operation", "asset-download", "extension", extension)
	}
	return savedPath, err
}

// saveChosenFile writes a file where the user chose, with the extension added
// when the save dialog left it out. The save dialog confirmed replacing only
// the path it returned, so a path with an added extension must not exist.
func (service *Service) saveChosenFile(path, extension string, contents []byte, operation, logModule string) (string, error) {
	fileFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !strings.EqualFold(filepath.Ext(path), "."+extension) {
		path += "." + extension
		fileFlags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	if err := writeDownloadedFile(path, fileFlags, contents); errors.Is(err, fs.ErrExist) {
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: fmt.Sprintf("%s already exists. Select another name, then try again.", filepath.Base(path)), Cause: err}
	} else if err != nil {
		service.logs.Module(logModule).Warn("could not write the chosen file", "operation", operation, "error", err)
		return "", &roblox.Error{Kind: roblox.KindProtocol, Endpoint: operation, Message: "Could not save the file. Check that the folder is writable and has free space, then try again.", Cause: err}
	}
	return path, nil
}

func writeDownloadedFile(path string, fileFlags int, contents []byte) error {
	file, err := os.OpenFile(path, fileFlags, 0o600)
	if err != nil {
		return fmt.Errorf("open downloaded file: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return fmt.Errorf("write downloaded file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close downloaded file: %w", err)
	}
	return nil
}
