# Notifications

Orion 通知服务会扫描 node session，并识别 agent 是否处于“等待人类输入”状态。

[English](README.md) | [简体中文](README_zh-CN.md)

## 工作方式

- Orion 持续追踪已注册 watcher 的 tmux pane 输出。
- 当输出被判定为等待输入时，Orion 触发通知。
- 你进入该 node 后，会自动 ack 该等待事件。
- Lark 交互卡片支持：
  - `Ack`：静音当前等待事件的后续提醒通知。
  - `Reply`：将快速回复路由回目标 node（通过 tmux 注入）。
- `Ack` 不会清除 `orion enter` 里展示的 pending wait-for-input 状态。
- 通知投递官方支持 Feishu/Lark Bot 通道。

## 命令

```bash
orion notification-service start
orion notification-service status
orion notification-service list-watchers
orion notification-service stop
```

## 配置

通知配置位于全局 `~/.orion.yaml`（所有 workspace 共用一套通知策略）：

```yaml
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

## 卡片内容

Lark 卡片包含：

- 来自 `notifications.lark.card_title` 的可配置标题
- Node 名称与 Label
- 当前 wait-input 原因
- 从终端输出提取的最近一次 agent response block（`notifications.last_block`）

## Last Block 提取规则

通过 `notifications.last_block` 适配不同代码 agent：

默认情况下，Orion 已按 OpenAI Codex 风格输出做了内置适配（`prefix: "• "`），使用 Codex 时无需额外配置。

```yaml
# Codex 风格（按 bullet 前缀提取）
notifications:
  last_block:
    enabled: true
    mode: prefix
    prefix: "• "
    max_chars: 1200
```

```yaml
# Kimi 风格（示例：按正则提取）
notifications:
  last_block:
    enabled: true
    mode: regex
    regex: "(?s)Final Answer:\\s*(.+)$"
    max_chars: 1200
```

## 官方通知通道

- 本地桌面通知
- Feishu/Lark Bot 通知

## 可扩展 Provider 架构

这套通知架构是 provider-oriented 的，可复用于更多协作平台：

- `Watcher` 与等待输入分类逻辑保留在 Orion Core。
- 投递层可扩展为不同通知 provider。
- 你可以在不改变检测语义的前提下切换通知通道。

欢迎社区贡献官方 provider：

- Slack
- Discord
