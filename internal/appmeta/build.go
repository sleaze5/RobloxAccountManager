package appmeta

import (
	"runtime"
	"runtime/debug"
	"strconv"
)

var buildTime string

var reportedModules = []string{
	"github.com/wailsapp/wails/v3",
}

type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Build struct {
	Version      string       `json:"version"`
	Release      bool         `json:"release"`
	GoVersion    string       `json:"go_version"`
	Dependencies []Dependency `json:"dependencies"`
	OS           string       `json:"os"`
	Architecture string       `json:"architecture"`
	BuildTime    int64        `json:"build_time,omitempty"`
}

func CurrentBuild() Build {
	build := Build{Version: Version, Release: Release, GoVersion: runtime.Version(), OS: runtime.GOOS,
		Architecture: runtime.GOARCH, Dependencies: []Dependency{}}
	if seconds, err := strconv.ParseInt(buildTime, 10, 64); err == nil && seconds > 0 {
		build.BuildTime = seconds
	}
	embeddedBuild, ok := debug.ReadBuildInfo()
	if !ok {
		return build
	}
	for _, path := range reportedModules {
		for _, dependency := range embeddedBuild.Deps {
			if dependency.Path != path {
				continue
			}
			version := dependency.Version
			if dependency.Replace != nil {
				version = dependency.Replace.Version + " (replacement)"
			}
			build.Dependencies = append(build.Dependencies, Dependency{Name: path, Version: version})
		}
	}
	return build
}
