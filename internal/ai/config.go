package ai

import (
	"fmt"
	"orion/internal/globalconfig"
)

// Config stores LLM configuration loaded from ~/.orion.yaml.
type Config struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// LoadConfig loads llm configuration from ~/.orion.yaml.
func LoadConfig() (*Config, error) {
	globalCfg, err := globalconfig.Load()
	if err != nil {
		path, pathErr := globalconfig.Path()
		if pathErr != nil {
			return nil, pathErr
		}
		return nil, fmt.Errorf("failed to load global config %s: %w", path, err)
	}

	cfg := &Config{
		APIKey:  globalCfg.LLM.APIKey,
		BaseURL: globalCfg.LLM.BaseURL,
		Model:   globalCfg.LLM.Model,
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.moonshot.cn/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "kimi-k2-turbo-preview"
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("llm.api_key is required in ~/.orion.yaml")
	}

	return cfg, nil
}

// ExampleConfig returns example configuration content
func ExampleConfig() string {
	return `# Orion Global Configuration
# Place this in ~/.orion.yaml

llm:
  api_key: "${MOONSHOT_API_KEY}"
  base_url: "https://api.moonshot.cn/v1"
  model: "kimi-k2-turbo-preview"

notifications:
  provider: "lark"
  lark:
    app_id: "${ORION_LARK_APP_ID}"
    app_secret: "${ORION_LARK_APP_SECRET}"
    open_id: "${ORION_LARK_OPEN_ID}"
    base_url: "https://open.feishu.cn"
    urgent_app: true
    card_title: "boss, 我想干活"
`
}
