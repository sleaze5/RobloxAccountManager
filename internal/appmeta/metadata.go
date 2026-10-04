package appmeta

import (
	_ "embed"
	"strings"
)

const (
	Name        = "RobloxAccountManager"
	DisplayName = "Roblox Account Manager"
	Description = "A portable desktop application for storing and managing Roblox accounts."
	Identifier  = "com.github.sleaze5.robloxaccountmanager"
)

//go:embed VERSION
var version string

var (
	Version   = strings.TrimSpace(version)
	UserAgent = Name + "/" + Version
)

//go:embed updater.key.pub
var UpdaterPublicKey []byte
