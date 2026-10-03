package logsexplorer

import "strings"

type clientLifecycle struct {
	lastAt     int64
	shutdownAt int64
}

func (client *clientLifecycle) record(line string, timestamp int64) {
	client.lastAt = max(client.lastAt, timestamp)
	if strings.Contains(line, "[FLog::SingleSurfaceApp]") && strings.Contains(line, "shutDown: (stage:") {
		client.shutdownAt = max(client.shutdownAt, timestamp)
	}
	if client.shutdownAt > 0 && (strings.Contains(line, "finished destroying luaApp") ||
		strings.Contains(line, "unregisterMemoryPrioritizationCallback") ||
		strings.Contains(line, "[FLog::SessionTransitionFSM] Tearing down.") ||
		strings.Contains(line, "AppPlatformQoSEmergencyHandler was destroyed.") ||
		strings.Contains(line, "Platform handler was destroyed.")) {
		client.shutdownAt = max(client.shutdownAt, timestamp)
	}
}

func (client clientLifecycle) finish(session *Session) {
	session.EndedAtMs = client.shutdownAt
	if session.EndedAtMs == 0 {
		session.EndedAtMs = client.lastAt
		session.EndEstimated = client.lastAt > 0
	}
	if session.StartedAtMs > 0 && session.EndedAtMs >= session.StartedAtMs {
		session.LifetimeMs = session.EndedAtMs - session.StartedAtMs
	}
}

func parseLaunch(line string, session *Session) {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "webclickelt") || strings.Contains(lower, "web launch") || strings.Contains(lower, "webplay noreload") {
		session.LaunchSource = SourceWebsite
	}
	switch {
	case strings.Contains(lower, "noreload") && strings.Contains(lower, "tray"):
		session.LaunchMode = LaunchTrayResume
		if session.LaunchSource != SourceWebsite {
			session.LaunchSource = SourceExistingClient
		}
	case strings.Contains(lower, "webclickelt") && strings.Contains(lower, "cold start"):
		session.LaunchMode = LaunchCold
	case strings.Contains(lower, "webclickelt") && strings.Contains(lower, "warm start"):
		if session.LaunchMode != LaunchTrayResume {
			session.LaunchMode = LaunchWarm
		}
	case strings.Contains(lower, "noreload") && strings.Contains(lower, "warm start"):
		if session.LaunchMode == "" || session.LaunchMode == LaunchTray {
			session.LaunchMode = LaunchWarm
		}
		if session.LaunchSource != SourceWebsite {
			session.LaunchSource = SourceExistingClient
		}
	case strings.Contains(lower, "appstate/traymode"):
		if session.LaunchMode == "" {
			session.LaunchMode = LaunchTray
		}
		if session.LaunchSource == "" {
			session.LaunchSource = SourceTray
		}
	}
}
