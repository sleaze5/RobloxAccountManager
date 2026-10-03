// Package appmeta defines the canonical application identity used by Go code.
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

// version is the only source of the application version. The frontend and
// the Windows version resource read the same file at build time.
//
//go:embed VERSION
var version string

var (
	Version   = strings.TrimSpace(version)
	UserAgent = Name + "/" + Version
)
