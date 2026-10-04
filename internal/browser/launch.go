package browser

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const launchBinding = "__ramLaunchRoblox"

//go:embed launch_bridge.js
var launchBridgeScript string

type launchBridge struct {
	client   *CDPClient
	events   chan cdpMessage
	launch   func(string)
	contexts map[string]map[int64]bool
	mu       sync.Mutex
	sessions map[string]string
	changed  chan struct{}
}

func (client *CDPClient) InterceptRobloxLaunches(ctx context.Context, launch func(string)) error {
	bridge := &launchBridge{
		client: client, events: make(chan cdpMessage, 128), launch: launch,
		contexts: make(map[string]map[int64]bool), sessions: make(map[string]string), changed: make(chan struct{}),
	}
	client.mu.Lock()
	client.launchBridge = bridge
	client.mu.Unlock()
	go bridge.run()
	return client.Call(ctx, "Target.setAutoAttach", map[string]any{
		"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true,
		"filter": []map[string]any{{"type": "page"}, {"exclude": true}},
	}, "", nil)
}

func (bridge *launchBridge) session(ctx context.Context, targetID string) (string, error) {
	for {
		bridge.mu.Lock()
		sessionID, changed := bridge.sessions[targetID], bridge.changed
		bridge.mu.Unlock()
		if sessionID != "" {
			return sessionID, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-bridge.client.done:
			return "", errors.New("browser launch bridge closed")
		case <-changed:
		}
	}
}

func (client *CDPClient) routeLaunchEvent(message cdpMessage) bool {
	switch message.Method {
	case "Target.attachedToTarget", "Target.detachedFromTarget", "Runtime.executionContextCreated", "Runtime.executionContextDestroyed", "Runtime.executionContextsCleared", "Runtime.bindingCalled":
	default:
		return true
	}
	client.mu.Lock()
	bridge := client.launchBridge
	client.mu.Unlock()
	if bridge == nil {
		return true
	}
	select {
	case <-client.done:
		return false
	case bridge.events <- message:
		return true
	default:
		client.fail(errors.New("browser launch event queue overflow"))
		return false
	}
}

func (bridge *launchBridge) run() {
	defer func() {
		for {
			select {
			case <-bridge.events:
			default:
				return
			}
		}
	}()
	for {
		select {
		case <-bridge.client.done:
			return
		case event := <-bridge.events:
			bridge.handle(event)
		}
	}
}

func (bridge *launchBridge) handle(event cdpMessage) {
	switch event.Method {
	case "Target.attachedToTarget":
		var params struct {
			SessionID string     `json:"sessionId"`
			Target    TargetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.Target.Type == "page" {
			bridge.attach(params.Target.TargetID, params.SessionID)
		}
	case "Target.detachedFromTarget":
		var params struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(event.Params, &params) == nil {
			bridge.detach(params.SessionID)
		}
	default:
		bridge.handleRuntimeEvent(event)
	}
}

func (bridge *launchBridge) detach(sessionID string) {
	delete(bridge.contexts, sessionID)
	bridge.mu.Lock()
	defer bridge.mu.Unlock()
	for targetID, attached := range bridge.sessions {
		if attached == sessionID {
			delete(bridge.sessions, targetID)
		}
	}
}

func (bridge *launchBridge) handleRuntimeEvent(event cdpMessage) {
	switch event.Method {
	case "Runtime.executionContextCreated":
		var params struct {
			Context struct {
				ID      int64  `json:"id"`
				Origin  string `json:"origin"`
				AuxData struct {
					IsDefault bool `json:"isDefault"`
				} `json:"auxData"`
			} `json:"context"`
		}
		if json.Unmarshal(event.Params, &params) == nil && bridge.contexts[event.SessionID] != nil {
			bridge.contexts[event.SessionID][params.Context.ID] = params.Context.AuxData.IsDefault && (params.Context.Origin == "https://www.roblox.com" || params.Context.Origin == "https://roblox.com")
		}
	case "Runtime.executionContextsCleared":
		clear(bridge.contexts[event.SessionID])
	case "Runtime.executionContextDestroyed":
		var params struct {
			ID int64 `json:"executionContextId"`
		}
		if json.Unmarshal(event.Params, &params) == nil {
			delete(bridge.contexts[event.SessionID], params.ID)
		}
	case "Runtime.bindingCalled":
		var params struct {
			Name    string `json:"name"`
			Payload string `json:"payload"`
			Context int64  `json:"executionContextId"`
		}
		if json.Unmarshal(event.Params, &params) == nil && params.Name == launchBinding && bridge.contexts[event.SessionID][params.Context] {
			bridge.launch(params.Payload)
		}
	}
}

func (bridge *launchBridge) attach(targetID, sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bridge.contexts[sessionID] = make(map[int64]bool)
	if err := bridge.install(ctx, sessionID); err != nil {
		bridge.detach(sessionID)
		bridge.handleAttachFailure(targetID, err)
		return
	}
	bridge.mu.Lock()
	bridge.sessions[targetID] = sessionID
	close(bridge.changed)
	bridge.changed = make(chan struct{})
	bridge.mu.Unlock()
}

func (bridge *launchBridge) handleAttachFailure(targetID string, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	targets, err := bridge.client.Targets(ctx)
	if err == nil {
		found := false
		for _, target := range targets {
			found = found || target.TargetID == targetID
		}
		if !found {
			return
		}
	}
	bridge.client.fail(fmt.Errorf("prepare browser game launch handoff: %w", cause))
}

func (bridge *launchBridge) install(ctx context.Context, sessionID string) error {
	if err := bridge.client.EnablePage(ctx, sessionID); err != nil {
		return err
	}
	if err := bridge.client.Call(ctx, "Runtime.enable", nil, sessionID, nil); err != nil {
		return err
	}
	if err := bridge.client.Call(ctx, "Runtime.addBinding", map[string]string{"name": launchBinding}, sessionID, nil); err != nil {
		return err
	}
	if err := bridge.client.Call(ctx, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": launchBridgeScript, "runImmediately": true}, sessionID, nil); err != nil {
		return err
	}
	return bridge.client.Call(ctx, "Runtime.runIfWaitingForDebugger", nil, sessionID, nil)
}
