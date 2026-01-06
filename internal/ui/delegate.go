package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// multiLineDelegate 支持多行描述的自定义 delegate
type multiLineDelegate struct {
	list.DefaultDelegate
	searchTerm string // 当前搜索词，用于高亮匹配文本
}

// Height 返回每个列表项的高度（行数）
// 由于每个 item 的描述有 3 行（Title 1行 + Description 3行），总共 4 行
func (d multiLineDelegate) Height() int {
	return 4
}

// Spacing 返回列表项之间的间距
func (d multiLineDelegate) Spacing() int {
	return 1
}

// highlightText 高亮文本中匹配搜索词的部分
func (d multiLineDelegate) highlightText(text string, isSelected bool) string {
	if d.searchTerm == "" {
		return text
	}

	searchTerm := strings.ToLower(d.searchTerm)
	lowerText := strings.ToLower(text)

	// 如果没有匹配，直接返回原文本
	if !strings.Contains(lowerText, searchTerm) {
		return text
	}

	// 高亮样式
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("236")).
		Bold(true)

	var result strings.Builder
	textRunes := []rune(text)
	lowerRunes := []rune(lowerText)
	searchRunes := []rune(searchTerm)

	i := 0
	for i < len(lowerRunes) {
		// 查找匹配位置
		matchIndex := -1
		for j := i; j <= len(lowerRunes)-len(searchRunes); j++ {
			matched := true
			for k := 0; k < len(searchRunes); k++ {
				if lowerRunes[j+k] != searchRunes[k] {
					matched = false
					break
				}
			}
			if matched {
				matchIndex = j
				break
			}
		}

		if matchIndex == -1 {
			// 没有更多匹配，添加剩余文本
			result.WriteString(string(textRunes[i:]))
			break
		}

		// 添加匹配前的文本
		if matchIndex > i {
			result.WriteString(string(textRunes[i:matchIndex]))
		}

		// 添加高亮的匹配文本
		matchedText := string(textRunes[matchIndex : matchIndex+len(searchRunes)])
		result.WriteString(highlightStyle.Render(matchedText))

		// 移动到匹配后
		i = matchIndex + len(searchRunes)
	}

	return result.String()
}

// Render 重写渲染方法以支持多行描述和高亮匹配文本
func (d multiLineDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	var (
		title, desc string
		matched     = index == m.Index()
	)

	if i, ok := listItem.(item); ok {
		title = i.Title()
		desc = i.Description()
	} else {
		title = listItem.FilterValue()
	}

	// 处理多行描述
	descLines := strings.Split(desc, "\n")

	if matched {
		// 高亮标题中的匹配文本
		highlightedTitle := d.highlightText(title, true)
		title = d.Styles.SelectedTitle.Render(highlightedTitle)
		// 为每一行描述应用选中样式并高亮匹配文本
		styledDesc := make([]string, len(descLines))
		for i, line := range descLines {
			highlightedLine := d.highlightText(line, true)
			styledDesc[i] = d.Styles.SelectedDesc.Render(highlightedLine)
		}
		desc = strings.Join(styledDesc, "\n")
	} else {
		// 高亮标题中的匹配文本
		highlightedTitle := d.highlightText(title, false)
		title = d.Styles.NormalTitle.Render(highlightedTitle)
		styledDesc := make([]string, len(descLines))
		for i, line := range descLines {
			highlightedLine := d.highlightText(line, false)
			styledDesc[i] = d.Styles.NormalDesc.Render(highlightedLine)
		}
		desc = strings.Join(styledDesc, "\n")
	}

	result := lipgloss.JoinVertical(lipgloss.Left, title, desc)
	fmt.Fprint(w, result)
}
