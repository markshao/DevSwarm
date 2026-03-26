package notification

import (
	"os"
	"strings"
	"testing"
)

func TestLarkBotSendWaitInputNotificationLive(t *testing.T) {
	cfg := LarkConfig{
		AppID:     strings.TrimSpace(os.Getenv("ORION_LARK_APP_ID")),
		AppSecret: strings.TrimSpace(os.Getenv("ORION_LARK_APP_SECRET")),
		BaseURL:   strings.TrimSpace(os.Getenv("ORION_LARK_BASE_URL")),
		OpenID:    strings.TrimSpace(os.Getenv("ORION_LARK_OPEN_ID")),
		ChatID:    strings.TrimSpace(os.Getenv("ORION_LARK_CHAT_ID")),
		UrgentApp: false,
		CardTitle: "boss, 我想干活",
	}

	if cfg.AppID == "" || cfg.AppSecret == "" || (cfg.OpenID == "" && cfg.ChatID == "") {
		t.Skip("set ORION_LARK_APP_ID, ORION_LARK_APP_SECRET, and ORION_LARK_OPEN_ID or ORION_LARK_CHAT_ID to run live test")
	}

	notifier, err := newLarkNotifier(cfg)
	if err != nil {
		t.Fatalf("newLarkNotifier returned error: %v", err)
	}

	if err := notifier.NotifyWatcher(&Watcher{NodeName: "demo-node", Label: "live-test"}, "approval required"); err != nil {
		t.Fatalf("failed to send lark notification: %v", err)
	}
}
