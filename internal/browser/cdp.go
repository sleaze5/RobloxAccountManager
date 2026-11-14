package browser

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
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
	writeMutex   sync.Mutex
	mutex        sync.Mutex
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

func (client *CDPClient) Call(ctx context.Context, method string, parameters any, sessionID string, target any) error {
	client.mutex.Lock()
	if client.err != nil {
		err := client.err
		client.mutex.Unlock()
		return err
	}
	client.nextID++
	callID := client.nextID
	response := make(chan cdpMessage, 1)
	client.pending[callID] = response
	client.mutex.Unlock()
	payload, err := json.Marshal(struct {
		ID        int64  `json:"id"`
		Method    string `json:"method"`
		Params    any    `json:"params,omitempty"`
		SessionID string `json:"sessionId,omitempty"`
	}{ID: callID, Method: method, Params: parameters, SessionID: sessionID})
	if err != nil {
		client.removePending(callID)
		return fmt.Errorf("encode CDP command: %w", err)
	}
	client.writeMutex.Lock()
	_, err = client.input.Write(append(payload, 0))
	client.writeMutex.Unlock()
	if err != nil {
		err = fmt.Errorf("write browser CDP input: %w", err)
		client.removePending(callID)
		client.fail(err)
		return err
	}
	select {
	case message := <-response:
		if message.Error != nil {
			return message.Error
		}
		if target != nil && len(message.Result) > 0 {
			if err := json.Unmarshal(message.Result, target); err != nil {
				return fmt.Errorf("decode %s response: %w", method, err)
			}
		}
		return nil
	case <-ctx.Done():
		client.removePending(callID)
		return ctx.Err()
	case <-client.done:
		client.mutex.Lock()
		err := client.err
		client.mutex.Unlock()
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
			client.mutex.Lock()
			target := client.pending[message.ID]
			delete(client.pending, message.ID)
			client.mutex.Unlock()
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
	client.mutex.Lock()
	if client.err != nil {
		client.mutex.Unlock()
		return
	}
	client.err = err
	for callID := range client.pending {
		delete(client.pending, callID)
	}
	close(client.done)
	client.mutex.Unlock()
	if notify && client.onFailure != nil {
		client.onFailure(err)
	}
}

func (client *CDPClient) removePending(callID int64) {
	client.mutex.Lock()
	delete(client.pending, callID)
	client.mutex.Unlock()
}

type CDPTarget struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
}

func (client *CDPClient) VerifyVersion(ctx context.Context, expected string) error {
	var result struct {
		Product string `json:"product"`
	}
	if err := client.Call(ctx, "Browser.getVersion", nil, "", &result); err != nil {
		return err
	}
	if result.Product != "Chrome/"+expected {
		return fmt.Errorf("%w: got %q, want %q", errBrowserVersionMismatch, result.Product, expected)
	}
	return nil
}

func (client *CDPClient) DiscoverTargets(ctx context.Context) error {
	return client.Call(ctx, "Target.setDiscoverTargets", map[string]any{"discover": true}, "", nil)
}

func (client *CDPClient) EnableDownloadEvents(ctx context.Context) error {
	return client.Call(ctx, "Browser.setDownloadBehavior", map[string]any{"behavior": "default", "eventsEnabled": true}, "", nil)
}

func (client *CDPClient) Targets(ctx context.Context) ([]CDPTarget, error) {
	var result struct {
		TargetInfos []CDPTarget `json:"targetInfos"`
	}
	if err := client.Call(ctx, "Target.getTargets", nil, "", &result); err != nil {
		return nil, err
	}
	return result.TargetInfos, nil
}

func (client *CDPClient) Attach(ctx context.Context, targetID string) (string, error) {
	client.mutex.Lock()
	bridge := client.launchBridge
	client.mutex.Unlock()
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
func (client *CDPClient) CreateTarget(ctx context.Context, url string) (string, error) {
	var result struct {
		TargetID string `json:"targetId"`
	}
	err := client.Call(ctx, "Target.createTarget", map[string]string{"url": url}, "", &result)
	return result.TargetID, err
}
func (client *CDPClient) Navigate(ctx context.Context, sessionID, url string) error {
	return client.Call(ctx, "Page.navigate", map[string]string{"url": url}, sessionID, nil)
}
func (client *CDPClient) CloseBrowser(ctx context.Context) error {
	return client.Call(ctx, "Browser.close", nil, "", nil)
}

type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Secure bool   `json:"secure"`
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

func (client *CDPClient) SetRobloxCookie(ctx context.Context, sessionID, cookieValue string) error {
	return client.setCookie(ctx, sessionID, map[string]any{"name": accounts.RoblosecurityCookieName, "value": cookieValue, "httpOnly": true}, "chrome rejected the Roblox cookie")
}

func (client *CDPClient) SetBrowserID(ctx context.Context, sessionID, browserID string) error {
	return client.setCookie(ctx, sessionID, map[string]any{"name": roblox.EventTrackerCookieName, "value": roblox.EventTrackerCookieValue(browserID)}, "chrome rejected the account browser ID")
}

func (client *CDPClient) setCookie(ctx context.Context, sessionID string, fields map[string]any, rejected string) error {
	cookie := map[string]any{"url": "https://www.roblox.com/", "domain": ".roblox.com", "path": "/", "secure": true, "sameSite": "Lax"}
	maps.Copy(cookie, fields)
	var result struct {
		Success bool `json:"success"`
	}
	if err := client.Call(ctx, "Network.setCookie", cookie, sessionID, &result); err != nil {
		return err
	}
	if !result.Success {
		return errors.New(rejected)
	}
	return nil
}
