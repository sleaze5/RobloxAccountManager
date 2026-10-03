package robloxmulti

type Process struct {
	Name      string `json:"name"`
	PID       uint32 `json:"pid"`
	StartTime string `json:"startTime"`
}

type ProcessSnapshot struct {
	Supported bool      `json:"supported"`
	Processes []Process `json:"processes"`
}
