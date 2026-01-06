package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fijdemon/gssh/internal/config"
)

// FormModel 表单模型
type FormModel struct {
	inputs        []textinput.Model
	currentIndex  int // 当前正在输入的字段索引
	width         int
	height        int
	isEdit        bool
	editingServer *config.Server
	onSave        func(config.Server) error
	onCancel      func()
	quitting      bool     // 标记是否正在退出
	fieldLabels   []string // 字段标签
}

// Init 初始化表单
func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update 更新表单
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.onCancel != nil {
				m.onCancel()
			}
			m.quitting = true
			return m, nil

		case "backspace":
			// 如果当前输入框为空，返回上一项
			if m.inputs[m.currentIndex].Value() == "" && m.currentIndex > 0 {
				m.currentIndex--
				m.inputs[m.currentIndex].Focus()
				return m, textinput.Blink
			}

		case "enter":
			// 验证当前字段（如果是必填字段）
			if m.isRequiredField(m.currentIndex) && m.inputs[m.currentIndex].Value() == "" {
				// 必填字段为空，不继续
				return m, nil
			}

			// 如果是最后一个字段，保存
			if m.currentIndex == len(m.inputs)-1 {
				return m.saveServer()
			}

			// 移动到下一个字段
			m.currentIndex++
			m.inputs[m.currentIndex].Focus()
			return m, textinput.Blink
		}
	}

	// 只更新当前输入框
	var cmd tea.Cmd
	m.inputs[m.currentIndex], cmd = m.inputs[m.currentIndex].Update(msg)
	return m, cmd
}

// isRequiredField 检查字段是否为必填
func (m FormModel) isRequiredField(index int) bool {
	// 名称、主机地址、用户名为必填字段
	return index == 0 || index == 1 || index == 2
}

// saveServer 保存服务器
func (m FormModel) saveServer() (tea.Model, tea.Cmd) {
	// 验证必填字段
	if m.inputs[0].Value() == "" {
		return m, nil // 名称不能为空
	}
	if m.inputs[1].Value() == "" {
		return m, nil // 主机地址不能为空
	}
	if m.inputs[2].Value() == "" {
		return m, nil // 用户名不能为空
	}

	// 解析端口
	port := 22
	if m.inputs[3].Value() != "" {
		if p, err := strconv.Atoi(m.inputs[3].Value()); err == nil {
			port = p
		}
	}

	// 解析标签
	tags := []string{}
	if m.inputs[6].Value() != "" {
		tags = strings.Split(m.inputs[6].Value(), ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	// 创建服务器配置
	server := config.Server{
		Name:        strings.TrimSpace(m.inputs[0].Value()),
		Hostname:    strings.TrimSpace(m.inputs[1].Value()),
		User:        strings.TrimSpace(m.inputs[2].Value()),
		Port:        port,
		Description: strings.TrimSpace(m.inputs[4].Value()),
		Group:       strings.TrimSpace(m.inputs[5].Value()),
		Tags:        tags,
		Auth: config.AuthConfig{
			Type:         strings.TrimSpace(m.inputs[7].Value()),
			Password:     strings.TrimSpace(m.inputs[8].Value()),
			IdentityFile: strings.TrimSpace(m.inputs[9].Value()),
		},
	}

	// 设置默认值
	if server.Auth.Type == "" {
		server.Auth.Type = "auto"
	}
	if server.Auth.IdentityFile == "" {
		server.Auth.IdentityFile = "~/.ssh/id_rsa"
	}

	if m.isEdit && m.editingServer != nil {
		// 编辑模式：保留创建时间
		server.CreatedAt = m.editingServer.CreatedAt
		server.LastUsed = m.editingServer.LastUsed
	} else {
		// 新建模式：设置创建时间
		server.CreatedAt = time.Now().Format(time.RFC3339)
	}

	// 调用保存回调
	if m.onSave != nil {
		if err := m.onSave(server); err != nil {
			return m, nil
		}
	}

	// 保存成功，退出表单
	m.quitting = true
	return m, nil
}

// View 渲染表单
func (m FormModel) View() string {
	// 即使 width 为 0 也显示表单，使用默认宽度
	if m.width == 0 {
		m.width = 80
	}

	var b strings.Builder
	title := "添加服务器"
	if m.isEdit {
		title = "编辑服务器"
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Render(title))
	b.WriteString("\n\n")

	// 显示已完成的字段（灰色）
	for i := 0; i < m.currentIndex; i++ {
		label := m.fieldLabels[i]
		value := m.inputs[i].Value()
		if value == "" {
			value = "(未填写)"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("✓ %s: %s", label, value)))
		b.WriteString("\n")
	}

	// 显示当前正在输入的字段（高亮）
	if m.currentIndex < len(m.inputs) {
		label := m.fieldLabels[m.currentIndex]
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render(fmt.Sprintf("> %s:", label)))
		b.WriteString("\n")
		b.WriteString(m.inputs[m.currentIndex].View())
		b.WriteString("\n")
	}

	// 显示提示信息
	b.WriteString("\n")
	if m.currentIndex < len(m.inputs)-1 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("按 Enter 继续下一个字段 | 输入框为空时按 Backspace 返回上一项 | Esc 取消"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("按 Enter 保存 | 输入框为空时按 Backspace 返回上一项 | Esc 取消"))
	}

	b.WriteString("\n")
	return b.String()
}

// NewFormModel 创建表单模型
func NewFormModel(editingServer *config.Server, onSave func(config.Server) error, onCancel func()) FormModel {
	m := FormModel{
		inputs:        make([]textinput.Model, 10),
		currentIndex:  0,
		isEdit:        editingServer != nil,
		editingServer: editingServer,
		onSave:        onSave,
		onCancel:      onCancel,
		fieldLabels: []string{
			"名称 *",
			"主机地址 *",
			"用户名 *",
			"端口",
			"描述",
			"分组",
			"标签（逗号分隔）",
			"认证类型 (auto/key/password)",
			"密码",
			"密钥路径",
		},
	}

	// 初始化输入框
	inputs := []textinput.Model{
		textinput.New(), // 名称
		textinput.New(), // 主机地址
		textinput.New(), // 用户名
		textinput.New(), // 端口
		textinput.New(), // 描述
		textinput.New(), // 分组
		textinput.New(), // 标签
		textinput.New(), // 认证类型
		textinput.New(), // 密码
		textinput.New(), // 密钥路径
	}

	// 设置输入框属性
	inputs[0].Placeholder = "例如: prod-web"
	inputs[0].Focus()
	inputs[0].CharLimit = 50

	inputs[1].Placeholder = "例如: 192.168.1.100"
	inputs[1].CharLimit = 100

	inputs[2].Placeholder = "例如: root"
	inputs[2].CharLimit = 50

	inputs[3].Placeholder = "22"
	inputs[3].CharLimit = 5

	inputs[4].Placeholder = "例如: 生产环境Web服务器"
	inputs[4].CharLimit = 100

	inputs[5].Placeholder = "例如: production"
	inputs[5].CharLimit = 50

	inputs[6].Placeholder = "例如: web,nginx,production"
	inputs[6].CharLimit = 200

	inputs[7].Placeholder = "auto"
	inputs[7].CharLimit = 20

	inputs[8].Placeholder = "留空则不存储密码"
	inputs[8].EchoMode = textinput.EchoPassword
	inputs[8].EchoCharacter = '*'

	inputs[9].Placeholder = "~/.ssh/id_rsa"
	inputs[9].CharLimit = 200

	// 如果是编辑模式，填充现有值
	if editingServer != nil {
		inputs[0].SetValue(editingServer.Name)
		inputs[1].SetValue(editingServer.Hostname)
		inputs[2].SetValue(editingServer.User)
		inputs[3].SetValue(fmt.Sprintf("%d", editingServer.Port))
		inputs[4].SetValue(editingServer.Description)
		inputs[5].SetValue(editingServer.Group)
		inputs[6].SetValue(strings.Join(editingServer.Tags, ","))
		inputs[7].SetValue(editingServer.Auth.Type)
		inputs[8].SetValue(editingServer.Auth.Password)
		inputs[9].SetValue(editingServer.Auth.IdentityFile)
	}

	// 设置样式
	for i := range inputs {
		inputs[i].Width = 50
		inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		inputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		// 只有第一个输入框获得焦点
		if i == 0 {
			inputs[i].Focus()
		} else {
			inputs[i].Blur()
		}
	}

	m.inputs = inputs
	return m
}
