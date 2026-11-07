//go:build !linux

package gamelaunch

func listClients() []ClientInfo {
	return []ClientInfo{}
}
