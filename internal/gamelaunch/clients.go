package gamelaunch

import "runtime"

// ClientInfo describes one Linux Roblox client the application can open.
type ClientInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// ClientState is the Linux client choice shown in settings, plus whether
// Windows-only direct launch is available.
type ClientState struct {
	ChoiceSupported bool         `json:"choiceSupported"`
	DirectLaunch    bool         `json:"directLaunch"`
	Selected        string       `json:"selected"`
	Clients         []ClientInfo `json:"clients"`
}

func CurrentClients(selected string) ClientState {
	clients := listClients()
	if clients == nil {
		clients = []ClientInfo{}
	}
	return ClientState{
		ChoiceSupported: runtime.GOOS == "linux",
		DirectLaunch:    runtime.GOOS == "windows",
		Selected:        selected,
		Clients:         clients,
	}
}
