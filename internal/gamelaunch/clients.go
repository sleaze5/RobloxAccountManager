package gamelaunch

import "runtime"

type ClientInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

type PlayerVersion struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DisplayPath string `json:"displayPath"`
}

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
