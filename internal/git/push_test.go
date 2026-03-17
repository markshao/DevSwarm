package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepoWithRemote 创建一个带有远程仓库的临时 Git 仓库
func setupTestRepoWithRemote(t *testing.T) (localPath, remotePath string, cleanup func()) {
	t.Helper()

	// 创建远程仓库（bare 仓库）
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to git init --bare: %v, output: %s", err, output)
	}

	// 创建本地仓库
	localDir, err := os.MkdirTemp("", "orion-local-test")
	if err != nil {
		os.RemoveAll(remoteDir)
		t.Fatalf("failed to create temp local dir: %v", err)
	}

	cmd = exec.Command("git", "init")
	cmd.Dir = localDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to git init: %v, output: %s", err, output)
	}

	// 配置用户信息
	_ = exec.Command("git", "-C", localDir, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", localDir, "config", "user.name", "Test User").Run()

	// 创建初始提交
	readme := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test Repo"), 0644); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "-C", localDir, "add", ".")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to git add")
	}

	cmd = exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to git commit")
	}

	// 添加远程仓库
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = localDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to add remote: %v, output: %s", err, output)
	}

	// 推送到远程
	cmd = exec.Command("git", "push", "-u", "origin", "main")
	cmd.Dir = localDir
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
		t.Fatalf("failed to push to remote: %v, output: %s", err, output)
	}

	cleanup = func() {
		os.RemoveAll(remoteDir)
		os.RemoveAll(localDir)
	}

	return localDir, remoteDir, cleanup
}

func TestPushBranch(t *testing.T) {
	localPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 在本地创建新分支并提交
	newFile := filepath.Join(localPath, "feature.txt")
	if err := os.WriteFile(newFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// 创建新分支
	if err := CreateBranch(localPath, "feature/test", "main"); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// 测试 PushBranch
	err := PushBranch(localPath, "feature/test")
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程仓库是否有该分支（使用 ls-remote 而不是 branch -r）
	cmd = exec.Command("git", "ls-remote", remotePath, "refs/heads/feature/test")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote branch: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/feature/test") {
		t.Errorf("remote branch 'feature/test' not found after push. Output: %s", string(output))
	}
}

func TestPushBranchAlreadyExists(t *testing.T) {
	localPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 在本地创建新分支并提交
	newFile := filepath.Join(localPath, "feature.txt")
	if err := os.WriteFile(newFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// 推送 main 分支（已经存在）
	err := PushBranch(localPath, "main")
	if err != nil {
		t.Fatalf("PushBranch failed for existing branch: %v", err)
	}

	// 验证远程仓库
	cmd = exec.Command("git", "ls-remote", remotePath, "main")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/main") {
		t.Errorf("remote branch 'main' not found after push. Output: %s", string(output))
	}
}

func TestPushBranchNonExistent(t *testing.T) {
	localPath, _, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 测试推送不存在的分支
	err := PushBranch(localPath, "non-existent-branch")
	if err == nil {
		t.Error("expected error for non-existent branch, got nil")
	}

	// 错误信息应该包含相关提示
	if !strings.Contains(err.Error(), "git push failed") {
		t.Errorf("expected 'git push failed' in error message, got: %v", err)
	}
}

func TestPushBranchWithMultipleCommits(t *testing.T) {
	localPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 创建多个提交
	for i := 0; i < 3; i++ {
		filename := filepath.Join(localPath, "file"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(filename, []byte("content "+string(rune('a'+i))), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = localPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to git add: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "Add file "+string(rune('a'+i)))
		cmd.Dir = localPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to git commit: %v", err)
		}
	}

	// 创建新分支
	if err := CreateBranch(localPath, "feature/multi-commit", "main"); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// 推送分支
	err := PushBranch(localPath, "feature/multi-commit")
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程仓库是否有该分支（使用 ls-remote）
	cmd := exec.Command("git", "ls-remote", remotePath, "refs/heads/feature/multi-commit")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote branch: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/feature/multi-commit") {
		t.Errorf("remote branch 'feature/multi-commit' not found after push. Output: %s", string(output))
	}
}

func TestPushBranchFromDifferentBase(t *testing.T) {
	localPath, remotePath, cleanup := setupTestRepoWithRemote(t)
	defer cleanup()

	// 创建基于 main 的 feature 分支
	if err := CreateBranch(localPath, "feature/base", "main"); err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// 在 feature 分支上创建提交
	cmd := exec.Command("git", "checkout", "feature/base")
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to checkout: %v, output: %s", err, output)
	}

	newFile := filepath.Join(localPath, "feature-file.txt")
	if err := os.WriteFile(newFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// 推送分支
	err := PushBranch(localPath, "feature/base")
	if err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	// 验证远程仓库（使用 ls-remote）
	cmd = exec.Command("git", "ls-remote", remotePath, "refs/heads/feature/base")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to verify remote branch: %v", err)
	}

	if !strings.Contains(string(output), "refs/heads/feature/base") {
		t.Errorf("remote branch 'feature/base' not found after push. Output: %s", string(output))
	}
}
