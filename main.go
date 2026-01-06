package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/fijdemon/gssh/cmd"
	"github.com/fijdemon/gssh/internal/config"
	"github.com/fijdemon/gssh/internal/ssh"
)

// getVersion 获取版本号
// 从 runtime/debug.ReadBuildInfo() 获取模块版本（适用于 go install 安装的情况）
func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		// 检查 Main 模块是否是当前模块
		if info.Main.Path == "github.com/fijdemon/gssh" {
			// 如果是从 go install @version 安装的，Main.Version 会是版本号
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				return info.Main.Version
			}
		}
	}
	// 默认返回 "dev"（开发版本或本地构建）
	return "dev"
}

func main() {
	if len(os.Args) < 2 {
		// 无参数时打开交互式界面
		if err := cmd.RunInteractive(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := os.Args[1]
	var err error

	switch command {
	case "init":
		err = cmd.RunInit()
	case "pull":
		err = cmd.RunPull()
	case "push":
		err = cmd.RunPush()
	case "version":
		fmt.Printf("gssh version %s\n", getVersion())
	case "help", "-h", "--help":
		printUsage()
	default:
		// 尝试作为服务器名称直接登录
		if err := connectToServerByName(command); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			fmt.Fprintf(os.Stderr, "未知命令: %s\n", command)
			printUsage()
			os.Exit(1)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("gssh - Go 版本 SSH 服务器管理工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  gssh                   打开交互式界面")
	fmt.Println("  gssh init               初始化配置文件")
	fmt.Println("  gssh <server-name>     直接登录指定服务器")
	fmt.Println("  gssh pull               从云端拉取配置")
	fmt.Println("  gssh push               推送配置到云端")
	fmt.Println("  gssh version            显示版本信息")
	fmt.Println("  gssh help               显示帮助信息")
	fmt.Println()
}

func connectToServerByName(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	server, err := cfg.GetServer(name)
	if err != nil {
		return err
	}

	fmt.Printf("正在连接到 %s (%s)...\n", server.Name, server.GetAddress())

	authConfig := ssh.AuthConfig{
		Type:         server.Auth.Type,
		Password:     server.Auth.Password,
		IdentityFile: server.Auth.IdentityFile,
	}

	if err := ssh.Connect(server.Hostname, server.User, server.Port, authConfig); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	// 更新最后使用时间
	server.UpdateLastUsed()
	for i := range cfg.Servers {
		if cfg.Servers[i].Name == server.Name {
			cfg.Servers[i].UpdateLastUsed()
			config.Save(cfg)
			break
		}
	}

	return nil
}
