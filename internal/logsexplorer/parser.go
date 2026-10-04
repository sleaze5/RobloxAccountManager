package logsexplorer

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

type parseState struct {
	timestamp int64
	ending    visitEndState
	client    clientLifecycle
	pending   joinMetadata
}

var (
	joinPattern     = regexp.MustCompile(`Joining game '([0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12})' place ([0-9]+)`)
	versionPattern  = regexp.MustCompile(`"version"\s*:\s*"([0-9.]+)"`)
	channelPattern  = regexp.MustCompile(`(?:The channel is |RobloxChannel has been set to )([a-zA-Z0-9_-]+)`)
	userPattern     = regexp.MustCompile(`(?:userId = |userid:)([0-9]+)`)
	universePattern = regexp.MustCompile(`universeid:([0-9]+)`)
	serverPattern   = regexp.MustCompile(`UDMUX Address = ([0-9a-fA-F.:]+), Port = ([0-9]+)`)
	filePattern     = regexp.MustCompile(`(?i)^([0-9.]+)_([0-9]{8}T[0-9]{6}Z)_Player_`)
)

func newSession(name string) Session {
	session := Session{FileName: name, Visits: []Visit{}}
	if match := filePattern.FindStringSubmatch(name); match != nil {
		session.Version = match[1]
		if started, err := time.Parse("20060102T150405Z", match[2]); err == nil {
			session.StartedAtMs = started.UnixMilli()
		}
	}
	return session
}

func parse(ctx context.Context, input io.Reader, session *Session) error {
	reader := bufio.NewReaderSize(input, 64*1024)
	var state parseState
	defer func() {
		if len(session.Visits) > 0 {
			state.ending.finish(&session.Visits[len(session.Visits)-1], 0, "")
		}
		state.client.finish(session)
	}()
	skipping := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			skipping = true
			session.Issue = "Some oversized log lines were skipped. Session details may be incomplete."
			continue
		}
		if !skipping {
			state.parseLine(strings.TrimSpace(string(line)), session)
		}
		skipping = false
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (state *parseState) parseLine(line string, session *Session) {
	if prefix, _, ok := strings.Cut(line, ","); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, prefix); err == nil {
			if state.timestamp == 0 {
				session.StartedAtMs = parsed.UnixMilli()
			}
			state.timestamp = parsed.UnixMilli()
			state.client.record(line, state.timestamp)
		}
	}
	parseMetadata(line, session)
	if len(session.Visits) == 0 {
		parseLaunch(line, session)
	}
	state.readJoinContext(line, session)
	if strings.Contains(line, "[FLog::Output]") && strings.Contains(line, "Joining game '") {
		if match := joinPattern.FindStringSubmatch(line); match != nil {
			state.appendVisit(session, match[1], match[2])
		}
	}
	if len(session.Visits) == 0 {
		return
	}
	visit := &session.Visits[len(session.Visits)-1]
	state.ending.record(line, state.timestamp, visit.StartedAtMs)
	if strings.Contains(line, "[FLog::GameJoinLoadTime]") {
		if value := capture(universePattern, line); value != "" {
			visit.UniverseID = value
		}
		if value := capture(referralPattern, line); value != "" {
			visit.ReferralPage = value
		}
	}
	if strings.Contains(line, "[DFLog::NetworkClient]") || strings.Contains(line, "[FLog::NetworkClient]") {
		if value := capture(datacenterPattern, line); value != "" {
			visit.DatacenterID = value
		}
	}
	if strings.Contains(line, "[FLog::Network]") {
		if match := serverPattern.FindStringSubmatch(line); match != nil {
			visit.ServerAddress = net.JoinHostPort(match[1], match[2])
		}
	}
}

func (state *parseState) appendVisit(session *Session, jobID, placeID string) {
	kind := VisitJoin
	if len(session.Visits) > 0 {
		previous := &session.Visits[len(session.Visits)-1]
		if state.ending.teleported || (!state.ending.left && !state.ending.shutdown) {
			kind = VisitTeleport
			if previous.PlaceID == placeID {
				kind = VisitRejoin
			}
		}
		state.ending.finish(previous, state.timestamp, placeID)
	}
	visit := Visit{Kind: kind, StartedAtMs: state.timestamp, JobID: jobID, PlaceID: placeID}
	state.pending.apply(&visit)
	state.pending = joinMetadata{}
	state.ending = visitEndState{}
	session.Visits = append(session.Visits, visit)
}

func parseMetadata(line string, session *Session) {
	if strings.Contains(line, "The channel is ") || strings.Contains(line, "RobloxChannel has been set to ") {
		if value := capture(channelPattern, line); value != "" {
			session.Channel = value
		}
	}
	if session.Version == "" && strings.Contains(line, `"version"`) {
		if value := capture(versionPattern, line); value != "" {
			session.Version = value
		}
	}
	if session.UserID == "" && (strings.Contains(line, "userId = ") || strings.Contains(line, "[FLog::GameJoinLoadTime]")) {
		session.UserID = capture(userPattern, line)
	}
}

func capture(pattern *regexp.Regexp, line string) string {
	if match := pattern.FindStringSubmatch(line); match != nil {
		return match[1]
	}
	return ""
}
