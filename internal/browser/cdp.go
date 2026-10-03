package browser

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/sleaze5/RobloxAccountManager/internal/accounts"
	"github.com/sleaze5/RobloxAccountManager/internal/roblox"
)

type cdpMessage struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *cdpError       `json:"error,omitempty"`
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *cdpError) Error() string { return fmt.Sprintf("CDP error %d: %s", err.Code, err.Message) }

type CDPClient struct {
	input        io.WriteCloser
	output       io.ReadCloser
	writeMu      sync.Mutex
	mu           sync.Mutex
	nextID       int64
	pending      map[int64]chan cdpMessage
	events       chan cdpMessage
	done         chan struct{}
	err          error
	onFailure    func(error)
	closeOnce    sync.Once
	launchBridge *launchBridge
}

func NewCDPClient(input io.WriteCloser, output io.ReadCloser, onFailure func(error)) *CDPClient {
	client := &CDPClient{input: input, output: output, pending: make(map[int64]chan cdpMessage), events: make(chan cdpMessage, 128), done: make(chan struct{}), onFailure: onFailure}
	go client.readLoop()
	return client
}

func (client *CDPClient) Call(ctx context.Context, method string, params any, sessionID string, result any) error {
	client.mu.Lock()
	if client.err != nil {
		err := client.err
		client.mu.Unlock()
		return err
	}
	client.nextID++
	id := client.nextID
	response := make(chan cdpMessage, 1)
	client.pending[id] = response
	client.mu.Unlock()
	payload, err := json.Marshal(struct {
		ID        int64  `json:"id"`
		Method    string `json:"method"`
		Params    any    `json:"params,omitempty"`
		SessionID string `json:"sessionId,omitempty"`
	}{ID: id, Method: method, Params: params, SessionID: sessionID})
	if err != nil {
		client.removePending(id)
		return err
	}
	client.writeMu.Lock()
	_, err = client.input.Write(append(payload, 0))
	client.writeMu.Unlock()
	if err != nil {
		err = fmt.Errorf("write browser CDP input: %w", err)
		client.removePending(id)
		client.fail(err)
		return err
	}
	select {
	case message := <-response:
		if message.Error != nil {
			return message.Error
		}
		if result != nil && len(message.Result) > 0 {
			if err := json.Unmarshal(message.Result, result); err != nil {
				return fmt.Errorf("decode %s response: %w", method, err)
			}
		}
		return nil
	case <-ctx.Done():
		client.removePending(id)
		return ctx.Err()
	case <-client.done:
		client.mu.Lock()
		err := client.err
		client.mu.Unlock()
		if err == nil {
			err = io.EOF
		}
		return err
	}
}

func (client *CDPClient) Events() <-chan cdpMessage { return client.events }

func (client *CDPClient) Close() error {
	client.stop(io.EOF, false)
	return nil
}

func (client *CDPClient) readLoop() {
	reader := bufio.NewReader(client.output)
	for {
		frame, err := reader.ReadBytes(0)
		if err != nil {
			client.fail(fmt.Errorf("read browser CDP output: %w", err))
			return
		}
		frame = frame[:len(frame)-1]
		if len(frame) == 0 {
			continue
		}
		var message cdpMessage
		if err := json.Unmarshal(frame, &message); err != nil {
			client.fail(fmt.Errorf("decode CDP message: %w", err))
			return
		}
		if message.ID > 0 {
			client.mu.Lock()
			target := client.pending[message.ID]
			delete(client.pending, message.ID)
			client.mu.Unlock()
			if target != nil {
				target <- message
			}
			continue
		}
		if !client.routeLaunchEvent(message) {
			return
		}
		if strings.HasPrefix(message.Method, "Runtime.") || message.Method == "Target.attachedToTarget" {
			continue
		}
		select {
		case client.events <- message:
		default:
		}
	}
}

func (client *CDPClient) fail(err error) {
	client.stop(err, true)
}

func (client *CDPClient) stop(err error, notify bool) {
	client.closeOnce.Do(func() { _ = client.input.Close(); _ = client.output.Close() })
	client.mu.Lock()
	if client.err != nil {
		client.mu.Unlock()
		return
	}
	client.err = err
	for id := range client.pending {
		delete(client.pending, id)
	}
	close(client.done)
	client.mu.Unlock()
	if notify && client.onFailure != nil {
		client.onFailure(err)
	}
}

func (client *CDPClient) removePending(id int64) {
	client.mu.Lock()
	delete(client.pending, id)
	client.mu.Unlock()
}

type TargetInfo struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Attached bool   `json:"attached"`
}

func (client *CDPClient) VerifyVersion(ctx context.Context, expected string) error {
	var result struct {
		Product string `json:"product"`
	}
	if err := client.Call(ctx, "Browser.getVersion", nil, "", &result); err != nil {
		return err
	}
	if result.Product != "Chrome/"+expected {
		return fmt.Errorf("browser product version %q does not match %q", result.Product, expected)
	}
	return nil
}

func (client *CDPClient) DiscoverTargets(ctx context.Context) error {
	return client.Call(ctx, "Target.setDiscoverTargets", map[string]any{"discover": true}, "", nil)
}

func (client *CDPClient) EnableDownloadEvents(ctx context.Context) error {
	return client.Call(ctx, "Browser.setDownloadBehavior", map[string]any{"behavior": "default", "eventsEnabled": true}, "", nil)
}

func (client *CDPClient) Targets(ctx context.Context) ([]TargetInfo, error) {
	var result struct {
		TargetInfos []TargetInfo `json:"targetInfos"`
	}
	if err := client.Call(ctx, "Target.getTargets", nil, "", &result); err != nil {
		return nil, err
	}
	return result.TargetInfos, nil
}

func (client *CDPClient) Attach(ctx context.Context, targetID string) (string, error) {
	client.mu.Lock()
	bridge := client.launchBridge
	client.mu.Unlock()
	if bridge != nil {
		return bridge.session(ctx, targetID)
	}
	var result struct {
		SessionID string `json:"sessionId"`
	}
	err := client.Call(ctx, "Target.attachToTarget", map[string]any{"targetId": targetID, "flatten": true}, "", &result)
	return result.SessionID, err
}

func (client *CDPClient) EnablePage(ctx context.Context, sessionID string) error {
	return client.Call(ctx, "Page.enable", nil, sessionID, nil)
}

func (client *CDPClient) EnableNetwork(ctx context.Context, sessionID string) error {
	return client.Call(ctx, "Network.enable", nil, sessionID, nil)
}

func (client *CDPClient) Activate(ctx context.Context, targetID string) error {
	return client.Call(ctx, "Target.activateTarget", map[string]string{"targetId": targetID}, "", nil)
}
func (client *CDPClient) Navigate(ctx context.Context, sessionID, url string) error {
	return client.Call(ctx, "Page.navigate", map[string]string{"url": url}, sessionID, nil)
}
func (client *CDPClient) CloseBrowser(ctx context.Context) error {
	return client.Call(ctx, "Browser.close", nil, "", nil)
}

type Cookie struct {
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Domain  string  `json:"domain"`
	Path    string  `json:"path"`
	Secure  bool    `json:"secure"`
	Expires float64 `json:"expires"`
}

func (client *CDPClient) Cookies(ctx context.Context, sessionID string) ([]Cookie, error) {
	var result struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := client.Call(ctx, "Network.getCookies", map[string]any{"urls": []string{"https://www.roblox.com/"}}, sessionID, &result); err != nil {
		return nil, err
	}
	filtered := result.Cookies[:0]
	for _, cookie := range result.Cookies {
		domain := strings.TrimPrefix(strings.ToLower(cookie.Domain), ".")
		if cookie.Name == accounts.RoblosecurityCookieName && cookie.Secure && (domain == "roblox.com" || strings.HasSuffix(domain, ".roblox.com")) {
			filtered = append(filtered, cookie)
		}
	}
	return filtered, nil
}

func (client *CDPClient) SetRobloxCookie(ctx context.Context, sessionID, value string) error {
	var result struct {
		Success bool `json:"success"`
	}
	err := client.Call(ctx, "Network.setCookie", map[string]any{"name": accounts.RoblosecurityCookieName, "value": value, "url": "https://www.roblox.com/", "domain": ".roblox.com", "path": "/", "secure": true, "httpOnly": true, "sameSite": "Lax"}, sessionID, &result)
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New("chrome rejected the Roblox cookie")
	}
	return nil
}

func (client *CDPClient) SetBrowserID(ctx context.Context, sessionID, browserID string) error {
	var result struct {
		Success bool `json:"success"`
	}
	err := client.Call(ctx, "Network.setCookie", map[string]any{
		"name": roblox.EventTrackerCookieName, "value": roblox.EventTrackerCookieValue(browserID),
		"url": "https://www.roblox.com/", "domain": ".roblox.com", "path": "/", "secure": true, "sameSite": "Lax",
	}, sessionID, &result)
	if err != nil {
		return err
	}
	if !result.Success {
		return errors.New("chrome rejected the account browser ID")
	}
	return nil
}
