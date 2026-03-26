package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

var lookPath = exec.LookPath
var sendWatcherNotification = dispatchWatcherNotification

type watcherNotifier interface {
	NotifyWatcher(watcher *Watcher, reason string) error
}

type notifierState struct {
	mu       sync.RWMutex
	notifier watcherNotifier
}

var globalNotifier = &notifierState{
	notifier: &macNotifier{},
}

func dispatchWatcherNotification(watcher *Watcher, reason string) error {
	globalNotifier.mu.RLock()
	current := globalNotifier.notifier
	globalNotifier.mu.RUnlock()
	return current.NotifyWatcher(watcher, reason)
}

func configureNotifier(cfg ServiceConfig) error {
	notifier, err := newNotifier(cfg)
	if err != nil {
		return err
	}
	globalNotifier.mu.Lock()
	globalNotifier.notifier = notifier
	globalNotifier.mu.Unlock()
	return nil
}

func newNotifier(cfg ServiceConfig) (watcherNotifier, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "mac", "macos", "darwin":
		return &macNotifier{}, nil
	case "lark":
		return newLarkNotifier(cfg.Lark)
	default:
		return nil, fmt.Errorf("unsupported notifications.provider: %s", cfg.Provider)
	}
}

type macNotifier struct{}

func NotifyWatcher(nodeName, label, reason string) error {
	return (&macNotifier{}).NotifyWatcher(&Watcher{NodeName: nodeName, Label: label}, reason)
}

func (m *macNotifier) NotifyWatcher(watcher *Watcher, reason string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("mac notifications are only supported on darwin")
	}
	nodeName := watcher.NodeName
	label := watcher.Label

	body := "Waiting for input"
	if reason != "" {
		body = fmt.Sprintf("Waiting for input: %s", reason)
	}

	cmd, err := buildNotificationCommand(nodeName, label, body)
	if err != nil {
		return err
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to send mac notification: %s: %w", string(output), err)
	}
	return nil
}

func buildNotificationCommand(nodeName, label, body string) (*exec.Cmd, error) {
	subtitle := fmt.Sprintf("Node %s", nodeName)
	if label != "" {
		subtitle = fmt.Sprintf("Node %s (%s)", nodeName, label)
	}

	if _, err := lookPath("terminal-notifier"); err == nil {
		cmd := exec.Command("terminal-notifier",
			"-title", "Orion",
			"-subtitle", subtitle,
			"-message", body,
			"-sound", "default",
		)
		return cmd, nil
	}

	script := fmt.Sprintf("display notification %q with title %q subtitle %q", body, "Orion", subtitle)
	if _, err := lookPath("osascript"); err == nil {
		return exec.Command("osascript", "-e", script), nil
	}

	return nil, fmt.Errorf("no supported mac notification command found")
}

type larkNotifier struct {
	cfg    LarkConfig
	client *lark.Client
}

func newLarkNotifier(cfg LarkConfig) (*larkNotifier, error) {
	cfg.AppID = resolveEnvReference(cfg.AppID)
	cfg.AppSecret = resolveEnvReference(cfg.AppSecret)
	cfg.BaseURL = resolveEnvReference(cfg.BaseURL)
	cfg.OpenID = resolveEnvReference(cfg.OpenID)
	cfg.ChatID = resolveEnvReference(cfg.ChatID)

	if strings.TrimSpace(cfg.AppID) == "" {
		return nil, fmt.Errorf("notifications.lark.app_id is required for lark provider")
	}
	if strings.TrimSpace(cfg.AppSecret) == "" {
		return nil, fmt.Errorf("notifications.lark.app_secret is required for lark provider")
	}
	if strings.TrimSpace(cfg.OpenID) == "" && strings.TrimSpace(cfg.ChatID) == "" {
		return nil, fmt.Errorf("notifications.lark.open_id or notifications.lark.chat_id is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://open.feishu.cn"
	}
	if strings.TrimSpace(cfg.CardTitle) == "" {
		cfg.CardTitle = "boss, 我想干活"
	}

	clientOptions := []lark.ClientOptionFunc{
		lark.WithReqTimeout(10 * time.Second),
	}
	if cfg.BaseURL != "" && cfg.BaseURL != "https://open.feishu.cn" {
		clientOptions = append(clientOptions, lark.WithOpenBaseUrl(cfg.BaseURL))
	}

	return &larkNotifier{
		cfg:    cfg,
		client: lark.NewClient(cfg.AppID, cfg.AppSecret, clientOptions...),
	}, nil
}

func (n *larkNotifier) NotifyWatcher(watcher *Watcher, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	messageID, err := n.sendInteractiveCard(ctx, watcher, reason)
	if err != nil {
		return err
	}
	if !n.cfg.UrgentApp {
		return nil
	}
	if strings.TrimSpace(n.cfg.OpenID) == "" {
		return nil
	}
	return n.markUrgent(ctx, messageID)
}

func (n *larkNotifier) sendInteractiveCard(ctx context.Context, watcher *Watcher, reason string) (string, error) {
	contentPayload := buildLarkCardPayload(n.cfg.CardTitle, watcher, reason)
	contentBytes, err := json.Marshal(contentPayload)
	if err != nil {
		return "", fmt.Errorf("failed to encode lark card content: %w", err)
	}

	receiveIDType := "chat_id"
	receiveID := strings.TrimSpace(n.cfg.ChatID)
	if receiveID == "" {
		receiveIDType = "open_id"
		receiveID = strings.TrimSpace(n.cfg.OpenID)
	}

	resp, err := n.client.Im.V1.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType("interactive").
			Content(string(contentBytes)).
			Build()).
		Build())
	if err != nil {
		return "", fmt.Errorf("lark send message failed: %w", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("lark send message failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Data == nil || resp.Data.MessageId == nil || strings.TrimSpace(*resp.Data.MessageId) == "" {
		return "", fmt.Errorf("lark send message returned empty message_id")
	}
	return strings.TrimSpace(*resp.Data.MessageId), nil
}

func buildLarkCardPayload(title string, watcher *Watcher, reason string) map[string]interface{} {
	nodeName := watcher.NodeName
	label := watcher.Label
	if strings.TrimSpace(reason) == "" {
		reason = watcher.LastReason
	}
	latestBlock := strings.TrimSpace(watcher.LastAgentBlock)
	if latestBlock == "" {
		latestBlock = "-"
	}

	ackValue := map[string]interface{}{
		"action":        "ack",
		"node_name":     watcher.NodeName,
		"label":         watcher.Label,
		"pane_id":       watcher.PaneID,
		"wait_event_id": watcher.WaitEventID,
		"screen_hash":   watcher.LastHash,
		"sent_at":       time.Now().Format(time.RFC3339),
	}
	replyValue := map[string]interface{}{
		"action":        "reply",
		"node_name":     watcher.NodeName,
		"label":         watcher.Label,
		"pane_id":       watcher.PaneID,
		"wait_event_id": watcher.WaitEventID,
		"screen_hash":   watcher.LastHash,
		"sent_at":       time.Now().Format(time.RFC3339),
	}

	return map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "red",
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": title,
			},
		},
		"elements": []map[string]interface{}{
			{
				"tag": "markdown",
				"content": fmt.Sprintf(
					"**Node:** %s\n**Label:** %s\n**State:** waiting_input\n**Reason:** %s",
					nodeName,
					emptyAsDash(label),
					emptyAsDash(reason),
				),
			},
			{
				"tag":     "markdown",
				"content": fmt.Sprintf("**Latest Agent Response**\n%s", latestBlock),
			},
			{
				"tag":  "form",
				"name": "reply_form",
				"elements": []map[string]interface{}{
					{
						"tag":        "input",
						"name":       "reply_text",
						"max_length": 1000,
						"multiline":  true,
						"placeholder": map[string]interface{}{
							"tag":     "plain_text",
							"content": "Quick reply to agent",
						},
					},
					{
						"tag":  "button",
						"name": "submit_reply",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "Reply 提交",
						},
						"type":        "primary",
						"action_type": "form_submit",
						"value":       replyValue,
					},
				},
			},
			{
				"tag": "action",
				"actions": []map[string]interface{}{
					{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "Ack",
						},
						"type":  "default",
						"value": ackValue,
					},
				},
			},
		},
	}
}

func (n *larkNotifier) markUrgent(ctx context.Context, messageID string) error {
	resp, err := n.client.Im.V1.Message.UrgentApp(ctx, larkim.NewUrgentAppMessageReqBuilder().
		MessageId(messageID).
		UserIdType("open_id").
		UrgentReceivers(larkim.NewUrgentReceiversBuilder().
			UserIdList([]string{n.cfg.OpenID}).
			Build()).
		Build())
	if err != nil {
		return fmt.Errorf("lark urgent_app failed: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("lark urgent_app failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

func emptyAsDash(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return "-"
	}
	return s
}

func resolveEnvReference(v string) string {
	s := strings.TrimSpace(v)
	if len(s) > 3 && strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		name := strings.TrimSpace(s[2 : len(s)-1])
		if name != "" {
			if env := strings.TrimSpace(os.Getenv(name)); env != "" {
				return env
			}
		}
	}
	return s
}
