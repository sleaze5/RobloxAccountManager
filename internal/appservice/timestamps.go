package appservice

import "time"

func unixMillisecondsFromRFC3339(timestamp string) int64 {
	parsed, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return 0
	}
	return parsed.UnixMilli()
}
