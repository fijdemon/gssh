package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fijdemon/gssh/internal/config"
	"github.com/fijdemon/gssh/internal/ssh"
	"gopkg.in/yaml.v3"
)

// SSHSync SSH方式同步
type SSHSync struct {
	config *config.SyncConfig
}

// NewSSHSync 创建SSH同步实例
func NewSSHSync(cfg *config.SyncConfig) *SSHSync {
	return &SSHSync{config: cfg}
}

// Pull 从远程服务器拉取配置
func (s *SSHSync) Pull() (*config.Config, error) {
	// 检查同步配置
	if s.config.SSHHost == "" {
		return nil, fmt.Errorf("同步服务器地址未配置")
	}
	if s.config.SSHUser == "" {
		return nil, fmt.Errorf("SSH 用户名未配置")
	}
	if s.config.SSHKey == "" && s.config.Password == "" {
		return nil, fmt.Errorf("请配置 SSH 密钥路径或密码（在 sync 配置中设置 ssh_key 或 password）")
	}

	// 构建认证配置
	authConfig := ssh.AuthConfig{
		Type:         "auto",
		Password:     s.config.Password,
		IdentityFile: s.config.SSHKey,
	}

	// 创建SSH客户端
	client, err := ssh.NewSSHClient(s.config.SSHHost, s.config.SSHUser, 22, authConfig)
	if err != nil {
		return nil, fmt.Errorf("连接远程服务器失败: %w", err)
	}
	defer client.Close()

	// 执行命令读取远程配置文件
	command := fmt.Sprintf("cat %s", s.config.SSHPath)
	output, err := ssh.ExecuteCommand(client, command)
	if err != nil {
		return nil, fmt.Errorf("读取远程配置失败: %w", err)
	}

	// 解析配置
	var cfg config.Config
	if err := yaml.Unmarshal([]byte(output), &cfg); err != nil {
		return nil, fmt.Errorf("解析远程配置失败: %w", err)
	}

	return &cfg, nil
}

// Push 推送配置到远程服务器
func (s *SSHSync) Push(cfg *config.Config) error {
	// 检查同步配置
	if s.config.SSHHost == "" {
		return fmt.Errorf("同步服务器地址未配置")
	}
	if s.config.SSHUser == "" {
		return fmt.Errorf("SSH 用户名未配置")
	}

	// 序列化配置
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "gssh-config-*.yaml")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	// 使用scp命令复制文件
	remotePath := fmt.Sprintf("%s@%s:%s", s.config.SSHUser, s.config.SSHHost, s.config.SSHPath)

	// 构建 scp 参数
	args := []string{
		"-o", "StrictHostKeyChecking=no",
	}

	if s.config.SSHKey != "" {
		// 使用密钥
		keyPath := s.config.SSHKey
		if keyPath[0] == '~' {
			homeDir, _ := os.UserHomeDir()
			keyPath = filepath.Join(homeDir, keyPath[1:])
		}
		args = append(args, "-i", keyPath)
	}

	// - 如果配置了密钥，则使用密钥无感推送；
	// - 如果未配置密钥，则由 scp 自行处理（可能使用默认密钥或提示用户输入密码）。
	args = append(args, tmpFile.Name(), remotePath)

	cmd := exec.Command("scp", args...)

	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	return nil
}
