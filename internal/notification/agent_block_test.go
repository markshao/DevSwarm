package notification

import (
	"strings"
	"testing"
)

func TestExtractLastBlockByPrefixKeepsBulletAndMultilineBody(t *testing.T) {
	screen := `some logs
• 好，我在线等你结果。
  你先重点看两点：

  1. 点 Ack 后：PENDING 还在、MUTED=muted。
  2. 点 Reply 后：目标 node 里是否收到文本并自动回车。

  有任何异常，直接把现象和这两条命令输出贴我就行：

  orion notification-service status
  orion notification-service list-watchers
`

	got := extractLastBlockByPrefix(screen, "• ")
	if !strings.HasPrefix(got, "• 好，我在线等你结果。") {
		t.Fatalf("expected to keep bullet prefix, got: %q", got)
	}
	if !strings.Contains(got, "1. 点 Ack 后") || !strings.Contains(got, "orion notification-service status") {
		t.Fatalf("expected multiline body in block, got: %q", got)
	}
}
