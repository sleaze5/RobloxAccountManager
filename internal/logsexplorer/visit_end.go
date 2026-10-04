package logsexplorer

import (
	"regexp"
	"strings"
)

var (
	disconnectPattern = regexp.MustCompile(`^\s*(?:Player:\s*)?([0-9]+)(?:\s*\(([A-Za-z][A-Za-z0-9_]*)\))?`)
	ackTimeoutPattern = regexp.MustCompile(`AckTimeout\s+([0-9]+)`)
)

type visitEndState struct {
	confirmedAt     int64
	candidateAt     int64
	disconnected    bool
	clientInitiated bool
	teleported      bool
	left            bool
	shutdown        bool
	timedOut        bool
	code            string
	reason          string
}

func (ending *visitEndState) record(line string, timestamp, startedAt int64) {
	if timestamp < startedAt {
		return
	}
	switch {
	case strings.Contains(line, "Disconnected from server for reason:"):
		if !ending.disconnected {
			ending.disconnected = true
			ending.confirmedAt = timestamp
			ending.readReason(line, "Disconnected from server for reason:")
		}
	case strings.Contains(line, "Sending disconnect with reason:"):
		if !ending.disconnected {
			ending.readReason(line, "Sending disconnect with reason:")
		}
	case strings.Contains(line, "Teleported."):
		ending.teleported = true
	case strings.Contains(line, "leaveUGCGameInternal"):
		if !ending.disconnected || timestamp <= ending.confirmedAt {
			ending.left = true
		}
	case strings.Contains(line, "shutDown: (stage:UGCGame)"):
		if !ending.disconnected || timestamp <= ending.confirmedAt {
			ending.shutdown = true
		}
	case strings.Contains(line, "Connection lost"):
		if timeout := capture(ackTimeoutPattern, line); timeout != "" && strings.TrimLeft(timeout, "0") != "" {
			ending.timedOut = true
		}
	case strings.Contains(line, "DisconnectClientInitiated"):
		ending.clientInitiated = true
	default:
		return
	}
	if ending.candidateAt == 0 {
		ending.candidateAt = timestamp
	}
}

func (ending *visitEndState) readReason(line, marker string) {
	_, value, _ := strings.Cut(line, marker)
	if match := disconnectPattern.FindStringSubmatch(value); match != nil {
		ending.code = match[1]
		ending.reason = match[2]
		ending.clientInitiated = match[2] == "DisconnectClientInitiated"
	}
}

func (ending visitEndState) finish(visit *Visit, nextAt int64, nextPlace string) {
	visit.EndedAtMs = ending.confirmedAt
	if visit.EndedAtMs == 0 {
		visit.EndedAtMs = ending.candidateAt
		if visit.EndedAtMs == 0 && nextAt >= visit.StartedAtMs {
			visit.EndedAtMs = nextAt
		}
		visit.EndEstimated = visit.EndedAtMs > 0
	}
	visit.EndReason = ending.endReason(visit.PlaceID, nextPlace)
	visit.DisconnectCode = ending.code
	visit.DisconnectReason = ending.reason
	if visit.StartedAtMs > 0 && visit.EndedAtMs >= visit.StartedAtMs {
		visit.DurationMs = visit.EndedAtMs - visit.StartedAtMs
	}
}

func (ending visitEndState) endReason(place, nextPlace string) EndReason {
	switch {
	case ending.teleported:
		if place == nextPlace {
			return EndRejoin
		}
		return EndTeleport
	case ending.disconnected && ending.reason != "" && !ending.clientInitiated:
		if ending.timedOut {
			return EndTimeout
		}
		return EndDisconnect
	case ending.shutdown:
		return EndShutdown
	case ending.left:
		return EndLeave
	case ending.clientInitiated:
		return EndClientDisconnect
	case ending.timedOut:
		return EndTimeout
	case ending.disconnected || ending.candidateAt > 0:
		return EndDisconnect
	case nextPlace != "":
		return EndNextJoin
	default:
		return ""
	}
}
