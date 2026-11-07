//go:build !linux

package gamelaunch

func listClients() []RobloxClient {
	return []RobloxClient{}
}
