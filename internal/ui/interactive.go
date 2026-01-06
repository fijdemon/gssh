package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fijdemon/gssh/internal/config"
	"github.com/fijdemon/gssh/internal/ssh"
)

// Model UI模型
type Model struct {
	list          list.Model
	servers       []config.Server
	config        *config.Config
	search        textinput.Model
	searchMode    bool
	selectedGroup string
	selectedTags  []string
	width         int
	height        int
	formMode      bool
	form          FormModel
	deleteConfirm bool
	pendingServer *config.Server // 待连接的服务器，在退出tea后执行
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return nil
}

// Update 更新模型
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.formMode {
			m.form.width = msg.Width
			m.form.height = msg.Height
		} else {
			m.list.SetWidth(msg.Width)
			m.list.SetHeight(msg.Height - 10)
		}
		return m, nil

	case tea.KeyMsg:
		// 表单模式
		if m.formMode {
			var cmd tea.Cmd
			var updated tea.Model
			updated, cmd = m.form.Update(msg)
			if formModel, ok := updated.(FormModel); ok {
				m.form = formModel
				// 检查表单是否要退出
				if m.form.quitting {
					m.formMode = false
					m.refreshList()
					return m, nil
				}
			} else {
				// 表单已退出，返回列表模式
				m.formMode = false
				m.refreshList()
			}
			return m, cmd
		}

		// 删除确认模式
		if m.deleteConfirm {
			switch msg.String() {
			case "y", "Y":
				selectedItem := m.list.SelectedItem()
				if item, ok := selectedItem.(item); ok {
					m.config.DeleteServer(item.server.Name)
					config.Save(m.config)
					m.refreshList()
				}
				m.deleteConfirm = false
				return m, nil
			case "n", "N", "esc":
				m.deleteConfirm = false
				return m, nil
			}
			return m, nil
		}

		// 搜索模式
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.search.Blur()
				return m, nil
			case "enter":
				m.searchMode = false
				m.search.Blur()
				// 应用搜索过滤
				return m, nil
			case "backspace":
				// 如果搜索框为空，退出搜索模式
				if m.search.Value() == "" {
					m.searchMode = false
					m.search.Blur()
					return m, nil
				}
				// 否则正常处理删除
				fallthrough
			default:
				var cmd tea.Cmd
				m.search, cmd = m.search.Update(msg)
				m.refreshList()
				return m, cmd
			}
		}

		// 正常模式
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "/":
			m.searchMode = true
			m.search.Focus()
			// 取消列表选中
			if len(m.list.Items()) > 0 {
				m.list.Select(-1)
			}
			m.refreshList()
			return m, textinput.Blink

		case "j":
			m.list.CursorDown()
			return m, nil

		case "k":
			m.list.CursorUp()
			return m, nil

		case "enter":
			// 登录选中的服务器
			selectedItem := m.list.SelectedItem()
			if item, ok := selectedItem.(item); ok {
				// 缓存服务器，在退出tea后执行连接
				serverCopy := item.server
				m.pendingServer = &serverCopy
				return m, tea.Quit
			}
			return m, nil

		case "a":
			// 添加服务器
			m.formMode = true
			m.form = NewFormModel(nil, func(server config.Server) error {
				if err := m.config.AddServer(server); err != nil {
					return err
				}
				return config.Save(m.config)
			}, func() {
				m.formMode = false
			})
			// 设置表单的宽高
			m.form.width = m.width
			m.form.height = m.height
			return m, nil

		case "d":
			// 删除服务器
			if len(m.config.Servers) == 0 {
				return m, nil
			}
			selectedItem := m.list.SelectedItem()
			if _, ok := selectedItem.(item); ok {
				m.deleteConfirm = true
			}
			return m, nil

		case "e":
			// 编辑服务器
			if len(m.config.Servers) == 0 {
				return m, nil
			}
			selectedItem := m.list.SelectedItem()
			if item, ok := selectedItem.(item); ok {
				m.formMode = true
				serverCopy := item.server
				m.form = NewFormModel(&serverCopy, func(server config.Server) error {
					// 删除旧服务器
					m.config.DeleteServer(item.server.Name)
					// 添加新服务器
					if err := m.config.AddServer(server); err != nil {
						return err
					}
					return config.Save(m.config)
				}, func() {
					m.formMode = false
				})
				// 设置表单的宽高
				m.form.width = m.width
				m.form.height = m.height
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View 渲染视图
func (m Model) View() string {
	if m.width == 0 {
		return "加载中..."
	}

	// 表单模式
	if m.formMode {
		return m.form.View()
	}

	// 删除确认
	if m.deleteConfirm {
		selectedItem := m.list.SelectedItem()
		if item, ok := selectedItem.(item); ok {
			return fmt.Sprintf("\n确认删除服务器 '%s'? (y/n)\n", item.server.Name)
		}
	}

	var b strings.Builder

	// 标题
	titleSeparatorLen := m.width
	if titleSeparatorLen > 0 {
		b.WriteString(strings.Repeat("─", titleSeparatorLen))
	}
	b.WriteString("\n")
	title := "gssh - SSH 快速登录工具"
	padding := max((m.width-len(title))/2, 0)
	if padding > 0 {
		b.WriteString(strings.Repeat(" ", padding))
	}
	b.WriteString(title)
	titleRemaining := m.width - len(title) - padding
	if titleRemaining > 0 {
		b.WriteString(strings.Repeat(" ", titleRemaining))
	}
	b.WriteString("\n")
	if titleSeparatorLen > 0 {
		b.WriteString(strings.Repeat("─", titleSeparatorLen))
	}
	b.WriteString("\n")

	// 搜索框
	if m.searchMode {
		b.WriteString(" 筛选: ")
		searchView := m.search.View()
		b.WriteString(searchView)
		b.WriteString("\n")

		b.WriteString(" 回车确认\n")
	} else {
		b.WriteString(" 筛选: ")
		searchView := m.search.View()
		b.WriteString(searchView)
		b.WriteString("\n")
		// 显示搜索提示

		b.WriteString(" 按 / 搜索\n")
	}

	separatorLen := m.width
	if separatorLen > 0 {
		b.WriteString(strings.Repeat("─", separatorLen))
	}
	b.WriteString("\n")

	// 列表
	if !m.searchMode && m.list.SelectedItem() == nil {
		if len(m.list.Items()) > 0 {
			m.list.Select(0)
		}
	}
	b.WriteString(m.list.View())
	b.WriteString("\n")

	// 底部操作提示
	if separatorLen > 0 {
		b.WriteString(strings.Repeat("─", separatorLen))
	}
	b.WriteString("\n")
	help := " 操作: j/k 移动 | Enter 登录 | / 搜索 | a 添加 | d 删除 | e 编辑 | q 退出"
	b.WriteString(help)
	b.WriteString("\n")
	if separatorLen > 0 {
		b.WriteString(strings.Repeat("─", separatorLen))
	}
	b.WriteString("\n")

	return b.String()
}

// item 列表项
type item struct {
	server config.Server
}

func (i item) FilterValue() string {
	return i.server.Name + " " + i.server.Description + " " + strings.Join(i.server.Tags, " ")
}

func (i item) Title() string {
	return i.server.GetDisplayName()
}

func (i item) Description() string {
	return fmt.Sprintf("%s (%s)", i.server.GetAddress(), i.server.User)
}

// NewModel 创建新的UI模型
func NewModel() (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建列表项
	items := make([]list.Item, 0, len(cfg.Servers))
	for _, s := range cfg.Servers {
		items = append(items, item{server: s})
	}

	// 创建列表
	delegate := list.NewDefaultDelegate()
	// 设置选中样式（使用lipgloss颜色）
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("212"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle

	// 创建搜索输入框
	search := textinput.New()
	search.Placeholder = "输入服务器名称、描述或标签..."
	search.CharLimit = 100
	search.Width = 50

	m := &Model{
		list:    l,
		servers: cfg.Servers,
		config:  cfg,
		search:  search,
	}

	return m, nil
}

// refreshList 刷新列表
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

		// 搜索过滤
		if searchTerm != "" {
			searchable := strings.ToLower(s.Name + " " + s.Description + " " + strings.Join(s.Tags, " "))
			if !strings.Contains(searchable, searchTerm) {
				continue
			}
		}

		filtered = append(filtered, item{server: s})
	}

	m.list.SetItems(filtered)
	m.list.ResetFilter()
}

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

// Run 运行交互式界面
func Run() error {
	m, err := NewModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("运行界面失败: %w", err)
	}

	// tea程序退出后，检查是否有待连接的服务器
	if finalModel != nil {
		if model, ok := finalModel.(Model); ok && model.pendingServer != nil {
			connectToServer(*model.pendingServer)
		}
	}

	return nil
}
