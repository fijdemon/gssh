package cmd

import (
	"fmt"
	"os"

	"github.com/fijdemon/gssh/internal/config"
	"github.com/fijdemon/gssh/internal/util"
)

// RunInit 执行初始化操作
func RunInit() error {
	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("获取配置路径失败: %w", err)
	}

	// 检查配置文件是否已存在
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("配置文件已存在: %s\n", configPath)
		fmt.Print("是否要覆盖现有配置? (y/N): ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("已取消初始化")
			return nil
		}
	}

	// 创建默认配置
	cfg := config.NewDefaultConfig()

	func() {
		// 询问是否设置同步
		fmt.Print("是否要设置云端同步? (y/N): ")
		var syncAnswer string
		fmt.Scanln(&syncAnswer)
		if !util.IsYes(syncAnswer) {
			return
		}

		fmt.Print("同步服务器地址 (例如: sync.example.com): ")
		var host string
		fmt.Scanln(&host)
		if host == "" {
			fmt.Print("同步服务器地址为空,取消同步设置")
			return
		}
		cfg.Sync.Enabled = true
		cfg.Sync.Type = "ssh"
		cfg.Sync.SSHHost = host

		fmt.Print("SSH 用户名: ")
		var user string
		fmt.Scanln(&user)
		cfg.Sync.SSHUser = user

		fmt.Print("远程配置文件路径 (默认: ~/.gssh/config.yaml): ")
		var path string
		fmt.Scanln(&path)
		if path == "" {
			path = "~/.gssh/config.yaml"
		}
		cfg.Sync.SSHPath = path

		fmt.Print("使用密钥认证还是密码认证? (key/password) [key]: ")
		var authType string
		fmt.Scanln(&authType)
		if authType == "password" || authType == "p" {
			fmt.Print("SSH 密码: ")
			var password string
			fmt.Scanln(&password)
			cfg.Sync.Password = password
		} else {
			fmt.Print("SSH 密钥路径 (例如: ~/.ssh/id_rsa) [~/.ssh/id_rsa]: ")
			var keyPath string
			fmt.Scanln(&keyPath)
			if keyPath == "" {
				keyPath = "~/.ssh/id_rsa"
			}
			cfg.Sync.SSHKey = keyPath
		}

		fmt.Print("启动时自动同步? (y/N): ")
		var autoSync string
		fmt.Scanln(&autoSync)
		cfg.Sync.AutoSync = util.IsYes(autoSync)
	}()

	// 保存配置
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("\n✅ 配置初始化成功！\n")
	fmt.Printf("配置文件位置: %s\n", configPath)
	fmt.Printf("\n接下来你可以:\n")
	fmt.Printf("  1. 手动编辑配置文件添加服务器\n")
	fmt.Printf("  2. 运行 'gssh' 打开交互式界面\n")
	fmt.Printf("  3. 运行 'gssh pull' 从云端拉取配置\n")

	return nil
}
