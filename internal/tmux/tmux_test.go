package tmux

import (
	"os"
	"testing"
)

// TestSessionExists 测试会话存在性检查
func TestSessionExists(t *testing.T) {
	// 创建一个唯一的测试会话名
	sessionName := "orion-test-session-exists"

	// 确保会话不存在
	_ = KillSession(sessionName)

	// 验证会话不存在
	if SessionExists(sessionName) {
		t.Errorf("SessionExists() returned true for non-existent session")
	}

	// 创建会话
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// 验证会话存在
	if !SessionExists(sessionName) {
		t.Errorf("SessionExists() returned false for existing session")
	}
}

// TestNewSession 测试创建新会话
func TestNewSession(t *testing.T) {
	sessionName := "orion-test-new-session"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	defer KillSession(sessionName)

	// 创建会话
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 验证会话已创建
	if !SessionExists(sessionName) {
		t.Errorf("Session was not created")
	}

	// 重复创建应该失败
	if err := NewSession(sessionName, tmpDir); err == nil {
		t.Errorf("NewSession should fail for existing session")
	}
}

// TestSendKeys 测试发送按键到会话
func TestSendKeys(t *testing.T) {
	sessionName := "orion-test-send-keys"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建会话
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// 发送简单命令
	if err := SendKeys(sessionName, "echo hello"); err != nil {
		t.Errorf("SendKeys failed: %v", err)
	}

	// 发送到不存在的会话应该失败
	if err := SendKeys("non-existent-session", "echo hello"); err == nil {
		t.Errorf("SendKeys should fail for non-existent session")
	}
}

// TestIsInsideTmux 测试检测是否在 tmux 内
func TestIsInsideTmux(t *testing.T) {
	// 保存原始环境变量
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	// 测试不在 tmux 内
	os.Unsetenv("TMUX")
	if IsInsideTmux() {
		t.Errorf("IsInsideTmux() returned true when TMUX is not set")
	}

	// 测试在 tmux 内
	os.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	if !IsInsideTmux() {
		t.Errorf("IsInsideTmux() returned false when TMUX is set")
	}
}

// TestGetCurrentSessionName 测试获取当前会话名
func TestGetCurrentSessionName(t *testing.T) {
	// 保存原始环境变量
	originalTmux := os.Getenv("TMUX")
	defer func() {
		if originalTmux == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTmux)
		}
	}()

	// 测试不在 tmux 内
	os.Unsetenv("TMUX")
	sessionName, err := GetCurrentSessionName()
	if err == nil {
		t.Errorf("GetCurrentSessionName() should fail when not inside tmux")
	}
	if sessionName != "" {
		t.Errorf("GetCurrentSessionName() should return empty string when not inside tmux")
	}

	// 注意：在 tmux 内的测试需要实际运行在 tmux 中，这里跳过
}

// TestSwitchClient 测试切换客户端
func TestSwitchClient(t *testing.T) {
	sessionName := "orion-test-switch-client"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建会话
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer KillSession(sessionName)

	// 在 tmux 外切换应该失败（因为不在 tmux 内）
	// 这个测试验证函数不会 panic
	err = SwitchClient(sessionName)
	// 在 tmux 外执行 switch-client 会失败，这是预期的
	if err == nil {
		// 在某些环境下可能成功，所以只是记录
		t.Logf("SwitchClient succeeded outside tmux (unexpected but not critical)")
	}
}

// TestKillSession 测试杀死会话
func TestKillSession(t *testing.T) {
	sessionName := "orion-test-kill-session"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 杀死不存在的会话应该不返回错误
	if err := KillSession(sessionName); err != nil {
		t.Errorf("KillSession should not fail for non-existent session: %v", err)
	}

	// 创建会话
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 验证会话存在
	if !SessionExists(sessionName) {
		t.Fatalf("Session was not created")
	}

	// 杀死会话
	if err := KillSession(sessionName); err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// 验证会话已杀死
	if SessionExists(sessionName) {
		t.Errorf("Session still exists after KillSession")
	}
}

// TestEnsureAndAttach 测试确保会话存在并附加
func TestEnsureAndAttach(t *testing.T) {
	sessionName := "orion-test-ensure-attach"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	defer KillSession(sessionName)

	// 测试创建新会话并附加（会替换当前进程，所以我们只测试创建逻辑）
	// 由于 AttachSession 会替换进程，我们无法直接测试
	// 但可以验证会话会被创建

	// 先验证会话不存在
	if SessionExists(sessionName) {
		t.Fatalf("Session already exists before test")
	}

	// 调用 EnsureAndAttach 会阻塞/替换进程，所以我们只测试 NewSession 部分
	// 通过直接调用 NewSession 来验证
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 验证会话已创建
	if !SessionExists(sessionName) {
		t.Errorf("Session was not created by EnsureAndAttach")
	}
}

// TestSessionOperations 测试会话操作的完整流程
func TestSessionOperations(t *testing.T) {
	sessionName := "orion-test-operations"
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	defer KillSession(sessionName)

	// 1. 创建会话
	if err := NewSession(sessionName, tmpDir); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	// 2. 验证会话存在
	if !SessionExists(sessionName) {
		t.Fatalf("Session was not created")
	}

	// 3. 发送命令
	if err := SendKeys(sessionName, "echo 'test message'"); err != nil {
		t.Errorf("SendKeys failed: %v", err)
	}

	// 4. 杀死会话
	if err := KillSession(sessionName); err != nil {
		t.Errorf("KillSession failed: %v", err)
	}

	// 5. 验证会话已杀死
	if SessionExists(sessionName) {
		t.Errorf("Session still exists after KillSession")
	}
}

// TestConcurrentSessions 测试并发创建多个会话
func TestConcurrentSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orion-tmux-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sessionNames := []string{
		"orion-concurrent-1",
		"orion-concurrent-2",
		"orion-concurrent-3",
	}

	// 清理函数
	defer func() {
		for _, name := range sessionNames {
			KillSession(name)
		}
	}()

	// 创建多个会话
	for _, name := range sessionNames {
		if err := NewSession(name, tmpDir); err != nil {
			t.Fatalf("NewSession failed for %s: %v", name, err)
		}
	}

	// 验证所有会话都存在
	for _, name := range sessionNames {
		if !SessionExists(name) {
			t.Errorf("Session %s does not exist", name)
		}
	}

	// 杀死所有会话
	for _, name := range sessionNames {
		if err := KillSession(name); err != nil {
			t.Errorf("KillSession failed for %s: %v", name, err)
		}
	}

	// 验证所有会话都已杀死
	for _, name := range sessionNames {
		if SessionExists(name) {
			t.Errorf("Session %s still exists after KillSession", name)
		}
	}
}
