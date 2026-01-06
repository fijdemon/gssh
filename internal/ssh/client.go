package ssh

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// Client SSH客户端封装
type Client struct {
	config *ssh.ClientConfig
}

// Connect 连接到服务器并执行命令（交互式，全部通过 expect 实现）
func Connect(hostname string, user string, port int, authConfig AuthConfig) error {
	fmt.Println("认证类型: ", authConfig.Type)
	switch authConfig.Type {
	case "key":
		// 纯 key 模式：只加 -i，不自动填充密码；如果失败（Permission denied）直接退出。
		fmt.Println("使用密钥文件（key 模式，失败后用户手动输入密码）...")
		return connectWithKeyExpect(hostname, user, port, authConfig.IdentityFile)
	case "auto":
		// auto 模式：加 -i，若密钥认证失败、出现密码提示则自动填充密码。
		fmt.Println("使用密钥 + 密码自动回退（auto 模式）...")
		return connectWithAutoExpect(hostname, user, port, authConfig.IdentityFile, authConfig.Password)
	case "password":
		// password 模式：不加 -i，只用密码登录。
		fmt.Println("使用密码登录（password 模式）...")
		return connectWithPassword(hostname, user, port, authConfig.Password)
	default:
		// 兜底逻辑：尽量不惊动老配置
		if authConfig.IdentityFile != "" {
			fmt.Println("未知认证类型，按 auto 处理（key + password）...")
			return connectWithAutoExpect(hostname, user, port, authConfig.IdentityFile, authConfig.Password)
		}
		fmt.Println("未知认证类型，按 password 处理（仅密码）...")
		return connectWithPassword(hostname, user, port, authConfig.Password)
	}
}

// escapeExpectString 转义expect脚本中的特殊字符
func escapeExpectString(s string) string {
	// 转义expect中的特殊字符: [, ], {, }, $, ", \, 等
	// 注意：使用 send -- 后，大部分特殊字符不需要转义，但为了安全起见还是转义
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// connectWithPassword 使用密码连接（通过expect）
func connectWithPassword(hostname string, user string, port int, password string) error {
	// password 模式：不加 -i，只使用密码自动登录
	// 构建SSH命令
	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, hostname),
	}

	// 转义密码中的特殊字符（使用专门的函数）
	escapedPassword := escapeExpectString(password)
	// 转义SSH命令参数中的特殊字符
	escapedSSHArgs := escapeExpectString(strings.Join(sshArgs, " "))

	// 使用 expect 自动输入密码
	expectScript := fmt.Sprintf(`
set timeout 30

set logged_in 0

spawn ssh %s

expect {
		-re "(?i)(yes/no)" {
			sleep 0.1
			send "yes\r"
			exp_continue
		}
		-re "(?i)(permission denied|failed password)" {
			puts stderr "Authentication failed"
			exit 1
		}
		-re "(?i)(password|Password):" {
			sleep 0.1
			send -- "%s\r"
			exp_continue
		}
		-re {.*[\$#] } {
			# 匹配可能的 shell 提示符
			set logged_in 1
		}
		eof {
			if {$logged_in == 0} {
				# 连接在未登录成功前就结束，视为失败
				exit 1
			}
		}
}

if {$logged_in == 1} {
	# 只有确认已经进入 shell 后才进入交互
	interact
}

exit
`, escapedSSHArgs, escapedPassword)

	cmd := exec.Command("expect", "-c", expectScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// connectWithKeyExpect 仅使用密钥（type=key），不自动填充密码
func connectWithKeyExpect(hostname string, user string, port int, identityFile string) error {
	// 展开密钥路径
	keyPath := identityFile
	if keyPath != "" && keyPath[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		keyPath = filepath.Join(homeDir, keyPath[1:])
	}

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
	if keyPath != "" {
		sshArgs = append(sshArgs, "-i", keyPath)
	}
	sshArgs = append(sshArgs,
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, hostname),
	)

	escapedSSHArgs := escapeExpectString(strings.Join(sshArgs, " "))

	// key 模式：不判断是否有 password，不自动填充。
	// 如果出现密码提示，直接把控制权交给用户；如果出现 Permission denied 等错误则直接退出。
	expectScript := fmt.Sprintf(`
set timeout 30

spawn ssh %s

expect {
	-re "(?i)(yes/no)" {
		sleep 0.1
		send "yes\r"
		exp_continue
	}
	-re "(?i)(permission denied|failed password)" {
		puts stderr "Authentication failed"
		exit 1
	}
	-re "(?i)(password|Password):" {
		# 出现密码提示时，不自动输入密码，直接交给用户
		interact
		exit
	}
	-re {.*[\$#] } {
		# 已经进入 shell，交互
		interact
		exit
	}
	eof {
		# 连接结束且未成功登录
		exit 1
	}
}

exit
`, escapedSSHArgs)

	cmd := exec.Command("expect", "-c", expectScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// connectWithAutoExpect 使用密钥，失败时自动用密码填充（type=auto）
func connectWithAutoExpect(hostname string, user string, port int, identityFile, password string) error {
	// 展开密钥路径
	keyPath := identityFile
	if keyPath != "" && keyPath[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		keyPath = filepath.Join(homeDir, keyPath[1:])
	}

	sshArgs := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}
	if keyPath != "" {
		sshArgs = append(sshArgs, "-i", keyPath)
	}
	sshArgs = append(sshArgs,
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, hostname),
	)

	escapedSSHArgs := escapeExpectString(strings.Join(sshArgs, " "))
	escapedPassword := escapeExpectString(password)

	expectScript := fmt.Sprintf(`
set timeout 30

set logged_in 0

spawn ssh %s

expect {
	-re "(?i)(yes/no)" {
		sleep 0.1
		send "yes\r"
		exp_continue
	}
	-re "(?i)(permission denied|failed password)" {
		puts stderr "Authentication failed"
		exit 1
	}
	-re "(?i)(password|Password):" {
		# 密钥失败后，自动用密码登录
		sleep 0.1
		send -- "%s\r"
		exp_continue
	}
	-re {.*[\$#] } {
		set logged_in 1
	}
	eof {
		if {$logged_in == 0} {
			exit 1
		}
	}
}

if {$logged_in == 1} {
	interact
}

exit
`, escapedSSHArgs, escapedPassword)

	cmd := exec.Command("expect", "-c", expectScript)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// AuthConfig 认证配置（从config包导入的类型）
type AuthConfig struct {
	Type         string
	Password     string
	IdentityFile string
}

// NewSSHClient 创建SSH客户端（用于程序化操作，非交互式登录）
func NewSSHClient(hostname string, user string, port int, authConfig AuthConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod
	var keyErrors []string

	// 首先尝试使用 ssh-agent（如果可用）
	if agentAuth := getAgentAuth(); agentAuth != nil {
		authMethods = append(authMethods, agentAuth)
	}

	// 尝试密钥认证
	if authConfig.IdentityFile != "" {
		keyPath := authConfig.IdentityFile
		if keyPath[0] == '~' {
			homeDir, _ := os.UserHomeDir()
			keyPath = filepath.Join(homeDir, keyPath[1:])
		}

		key, err := os.ReadFile(keyPath)
		if err != nil {
			keyErrors = append(keyErrors, fmt.Sprintf("无法读取密钥文件 %s: %v", keyPath, err))
		} else {
			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				// 检查是否是密码保护的密钥
				if strings.Contains(err.Error(), "passphrase") {
					// 如果 ssh-agent 已经可用，就不需要输入 passphrase
					// ssh-agent 中的密钥会优先使用
					if len(authMethods) == 0 {
						// ssh-agent 不可用，提示用户输入 passphrase
						passphrase, err := promptPassphrase(keyPath)
						if err != nil {
							keyErrors = append(keyErrors, fmt.Sprintf("无法获取密钥密码: %v", err))
						} else {
							// 使用 passphrase 解析密钥
							signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
							if err != nil {
								keyErrors = append(keyErrors, fmt.Sprintf("无法解析密钥文件（密码错误）: %v", err))
							} else {
								authMethods = append(authMethods, ssh.PublicKeys(signer))
							}
						}
					}
					// 如果 ssh-agent 可用，跳过文件解析，直接使用 agent 中的密钥
				} else {
					keyErrors = append(keyErrors, fmt.Sprintf("无法解析密钥文件 %s: %v", keyPath, err))
				}
			} else {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	// 添加密码认证
	if authConfig.Password != "" {
		authMethods = append(authMethods, ssh.Password(authConfig.Password))
	}

	if len(authMethods) == 0 {
		var errMsg strings.Builder
		errMsg.WriteString("没有可用的认证方法。")
		if len(keyErrors) > 0 {
			errMsg.WriteString("\n密钥认证失败：\n")
			errMsg.WriteString("  - ")
			errMsg.WriteString(keyErrors[0])
			errMsg.WriteString("\n")
		}
		if authConfig.IdentityFile == "" && authConfig.Password == "" {
			errMsg.WriteString("\n请配置 SSH 密钥路径或密码。")
		} else if authConfig.Password == "" {
			errMsg.WriteString("\n密钥认证失败且未配置密码，请检查密钥文件或配置密码。")
		}
		return nil, fmt.Errorf("%s", errMsg.String())
	}

	// 设置known_hosts
	homeDir, _ := os.UserHomeDir()
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	hostKeyCallback, _ := knownhosts.New(knownHostsPath)

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}

	return client, nil
}

// ExecuteCommand 在远程服务器执行命令
func ExecuteCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// getAgentAuth 尝试从 ssh-agent 获取认证方法
func getAgentAuth() ssh.AuthMethod {
	// 检查 SSH_AUTH_SOCK 环境变量
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}

	// 返回一个回调函数，在需要时才连接 agent
	// 这样可以避免连接过早关闭的问题
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		// 连接到 ssh-agent
		conn, err := net.Dial("unix", socket)
		if err != nil {
			return nil, fmt.Errorf("连接 ssh-agent 失败: %w", err)
		}
		// 注意：这里不能关闭连接，因为 agent 客户端需要使用它
		// 连接会在 SSH 客户端关闭时自动关闭

		// 创建 agent 客户端
		agentClient := agent.NewClient(conn)

		// 获取 signers
		return agentClient.Signers()
	})
}

// promptPassphrase 提示用户输入密钥密码
func promptPassphrase(keyPath string) (string, error) {
	fmt.Printf("密钥文件需要密码保护: %s\n", keyPath)
	fmt.Print("请输入密钥密码: ")

	// 使用 term 包隐藏密码输入
	passphrase, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("读取密码失败: %w", err)
	}
	fmt.Println() // 换行

	return string(passphrase), nil
}

// CopyFile 通过SSH复制文件（已弃用，使用scp命令代替）
// 保留此方法以保持API兼容性，但实际实现已改用系统scp命令
func CopyFile(client *ssh.Client, localPath string, remotePath string) error {
	// 这个方法不再使用，sync包已改用系统scp命令
	return fmt.Errorf("此方法已弃用，请使用系统scp命令")
}
