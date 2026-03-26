package notification

import (
	"context"
	"fmt"
	"strings"
	"sync"

	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	larkdispatcher "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcallback "github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

var larkCallbackRunner struct {
	mu      sync.Mutex
	started bool
}

func startLarkCallbackWS(rootPath string, cfg ServiceConfig) error {
	larkCfg := cfg.Lark
	larkCfg.AppID = resolveEnvReference(larkCfg.AppID)
	larkCfg.AppSecret = resolveEnvReference(larkCfg.AppSecret)
	if strings.TrimSpace(larkCfg.AppID) == "" || strings.TrimSpace(larkCfg.AppSecret) == "" {
		return fmt.Errorf("notifications.lark.app_id and notifications.lark.app_secret are required for lark websocket callbacks")
	}

	larkCallbackRunner.mu.Lock()
	if larkCallbackRunner.started {
		larkCallbackRunner.mu.Unlock()
		return nil
	}
	larkCallbackRunner.started = true
	larkCallbackRunner.mu.Unlock()

	dispatcher := larkdispatcher.NewEventDispatcher("", "")
	dispatcher.OnP2CardActionTrigger(func(ctx context.Context, event *larkcallback.CardActionTriggerEvent) (*larkcallback.CardActionTriggerResponse, error) {
		if event == nil || event.Event == nil || event.Event.Action == nil {
			return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "warning", Content: "empty action"}}, nil
		}

		meta := parseCardActionMetadata(event.Event.Action.Value)
		action := strings.ToLower(strings.TrimSpace(meta.Action))
		if action == "" {
			action = strings.ToLower(strings.TrimSpace(event.Event.Action.Name))
		}

		switch action {
		case "ack":
			if err := MuteWaitReminder(rootPath, meta); err != nil {
				serviceLogf("notification.lark_callback.ack_failed node=%s error=%q", meta.NodeName, err.Error())
				return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "danger", Content: "Ack failed"}}, nil
			}
			serviceLogf("notification.lark_callback.ack_success node=%s wait_event=%d", meta.NodeName, meta.WaitEvent)
			return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "success", Content: "Reminder muted"}}, nil
		case "reply":
			reply := extractReplyText(event.Event.Action)
			if err := RouteReplyToNode(rootPath, meta, reply); err != nil {
				serviceLogf("notification.lark_callback.reply_failed node=%s error=%q", meta.NodeName, err.Error())
				return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "danger", Content: "Reply failed"}}, nil
			}
			serviceLogf("notification.lark_callback.reply_success node=%s wait_event=%d", meta.NodeName, meta.WaitEvent)
			return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "success", Content: "Reply sent"}}, nil
		default:
			return &larkcallback.CardActionTriggerResponse{Toast: &larkcallback.Toast{Type: "warning", Content: "Unknown action"}}, nil
		}
	})

	// Ignore noisy events from the same app to avoid error logs on ws handler.
	dispatcher.OnCustomizedEvent("im.message.message_read_v1", func(ctx context.Context, event *larkevent.EventReq) error { return nil })
	dispatcher.OnCustomizedEvent("im.chat.access_event.bot_p2p_chat_entered_v1", func(ctx context.Context, event *larkevent.EventReq) error { return nil })

	wsClient := larkws.NewClient(larkCfg.AppID, larkCfg.AppSecret, larkws.WithEventHandler(dispatcher))
	go func() {
		if err := wsClient.Start(context.Background()); err != nil {
			serviceLogf("notification.lark_callback.ws_stopped error=%q", err.Error())
		}
	}()
	serviceLogf("notification.lark_callback.ws_started")
	return nil
}

func parseCardActionMetadata(value map[string]interface{}) CardActionMetadata {
	meta := CardActionMetadata{}
	if value == nil {
		return meta
	}
	meta.Action, _ = asString(value["action"])
	meta.NodeName, _ = asString(value["node_name"])
	meta.Label, _ = asString(value["label"])
	meta.PaneID, _ = asString(value["pane_id"])
	meta.ScreenHash, _ = asString(value["screen_hash"])
	meta.SentAt, _ = asString(value["sent_at"])
	meta.WaitEvent = asInt(value["wait_event_id"])
	return meta
}

func extractReplyText(action *larkcallback.CallBackAction) string {
	if action == nil {
		return ""
	}
	if action.FormValue != nil {
		if v, ok := action.FormValue["reply_text"]; ok {
			if s, ok := asString(v); ok {
				return s
			}
		}
	}
	if action.InputValue != "" {
		return action.InputValue
	}
	return ""
}

func asString(v interface{}) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	case string:
		n = strings.TrimSpace(n)
		if n == "" {
			return 0
		}
		var out int
		_, _ = fmt.Sscanf(n, "%d", &out)
		return out
	default:
		return 0
	}
}
