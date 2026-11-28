package services

import (
	"context"
	"encoding/json"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/sleaze5/RobloxAccountManager/internal/gamelaunch"
	"github.com/sleaze5/RobloxAccountManager/internal/games"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

const (
	maxServerAge             = 60 * 24 * time.Hour
	maxServerDetailTextBytes = 64
)

type joinEndpoint struct {
	Address string `json:"Address"`
	Port    int64  `json:"Port"`
}

// serverDetailsResponse holds the fields of a join response that describe the
// server. The join script also holds tickets, tokens, and the account's own
// address and location, which stay undecoded.
type serverDetailsResponse struct {
	QueuePosition int64 `json:"queuePosition"`
	JoinScript    *struct {
		// Roblox calls the server ID GameId in the join script.
		ServerID     string `json:"GameId"`
		DatacenterID int64  `json:"DataCenterId"`
		// Roblox calls the time that the server started ServerClaimedTime,
		// in Unix milliseconds.
		StartedMs    int64 `json:"ServerClaimedTime"`
		PlaceVersion int64 `json:"PlaceVersion"`
		// Roblox calls the version of the server engine RccVersion.
		EngineVersion string `json:"RccVersion"`
		// Roblox calls the release channel ChannelName. Production servers
		// have an empty name.
		ReleaseChannel string `json:"ChannelName"`
		Locale         string `json:"GameLocale"`
		MachineAddress string `json:"MachineAddress"`
		ServerPort     int64  `json:"ServerPort"`
		// Roblox calls its UDP relays UDMUX.
		RelayEndpoints      []joinEndpoint `json:"UdmuxEndpoints"`
		PingIntervalSeconds int64          `json:"PingInterval"`
	} `json:"joinScript"`
}

// ServerDetails asks Roblox's join service about a server as the account, as
// RoSeal and BTRoblox do to show a server's region. It starts no Roblox client.
func (service *Games) ServerDetails(ctx context.Context, accountID, placeID int64, serverID string) (games.ServerDetails, error) {
	target := gamelaunch.Request{PlaceID: placeID, ServerID: serverID}
	body, status, err := service.requestJoin(ctx, accountID, 0, target)
	if err != nil {
		return games.ServerDetails{}, err
	}
	var check joinResponse
	var payload serverDetailsResponse
	if err := json.Unmarshal(body, &check); err != nil {
		return games.ServerDetails{}, invalidServerDetails(status, err)
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return games.ServerDetails{}, invalidServerDetails(status, err)
	}
	now := time.Now()
	details := games.ServerDetails{ServerID: serverID, Join: classifyJoin(check, target, now), QueuePosition: max(payload.QueuePosition, 0)}
	script := payload.JoinScript
	if script == nil || (script.ServerID != "" && !strings.EqualFold(script.ServerID, serverID)) {
		return details, nil
	}
	details.HasJoinScript = true
	details.DatacenterID = max(script.DatacenterID, 0)
	details.PlaceVersion = max(script.PlaceVersion, 0)
	details.PingIntervalSeconds = max(script.PingIntervalSeconds, 0)
	if started := time.UnixMilli(script.StartedMs); script.StartedMs > 0 && started.Before(now.Add(5*time.Minute)) && now.Sub(started) < maxServerAge {
		details.StartedMs = script.StartedMs
	}
	if isVersionText(script.EngineVersion) {
		details.EngineVersion = script.EngineVersion
	}
	details.ReleaseChannel = serverDetailText(script.ReleaseChannel)
	details.Locale = serverDetailText(script.Locale)
	details.Connection, details.Address = serverConnection(script.RelayEndpoints, script.MachineAddress, script.ServerPort)
	return details, nil
}

func invalidServerDetails(status int, err error) error {
	return &roblox.Error{Kind: roblox.KindProtocol, Endpoint: gameJoinCheckEndpoint, Status: status, Message: "Roblox returned invalid server details.", Cause: err}
}

// serverConnection reads how the client would reach the server and the
// address that it would connect to: the first relay, or the server itself.
func serverConnection(relays []joinEndpoint, machineAddress string, serverPort int64) (games.ServerConnection, string) {
	for _, relay := range relays {
		if address, ok := endpointAddress(relay.Address, relay.Port); ok {
			return games.ServerConnectionRelay, address
		}
	}
	if address, ok := endpointAddress(machineAddress, serverPort); ok {
		return games.ServerConnectionDirect, address
	}
	return games.ServerConnectionUnknown, ""
}

func endpointAddress(host string, port int64) (string, bool) {
	address, err := netip.ParseAddr(host)
	if err != nil || port <= 0 || port > 65535 {
		return "", false
	}
	return net.JoinHostPort(address.String(), strconv.FormatInt(port, 10)), true
}

func isVersionText(version string) bool {
	if version == "" || len(version) > maxServerDetailTextBytes {
		return false
	}
	for _, character := range version {
		if (character < '0' || character > '9') && character != '.' {
			return false
		}
	}
	return true
}

func serverDetailText(rawText string) string {
	if len(rawText) > maxServerDetailTextBytes {
		return ""
	}
	for _, character := range rawText {
		if character < ' ' || character > '~' {
			return ""
		}
	}
	return rawText
}
