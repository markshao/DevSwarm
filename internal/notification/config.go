package notification

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"orion/internal/globalconfig"
	"orion/internal/types"

	"gopkg.in/yaml.v3"
)

func defaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		Enabled:             true,
		Provider:            "macos",
		PollInterval:        5 * time.Second,
		SilenceThreshold:    20 * time.Second,
		ReminderInterval:    5 * time.Minute,
		SimilarityThreshold: 0.99,
		TailLines:           80,
		LLMEnabled:          true,
		Lark: LarkConfig{
			BaseURL:   "https://open.feishu.cn",
			UrgentApp: true,
			CardTitle: "boss, 我想干活",
		},
	}
}

func LoadServiceConfig(rootPath string) (ServiceConfig, error) {
	cfg := defaultServiceConfig()

	configPath := filepath.Join(rootPath, ".orion", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return ServiceConfig{}, fmt.Errorf("failed to read notification config: %w", err)
	}

	var workspaceCfg types.Config
	if err := yaml.Unmarshal(data, &workspaceCfg); err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to parse notification config: %w", err)
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to inspect notification config: %w", err)
	}

	notificationsRaw, hasNotifications := raw["notifications"].(map[string]interface{})
	if hasNotifications {
		if enabled, ok := notificationsRaw["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
		if provider, ok := notificationsRaw["provider"].(string); ok && provider != "" {
			cfg.Provider = provider
		}
		if llmRaw, ok := notificationsRaw["llm_classifier"].(map[string]interface{}); ok {
			if enabled, ok := llmRaw["enabled"].(bool); ok {
				cfg.LLMEnabled = enabled
			}
		}
	}

	if workspaceCfg.Notifications.Provider != "" {
		cfg.Provider = workspaceCfg.Notifications.Provider
	}

	if workspaceCfg.Notifications.PollInterval != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.PollInterval)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.poll_interval: %w", err)
		}
		cfg.PollInterval = d
	}
	if workspaceCfg.Notifications.SilenceThreshold != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.SilenceThreshold)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.silence_threshold: %w", err)
		}
		cfg.SilenceThreshold = d
	}
	if workspaceCfg.Notifications.ReminderInterval != "" {
		d, err := time.ParseDuration(workspaceCfg.Notifications.ReminderInterval)
		if err != nil {
			return ServiceConfig{}, fmt.Errorf("invalid notifications.reminder_interval: %w", err)
		}
		cfg.ReminderInterval = d
	}
	if workspaceCfg.Notifications.SimilarityThreshold > 0 {
		cfg.SimilarityThreshold = workspaceCfg.Notifications.SimilarityThreshold
	}
	if workspaceCfg.Notifications.TailLines > 0 {
		cfg.TailLines = workspaceCfg.Notifications.TailLines
	}
	globalCfg, err := globalconfig.LoadOptional()
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("failed to load global config: %w", err)
	}
	if globalCfg != nil {
		if globalCfg.Notifications.Provider != "" {
			cfg.Provider = globalCfg.Notifications.Provider
		}
		globalLark := globalCfg.Notifications.Lark
		if globalLark.AppID != "" {
			cfg.Lark.AppID = globalLark.AppID
		}
		if globalLark.AppSecret != "" {
			cfg.Lark.AppSecret = globalLark.AppSecret
		}
		if globalLark.BaseURL != "" {
			cfg.Lark.BaseURL = globalLark.BaseURL
		}
		if globalLark.OpenID != "" {
			cfg.Lark.OpenID = globalLark.OpenID
		}
		if globalLark.ChatID != "" {
			cfg.Lark.ChatID = globalLark.ChatID
		}
		if globalLark.UrgentApp != nil {
			cfg.Lark.UrgentApp = *globalLark.UrgentApp
		}
		if globalLark.CardTitle != "" {
			cfg.Lark.CardTitle = globalLark.CardTitle
		}
	}
	return cfg, nil
}
