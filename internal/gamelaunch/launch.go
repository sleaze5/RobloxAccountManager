package gamelaunch

import (
	"context"
	"errors"
)

var ErrDesktopUnavailable = errors.New("a non-admin Windows desktop is unavailable")

type ShareType string

const (
	ShareServer           ShareType = "Server"
	ShareExperienceInvite ShareType = "ExperienceInvite"
)

type Request struct {
	PlaceID      int64
	JobID        string
	UserID       int64
	Username     string
	ShareCode    string
	ShareType    ShareType
	LinkCode     string
	AccessCode   string
	Teleport     bool
	DirectLaunch bool
	LaunchData   string
}

type Launcher interface {
	Launch(context.Context, string, string, Request) error
	LaunchProtocol(context.Context, string) error
	CommandLine(context.Context, string, string, Request) (string, error)
}
