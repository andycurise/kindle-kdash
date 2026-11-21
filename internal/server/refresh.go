package server

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/andycurise/kindle-kdash/internal/config"
)

// refreshOverSSH only runs a fixed client command. Configuration and rendered
// images travel over HTTP, so browser values never become shell source text.
func refreshOverSSH(ctx context.Context, cfg *config.Config) (string, error) {
	ssh := cfg.KindleSSH
	if !ssh.Enabled {
		return "", fmt.Errorf("未启用 SSH 立即刷新")
	}
	if err := cfg.Validate(); err != nil {
		return "", fmt.Errorf("SSH 配置无效: %w", err)
	}
	host := ssh.Host
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	target := ssh.User + "@" + host
	args := []string{
		"-i", cfg.ResolvePath(ssh.PrivateKeyPath),
		"-p", strconv.Itoa(ssh.Port),
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + cfg.ResolvePath(ssh.KnownHostsPath),
		"-o", "ConnectTimeout=4",
		"--",
		target,
		"sh /mnt/us/kdash/sync-screen.sh once",
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		return text, fmt.Errorf("Kindle 立即刷新命令失败: %w (%s)", err, text)
	}
	return text, nil
}
