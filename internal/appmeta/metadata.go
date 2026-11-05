package appmeta

import (
	"cmp"
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
var releaseVersion string

// The build sets commit for every build that is not a release.
var commit string

var (
	Release   = commit == ""
	Version   = cmp.Or(commit, strings.TrimSpace(releaseVersion))
	UserAgent = Name + "/" + Version
)

//go:embed updater.key.pub
var UpdaterPublicKey []byte
