# Configuration

Orion has two main configuration surfaces.

## 1) AI config for `orion ai`

File: `~/.orion.conf`

```yaml
api_key: "$MOONSHOT_API_KEY"
base_url: "https://api.moonshot.cn/v1"
model: "kimi-k2-turbo-preview"
```

`api_key` can be a raw key or an environment variable reference.

## 2) Workspace config

File: `.orion/config.yaml`

```yaml
version: 1

git:
  main_branch: main

runtime:
  artifact_dir: .orion/runs

agents:
  default_provider: traecli
  providers:
    traecli:
      command: 'traecli "{{.Prompt}}" -py'
    qwen:
      command: 'qwen "{{.Prompt}}" -y'
    kimi:
      command: 'kimi -y -p "{{.Prompt}}"'

notifications:
  enabled: true
  provider: lark
  poll_interval: 5s
  silence_threshold: 20s
  reminder_interval: 5m
  similarity_threshold: 0.99
  tail_lines: 80
  last_block:
    enabled: true
    mode: prefix
    prefix: "• "
    max_chars: 1200
  llm_classifier:
    enabled: true
  lark:
    app_id: ${ORION_LARK_APP_ID}
    app_secret: ${ORION_LARK_APP_SECRET}
    open_id: ${ORION_LARK_OPEN_ID}
    base_url: https://open.feishu.cn
    urgent_app: true
    card_title: "boss, I want to work"
```

`notifications.last_block` controls how Orion extracts the latest agent response block for interactive Lark cards (supports `prefix` and `regex` modes).

## Related workflow files

- `.orion/workflows/*.yaml`
- `.orion/agents/*.yaml`
- `.orion/prompts/*.md`

These define pipeline steps, runtime binding, and prompt behavior.
