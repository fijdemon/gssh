package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fijdemon/gssh/internal/config"
)

// Sync 同步接口
type Sync interface {
	Pull() (*config.Config, error)
	Push(cfg *config.Config) error
}

// NewSync 根据配置创建同步实例
func NewSync(cfg *config.SyncConfig) (Sync, error) {
	switch cfg.Type {
	case "ssh":
		return NewSSHSync(cfg), nil
	case "http":
		// 未来实现
		return nil, fmt.Errorf("HTTP同步尚未实现")
	case "ftp":
		// 未来实现
		return nil, fmt.Errorf("FTP同步尚未实现")
	default:
		return nil, fmt.Errorf("不支持的同步类型: %s", cfg.Type)
	}
}

// Pull 从云端拉取配置
func Pull() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if !cfg.Sync.Enabled {
		return fmt.Errorf("同步功能未启用，请先运行 'gssh init' 配置同步设置")
	}

	// 验证同步配置
	if cfg.Sync.SSHHost == "" {
		return fmt.Errorf("同步配置不完整：缺少 ssh_host，请运行 'gssh init' 重新配置")
	}
	if cfg.Sync.SSHUser == "" {
		return fmt.Errorf("同步配置不完整：缺少 ssh_user，请运行 'gssh init' 重新配置")
	}
	if cfg.Sync.SSHKey == "" && cfg.Sync.Password == "" {
		return fmt.Errorf("同步配置不完整：缺少 ssh_key 或 password，请运行 'gssh init' 重新配置或手动编辑配置文件")
	}

	// 检查密钥文件是否存在
	if cfg.Sync.SSHKey != "" {
		keyPath := cfg.Sync.SSHKey
		if keyPath[0] == '~' {
			homeDir, _ := os.UserHomeDir()
			keyPath = filepath.Join(homeDir, keyPath[1:])
		}
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			return fmt.Errorf("密钥文件不存在: %s，请检查路径或运行 'gssh init' 重新配置", cfg.Sync.SSHKey)
		}
	}

	sync, err := NewSync(&cfg.Sync)
	if err != nil {
		return err
	}

	remoteCfg, err := sync.Pull()
	if err != nil {
		// 提供详细的诊断信息
		fmt.Fprintf(os.Stderr, "\n诊断信息：\n")
		fmt.Fprintf(os.Stderr, "  同步服务器: %s\n", cfg.Sync.SSHHost)
		fmt.Fprintf(os.Stderr, "  SSH 用户: %s\n", cfg.Sync.SSHUser)
		if cfg.Sync.SSHKey != "" {
			fmt.Fprintf(os.Stderr, "  密钥路径: %s\n", cfg.Sync.SSHKey)
		} else {
			fmt.Fprintf(os.Stderr, "  认证方式: 密码\n")
		}
		fmt.Fprintf(os.Stderr, "\n请检查：\n")
		if cfg.Sync.SSHKey != "" {
			fmt.Fprintf(os.Stderr, "  1. 密钥文件是否存在且可读\n")
			fmt.Fprintf(os.Stderr, "  2. 密钥文件权限是否正确（建议 600）\n")
		}
		fmt.Fprintf(os.Stderr, "  3. 是否可以手动 SSH 连接到同步服务器\n")
		fmt.Fprintf(os.Stderr, "  4. 运行 'gssh init' 重新配置同步设置\n\n")
		return fmt.Errorf("拉取配置失败: %w", err)
	}

	// 只使用远程的 servers，保留本地的 sync 配置
	cfg.Servers = remoteCfg.Servers
	cfg.Version = remoteCfg.Version
	cfg.Sync.LastSync = getCurrentTime()

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("配置拉取成功，更新了 %d 个服务器配置\n", len(remoteCfg.Servers))
	return nil
}

// Push 推送配置到云端
func Push() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	if !cfg.Sync.Enabled {
		return fmt.Errorf("同步功能未启用，请先配置同步设置")
	}

	sync, err := NewSync(&cfg.Sync)
	if err != nil {
		return err
	}

	// 创建只包含 servers 的配置用于推送（不包含 sync 配置）
	pushCfg := &config.Config{
		Version: cfg.Version,
		Servers: cfg.Servers,
		// Sync 部分不推送，由各客户端自己维护
	}

	if err := sync.Push(pushCfg); err != nil {
		return fmt.Errorf("推送配置失败: %w", err)
	}

	// 更新最后同步时间
	cfg.Sync.LastSync = getCurrentTime()
	config.Save(cfg)

	fmt.Printf("配置推送成功，推送了 %d 个服务器配置\n", len(cfg.Servers))
	return nil
}

func getCurrentTime() string {
	return time.Now().Format(time.RFC3339)
}
