package appservice

import "time"

// unixMillisecondsFromRFC3339 returns 0 for a timestamp that Roblox left empty
// or wrote in another format.
func unixMillisecondsFromRFC3339(timestamp string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return 0
	}
	return parsed.UnixMilli()
}
