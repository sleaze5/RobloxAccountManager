//go:build unix

package gamelaunch

func PlayerVersions() []PlayerVersion {
	return []PlayerVersion{}
}

func CheckPlayerPath(string) (string, bool, error) {
	return "", false, &ClientError{Message: "custom Roblox Player paths are only available on Windows"}
}
