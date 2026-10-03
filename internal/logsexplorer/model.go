// Package logsexplorer reads on-demand snapshots of Roblox Player logs.
package logsexplorer

type VisitKind string

type EndReason string

type LaunchMode string

type LaunchSource string

const (
	LaunchCold       LaunchMode = "cold"
	LaunchWarm       LaunchMode = "warm"
	LaunchTray       LaunchMode = "tray"
	LaunchTrayResume LaunchMode = "tray-resume"
)

const (
	SourceWebsite        LaunchSource = "website"
	SourceExistingClient LaunchSource = "existing-client"
	SourceTray           LaunchSource = "tray"
)

const (
	VisitJoin     VisitKind = "join"
	VisitTeleport VisitKind = "teleport"
	VisitRejoin   VisitKind = "rejoin"
)

const (
	EndTeleport         EndReason = "teleport"
	EndRejoin           EndReason = "rejoin"
	EndLeave            EndReason = "leave"
	EndShutdown         EndReason = "shutdown"
	EndClientDisconnect EndReason = "client-disconnect"
	EndTimeout          EndReason = "timeout"
	EndDisconnect       EndReason = "disconnect"
	EndNextJoin         EndReason = "next-join"
)

type Visit struct {
	Kind             VisitKind `json:"kind"`
	StartedAtMs      int64     `json:"startedAtMs"`
	EndedAtMs        int64     `json:"endedAtMs"`
	DurationMs       int64     `json:"durationMs"`
	EndReason        EndReason `json:"endReason"`
	EndEstimated     bool      `json:"endEstimated"`
	DisconnectCode   string    `json:"disconnectCode"`
	DisconnectReason string    `json:"disconnectReason"`
	PlaceID          string    `json:"placeId"`
	JobID            string    `json:"jobId"`
	UniverseID       string    `json:"universeId"`
	ServerAddress    string    `json:"serverAddress"`
	DatacenterID     string    `json:"datacenterId"`
	ReferralPage     string    `json:"referralPage"`
	JoinSource       string    `json:"joinSource"`
	RequestType      string    `json:"requestType"`
}

type Session struct {
	FileName     string       `json:"fileName"`
	StartedAtMs  int64        `json:"startedAtMs"`
	EndedAtMs    int64        `json:"endedAtMs"`
	LifetimeMs   int64        `json:"lifetimeMs"`
	EndEstimated bool         `json:"endEstimated"`
	LaunchMode   LaunchMode   `json:"launchMode"`
	LaunchSource LaunchSource `json:"launchSource"`
	SizeBytes    int64        `json:"sizeBytes"`
	Version      string       `json:"version"`
	Channel      string       `json:"channel"`
	UserID       string       `json:"userId"`
	Visits       []Visit      `json:"visits"`
	Issue        string       `json:"issue"`
}

type Snapshot struct {
	Sessions []Session `json:"sessions"`
}
