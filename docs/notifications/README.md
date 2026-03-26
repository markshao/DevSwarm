# Notifications

Orion notification service scans node sessions and detects when an agent is waiting for human input.

## How it works

- Orion tracks tmux pane output for registered node watchers.
- When output is classified as waiting for input, Orion emits a local notification.
- Entering the node acknowledges the pending wait event.
- Lark interactive card supports:
  - `Ack`: mute reminder notifications for the current wait event.
  - `Reply`: send quick reply text back to the target node via tmux.
- `Ack` does not clear pending wait-input status shown in `orion enter`.
- Notification delivery supports Feishu/Lark bot integration as an official channel.

## Commands

```bash
orion notification-service start
orion notification-service status
orion notification-service list-watchers
orion notification-service stop
```

## Configuration

Notification settings are in global `~/.orion.yaml` (shared across all workspaces):

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
    card_title: "boss, I want to work"
```

## Card Content

Lark card includes:

- Configurable title from `notifications.lark.card_title`
- Node name and label
- Current wait-input reason
- Latest agent response block extracted from terminal output (`notifications.last_block`)

## Last Block Extraction

Use `notifications.last_block` to adapt different coding agents:

By default, Orion is tuned for OpenAI Codex-style terminal output (`prefix: "• "`), so no extra config is required for Codex.

```yaml
# Codex-style bullet block
notifications:
  last_block:
    enabled: true
    mode: prefix
    prefix: "• "
    max_chars: 1200
```

```yaml
# Kimi-style regex block (example)
notifications:
  last_block:
    enabled: true
    mode: regex
    regex: "(?s)Final Answer:\\s*(.+)$"
    max_chars: 1200
```

## Official Channels

- Local desktop notification
- Feishu/Lark bot notification

## Extensible Provider Architecture

This notification architecture is provider-oriented and reusable across chat systems:

- `Watcher` and wait-input classification stay inside Orion core.
- Delivery adapters can target different notification providers.
- Teams can keep the same detection semantics while swapping transport channels.

Contributions are welcome to add official providers for:

- Slack
- Discord

## Contributing: Notification Providers

If you want to add a new provider, keep this boundary:

- Reuse Orion core watcher and wait-input classification logic.
- Add or extend only the delivery adapter for your target platform.
- Keep provider-specific config isolated so existing channels are unaffected.

Good first targets:

- Slack
- Discord
