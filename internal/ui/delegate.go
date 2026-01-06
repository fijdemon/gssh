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

// Render 重写渲染方法以支持多行描述
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
		title = d.Styles.SelectedTitle.Render(title)
		// 为每一行描述应用选中样式
		styledDesc := make([]string, len(descLines))
		for i, line := range descLines {
			styledDesc[i] = d.Styles.SelectedDesc.Render(line)
		}
		desc = strings.Join(styledDesc, "\n")
	} else {
		title = d.Styles.NormalTitle.Render(title)
		styledDesc := make([]string, len(descLines))
		for i, line := range descLines {
			styledDesc[i] = d.Styles.NormalDesc.Render(line)
		}
		desc = strings.Join(styledDesc, "\n")
	}

	result := lipgloss.JoinVertical(lipgloss.Left, title, desc)
	fmt.Fprint(w, result)
}
