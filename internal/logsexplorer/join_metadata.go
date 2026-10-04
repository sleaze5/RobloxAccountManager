package logsexplorer

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	datacenterPattern    = regexp.MustCompile(`DatacenterId=([0-9]+)`)
	referralPattern      = regexp.MustCompile(`referral_page:\s*([A-Za-z0-9_-]+)`)
	metadataPattern      = regexp.MustCompile(`(?i)"?GameJoinMetadata"?\s*[:=]\s*(\{[^}]*\})`)
	metadataEndPattern   = regexp.MustCompile(`(?i)}|%7d`)
	joinSourcePattern    = regexp.MustCompile(`(?i)"?JoinSource"?\s*[:=]\s*"?([A-Za-z0-9_-]+)`)
	requestTypePattern   = regexp.MustCompile(`(?i)"?RequestType"?\s*[:=]\s*"?([A-Za-z0-9_-]+)`)
	metadataPlacePattern = regexp.MustCompile(`(?i)(?:"|%22)?PlaceId(?:"|%22)?\s*(?:[:=]|%3a|%3d)\s*(?:"|%22)?([0-9]+)`)
)

type joinMetadata struct {
	placeID     string
	joinSource  string
	requestType string
}

func (state *parseState) readJoinContext(line string, session *Session) {
	if strings.Contains(line, "raiseTeleportInitFailedEvent") {
		state.pending = joinMetadata{}
		return
	}
	metadata := readMetadataObject(line)
	if metadata == "" {
		return
	}
	source, request := capture(joinSourcePattern, metadata), capture(requestTypePattern, metadata)
	if strings.Contains(line, "[FLog::GameJoinLoadTime]") && len(session.Visits) > 0 {
		visit := &session.Visits[len(session.Visits)-1]
		visit.JoinSource, visit.RequestType = source, request
		return
	}
	state.pending.placeID = capture(metadataPlacePattern, line)
	state.pending.joinSource, state.pending.requestType = source, request
}

func readMetadataObject(line string) string {
	_, tail, found := strings.Cut(line, "GameJoinMetadata")
	if !found {
		return ""
	}
	end := metadataEndPattern.FindStringIndex(tail)
	if end == nil {
		return ""
	}
	decoded, err := url.QueryUnescape("GameJoinMetadata" + tail[:end[1]])
	if err != nil {
		return ""
	}
	return capture(metadataPattern, decoded)
}

func (metadata joinMetadata) apply(visit *Visit) {
	if metadata.placeID == "" || metadata.placeID == visit.PlaceID {
		visit.JoinSource = metadata.joinSource
		visit.RequestType = metadata.requestType
	}
}
