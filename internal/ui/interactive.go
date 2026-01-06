package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fijdemon/gssh/internal/config"
)

// Model UI模型
type Model struct {
	list          list.Model
	delegate      multiLineDelegate // 保存 delegate 引用以便更新搜索词
	servers       []config.Server
	config        *config.Config
	search        textinput.Model
	searchMode    bool
	preSearchMode bool // 预搜索模式：搜索框高亮但未激活输入
	selectedGroup string
	selectedTags  []string
	width         int
	height        int
	formMode      bool
	form              FormModel
	deleteConfirm     bool
	deleteConfirmInput textinput.Model // 删除确认输入框
	pendingServer     *config.Server   // 待连接的服务器，在退出tea后执行
}

// Init 初始化
func (m Model) Init() tea.Cmd {
	return nil
}

// enterPreSearchMode 进入预搜索模式
func (m *Model) enterPreSearchMode() (tea.Model, tea.Cmd) {
	m.preSearchMode = true
	if len(m.list.Items()) > 0 {
		m.list.Select(-1)
	}
	return *m, nil
}

// enterSearchMode 进入搜索模式
func (m *Model) enterSearchMode() (tea.Model, tea.Cmd) {
	m.preSearchMode = false
	m.searchMode = true
	m.search.Focus()
	if len(m.list.Items()) > 0 {
		m.list.Select(-1)
	}
	m.refreshList()
	return *m, textinput.Blink
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
		} else if m.deleteConfirm {
			// 删除确认模式下，更新输入框宽度
			if msg.Width > 20 {
				m.deleteConfirmInput.Width = msg.Width - 20
			}
		} else {
			m.list.SetWidth(msg.Width)
			// 预留标题、搜索框、上下分割线和底部帮助等固定行数
			// 标题3行 + 搜索框2行 + 列表上方分割线1行 + 列表下方空行1行 + 底部帮助4行 = 11行
			// 由于每个 item 占 4 行（Title 1行 + Description 3行），列表高度需要是 4 的倍数
			availableHeight := max(msg.Height-11, 4)
			// 确保列表高度不超过可用空间
			m.list.SetHeight(availableHeight)
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
			case "esc":
				// Esc 取消删除
				m.deleteConfirm = false
				m.deleteConfirmInput.Blur()
				m.deleteConfirmInput.SetValue("")
				return m, nil
			case "enter":
				// 回车确认删除
				selectedItem := m.list.SelectedItem()
				if item, ok := selectedItem.(item); ok {
					inputName := strings.TrimSpace(m.deleteConfirmInput.Value())
					serverName := item.server.Name
					// 只有输入的名称完全匹配时才执行删除
					if inputName == serverName {
						m.config.DeleteServer(serverName)
						config.Save(m.config)
						m.refreshList()
						m.deleteConfirm = false
						m.deleteConfirmInput.Blur()
						m.deleteConfirmInput.SetValue("")
						// 确保列表选中第一个元素
						if len(m.list.Items()) > 0 {
							m.list.Select(0)
						}
					}
					// 如果不匹配，不清除输入，让用户重新输入
				}
				return m, nil
			default:
				// 处理输入
				var cmd tea.Cmd
				m.deleteConfirmInput, cmd = m.deleteConfirmInput.Update(msg)
				return m, cmd
			}
		}

		// 预搜索模式
		if m.preSearchMode {
			switch msg.String() {
			case "enter", "a", "i":
				// 回车进入搜索模式
				m.preSearchMode = false
				return m.enterSearchMode()
			case "j":
				// 按 j 返回列表第一个，退出预搜索模式
				m.preSearchMode = false
				if len(m.list.Items()) > 0 {
					m.list.Select(0)
				}
				return m, nil
			case "backspace":
				m.search.SetValue("")
				m.refreshList()
				return m, nil
			case "esc":
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			case "/":
				return m.enterSearchMode()
			}
			return m, nil
		}

		// 搜索模式
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.preSearchMode = true
				m.search.Blur()
				return m, nil
			case "enter":
				m.searchMode = false
				m.search.Blur()
				m.preSearchMode = true
				return m, nil
			case "backspace":
				// 如果搜索框为空，退出搜索模式
				if m.search.Value() == "" {
					m.searchMode = false
					m.search.Blur()
					// 退出搜索模式时，确保列表选中第一个元素
					if len(m.list.Items()) > 0 {
						m.list.Select(0)
					}
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
			return m.enterSearchMode()

		case "j":
			m.list.CursorDown()
			return m, nil

		case "k":
			if m.list.Cursor() == 0 && m.list.Paginator.Page == 0 {
				// 如果已经在第一个位置，按 k 进入预搜索模式
				return m.enterPreSearchMode()
			}
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
			if item, ok := selectedItem.(item); ok {
				m.deleteConfirm = true
				// 初始化删除确认输入框
				deleteInput := textinput.New()
				deleteInput.Placeholder = fmt.Sprintf("输入服务器名称 '%s' 以确认删除", item.server.Name)
				deleteInput.CharLimit = 100
				deleteInput.Width = 60
				deleteInput.Focus()
				m.deleteConfirmInput = deleteInput
			}
			return m, textinput.Blink

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
		case "esc":
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
			var b strings.Builder
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render("⚠️  危险操作：删除服务器"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("服务器名称: %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Render(item.server.Name)))
			b.WriteString(fmt.Sprintf("地址: %s\n", item.server.GetAddress()))
			b.WriteString(fmt.Sprintf("用户: %s\n", item.server.User))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("请输入服务器名称以确认删除:"))
			b.WriteString("\n")
			b.WriteString(m.deleteConfirmInput.View())
			b.WriteString("\n\n")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("提示: 输入完整的服务器名称并按 Enter 确认，或按 Esc 取消"))
			b.WriteString("\n")
			return b.String()
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
	} else if m.preSearchMode {
		// 预搜索模式：搜索框高亮显示
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Render(" 筛选: "))
		searchView := m.search.View()
		// 高亮搜索框内容
		highlightedSearch := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(lipgloss.Color("236")).Render(searchView)
		b.WriteString(highlightedSearch)
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Render(" 回车进入搜索 | 退格清空搜索 | j/ESC 返回列表\n"))
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
	b.WriteString(m.list.View())
	b.WriteString("\n")

	// 底部操作提示
	if separatorLen > 0 {
		b.WriteString(strings.Repeat("─", separatorLen))
	}
	b.WriteString("\n")
	help := " 操作: j/k 移动 h/l 翻页 gG跳转 | Enter 登录 | / 搜索 | a 添加 | d 删除 | e 编辑 | q 退出"
	b.WriteString(help)
	b.WriteString("\n")
	if separatorLen > 0 {
		b.WriteString(strings.Repeat("─", separatorLen))
	}
	b.WriteString("\n")

	return b.String()
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

	// 创建列表，使用支持多行的自定义 delegate
	delegate := multiLineDelegate{
		DefaultDelegate: list.NewDefaultDelegate(),
		searchTerm:      "", // 初始搜索词为空
	}
	// 设置选中样式（使用lipgloss颜色）
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.Color("212"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	// 隐藏标题样式，避免显示蓝色方块（设置为完全透明）
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("")).Background(lipgloss.Color("")).Width(0).Height(0)
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle

	// 创建搜索输入框
	search := textinput.New()
	search.Placeholder = "输入名称、描述、标签、IP 或用户名..."
	search.CharLimit = 100
	search.Width = 50

	m := &Model{
		list:     l,
		delegate: delegate, // 保存 delegate 引用
		servers:  cfg.Servers,
		config:   cfg,
		search:   search,
	}

	return m, nil
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
