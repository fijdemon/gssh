package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config 主配置结构
type Config struct {
	Version string     `yaml:"version"`
	Sync    SyncConfig `yaml:"sync"`
	Servers []Server   `yaml:"servers"`
}

// SyncConfig 同步配置
type SyncConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Type     string `yaml:"type"`      // ssh, http, ftp (未来扩展)
	SSHHost  string `yaml:"ssh_host"`  // SSH同步时的主机地址
	SSHUser  string `yaml:"ssh_user"`  // SSH同步时的用户名
	SSHPath  string `yaml:"ssh_path"`  // SSH同步时的远程路径
	SSHKey   string `yaml:"ssh_key"`   // SSH密钥路径（可选）
	Password string `yaml:"password"`  // 密码（可选，用于SSH认证）
	AutoSync bool   `yaml:"auto_sync"` // 启动时自动同步
	LastSync string `yaml:"last_sync"` // 最后同步时间
}

// Server 服务器配置
type Server struct {
	Name        string     `yaml:"name"`
	Hostname    string     `yaml:"hostname"`
	User        string     `yaml:"user"`
	Port        int        `yaml:"port"`
	Description string     `yaml:"description"`
	Tags        []string   `yaml:"tags"`
	Group       string     `yaml:"group"` // 分组
	Auth        AuthConfig `yaml:"auth"`
	LastUsed    string     `yaml:"last_used"`
	CreatedAt   string     `yaml:"created_at"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Type         string `yaml:"type"`          // auto, password, key
	Password     string `yaml:"password"`      // 密码（不加密存储）
	IdentityFile string `yaml:"identity_file"` // 密钥文件路径
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	configDir := filepath.Join(homeDir, ".gssh")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return filepath.Join(configDir, "config.yaml"), nil
}

// GetBackupPath 获取备份文件路径
func GetBackupPath() (string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return "", err
	}
	return configPath + ".backup", nil
}

// UpdateLastUsed 更新服务器的最后使用时间
func (s *Server) UpdateLastUsed() {
	s.LastUsed = time.Now().Format(time.RFC3339)
}

// GetDisplayName 获取显示名称（包含描述）
func (s *Server) GetDisplayName() string {
	return fmt.Sprintf("[%s] %s", s.Name, s.Description)
}

// GetAddress 获取完整地址
func (s *Server) GetAddress() string {
	if s.Port > 0 && s.Port != 22 {
		return fmt.Sprintf("%s:%d", s.Hostname, s.Port)
	}
	return s.Hostname
}
