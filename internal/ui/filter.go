package ui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

// refreshList 刷新列表，应用搜索、分组和标签过滤
func (m *Model) refreshList() {
	// 应用搜索过滤
	searchTerm := strings.ToLower(m.search.Value())
	filtered := make([]list.Item, 0)

	for _, s := range m.config.Servers {
		// 分组过滤
		if m.selectedGroup != "" && s.Group != m.selectedGroup {
			continue
		}

		// 标签过滤
		if len(m.selectedTags) > 0 {
			matched := false
			for _, tag := range m.selectedTags {
				if slices.Contains(s.Tags, tag) {
					matched = true
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 搜索过滤（支持名称、描述、标签、IP/主机名和用户名）
		if searchTerm != "" {
			// 使用 FilterValue() 方法避免重复构造搜索字符串
			searchable := strings.ToLower(item{server: s}.FilterValue())
			if !strings.Contains(searchable, searchTerm) {
				continue
			}
		}

		filtered = append(filtered, item{server: s})
	}

	m.list.SetItems(filtered)
	m.list.ResetFilter()
}

