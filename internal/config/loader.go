package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Load 加载配置文件
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := NewDefaultConfig()
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("创建默认配置失败: %w", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Version == "" {
		cfg.Version = "1.0"
	}

	return &cfg, nil
}

// Save 保存配置文件
func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// 备份现有配置
	if _, err := os.Stat(configPath); err == nil {
		backupPath, err := GetBackupPath()
		if err == nil {
			data, _ := os.ReadFile(configPath)
			os.WriteFile(backupPath, data, 0644)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Sync: SyncConfig{
			Enabled:  false,
			Type:     "ssh",
			AutoSync: false,
		},
		Servers: []Server{},
	}
}

// AddServer 添加服务器
func (c *Config) AddServer(server Server) error {
	// 检查名称是否已存在
	for _, s := range c.Servers {
		if s.Name == server.Name {
			return fmt.Errorf("服务器名称 '%s' 已存在", server.Name)
		}
	}

	// 设置默认值
	if server.Port == 0 {
		server.Port = 22
	}
	if server.Auth.Type == "" {
		server.Auth.Type = "auto"
	}
	if server.CreatedAt == "" {
		server.CreatedAt = time.Now().Format(time.RFC3339)
	}

	c.Servers = append(c.Servers, server)
	return nil
}

// GetServer 获取服务器配置
func (c *Config) GetServer(name string) (*Server, error) {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i], nil
		}
	}
	return nil, fmt.Errorf("服务器 '%s' 不存在", name)
}

// DeleteServer 删除服务器
func (c *Config) DeleteServer(name string) error {
	for i, s := range c.Servers {
		if s.Name == name {
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("服务器 '%s' 不存在", name)
}

// FilterServers 根据标签和分组过滤服务器
func (c *Config) FilterServers(tags []string, group string) []Server {
	var result []Server

	for _, s := range c.Servers {
		// 分组过滤
		if group != "" && s.Group != group {
			continue
		}

		// 标签过滤
		if len(tags) > 0 {
			matched := false
			for _, tag := range tags {
				for _, serverTag := range s.Tags {
					if serverTag == tag {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		result = append(result, s)
	}

	return result
}

// GetGroups 获取所有分组
func (c *Config) GetGroups() []string {
	groupMap := make(map[string]bool)
	for _, s := range c.Servers {
		if s.Group != "" {
			groupMap[s.Group] = true
		}
	}

	groups := make([]string, 0, len(groupMap))
	for g := range groupMap {
		groups = append(groups, g)
	}
	return groups
}

// GetTags 获取所有标签
func (c *Config) GetTags() []string {
	tagMap := make(map[string]bool)
	for _, s := range c.Servers {
		for _, tag := range s.Tags {
			tagMap[tag] = true
		}
	}

	tags := make([]string, 0, len(tagMap))
	for t := range tagMap {
		tags = append(tags, t)
	}
	return tags
}
