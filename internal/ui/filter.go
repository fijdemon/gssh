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

	// 更新 delegate 的搜索词以支持高亮
	m.delegate.searchTerm = searchTerm
	m.list.SetDelegate(m.delegate)

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

		// 搜索过滤：按照实际显示的内容进行筛选（用户看到什么就能筛选什么）
		if searchTerm != "" {
			// 创建 item 以获取实际显示的内容
			it := item{server: s}
			// 使用实际显示的 Title 和 Description 进行筛选
			title := it.Title()                    // 例如: "[服务器名] 描述"
			desc := it.Description()               // 例如: "192.168.1.1:22 (root)\n上次使用: ... | 创建时间: ...\n标签: ..."
			// 将多行描述合并为单行，移除换行符，用于搜索
			descForSearch := desc
			// 组合标题和描述进行搜索
			searchable := strings.ToLower(title + "\n" + descForSearch)
			if !strings.Contains(searchable, searchTerm) {
				continue
			}
		}

		filtered = append(filtered, item{server: s})
	}

	m.list.SetItems(filtered)
	m.list.ResetFilter()
}
