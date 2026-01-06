package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/fijdemon/gssh/internal/config"
)

// item 列表项
type item struct {
	server config.Server
}

// formatTimeLocal 将存储的时间字符串转换为当前系统时区并格式化显示
// 期望输入为 RFC3339 格式（例如 2025-01-06T12:34:56+08:00）
func formatTimeLocal(raw string) string {
	if raw == "" {
		return "-"
	}

	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		// 无法解析时，直接原样返回，避免信息丢失
		return raw
	}

	// 转为本地时区，再用更友好的格式展示
	return t.In(time.Local).Format("2006-01-02 15:04:05")
}

func (i item) FilterValue() string {
	// 支持按名称、描述、标签、IP/主机名和用户名筛选
	return i.server.Name + " " +
		i.server.Description + " " +
		i.server.Hostname + " " +
		i.server.User + " " +
		strings.Join(i.server.Tags, " ")
}

func (i item) Title() string {
	return i.server.GetDisplayName()
}

func (i item) Description() string {
	created := formatTimeLocal(i.server.CreatedAt)
	last := formatTimeLocal(i.server.LastUsed)

	// 显示地址、用户名、创建时间和最后登录时间
	// 使用换行符分隔，第一行显示地址和用户名，第二行显示时间信息
	return fmt.Sprintf(
		"%s (%s)\n上次使用: %s | 创建时间: %s\n标签: %s",
		i.server.GetAddress(),
		i.server.User,
		last,
		created,
		strings.Join(i.server.Tags, ","),
	)
}
