package ui

import (
	"fmt"

	"github.com/fijdemon/gssh/internal/config"
	"github.com/fijdemon/gssh/internal/ssh"
)

// connectToServer 连接到服务器
func connectToServer(s config.Server) {
	fmt.Printf("正在连接到 %s (%s)...\n", s.Name, s.GetAddress())

	authConfig := ssh.AuthConfig{
		Type:         s.Auth.Type,
		Password:     s.Auth.Password,
		IdentityFile: s.Auth.IdentityFile,
	}

	if err := ssh.Connect(s.Hostname, s.User, s.Port, authConfig); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	// 更新最后使用时间
	s.UpdateLastUsed()
	cfg, _ := config.Load()
	for i := range cfg.Servers {
		if cfg.Servers[i].Name == s.Name {
			cfg.Servers[i].UpdateLastUsed()
			config.Save(cfg)
			break
		}
	}
}

