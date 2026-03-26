# Configuration

Orion 主要有两层配置。

[English](README.md) | [简体中文](README_zh-CN.md)

## 1) `orion ai` 的 AI 配置

文件：`~/.orion.conf`

```yaml
api_key: "$MOONSHOT_API_KEY"
base_url: "https://api.moonshot.cn/v1"
model: "kimi-k2-turbo-preview"
```

`api_key` 支持直接填写，或使用环境变量引用。

## 2) 工作区配置

文件：`.orion/config.yaml`

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
    card_title: "boss, 我想干活"
```

`notifications.last_block` 用于控制 Orion 如何提取最近一次 agent response block 供 Lark 交互卡片展示（支持 `prefix` 和 `regex` 模式）。

## Workflow 相关文件

- `.orion/workflows/*.yaml`
- `.orion/agents/*.yaml`
- `.orion/prompts/*.md`

它们分别定义 pipeline step、agent 运行时和 prompt 行为。
