package appmeta

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
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
	GoVersion    string       `json:"go_version"`
	Dependencies []Dependency `json:"dependencies"`
	OS           string       `json:"os"`
	Architecture string       `json:"architecture"`
	Revision     string       `json:"revision"`
	RevisionTime int64        `json:"revision_time,omitempty"`
	Modified     *bool        `json:"modified,omitempty"`
	BuildTime    int64        `json:"build_time,omitempty"`
}

func CurrentBuild() Build {
	build := Build{Version: Version, GoVersion: runtime.Version(), OS: runtime.GOOS,
		Architecture: runtime.GOARCH, Revision: "unavailable", Dependencies: []Dependency{}}
	if seconds, err := strconv.ParseInt(buildTime, 10, 64); err == nil && seconds > 0 {
		build.BuildTime = seconds
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return build
	}
	for _, path := range reportedModules {
		for _, dependency := range info.Deps {
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
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			build.Revision = setting.Value
		case "vcs.time":
			if value, err := time.Parse(time.RFC3339, setting.Value); err == nil {
				build.RevisionTime = value.UnixMilli()
			}
		case "vcs.modified":
			modified := setting.Value == "true"
			build.Modified = &modified
		}
	}
	return build
}
