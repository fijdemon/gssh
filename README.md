# gssh

Go 版本 SSH 服务器管理工具 - 快速登录和管理多台服务器

## 功能特性

- 🚀 **快速登录** - 通过交互式界面或命令行快速连接服务器
- 📝 **配置管理** - 统一管理多台服务器的连接信息
- 🔐 **多种认证** - 支持密钥登录和密码登录
- 🏷️ **标签分组** - 支持服务器标签和分组管理
- ☁️ **云端同步** - 通过 SSH 方式同步配置到云端（支持多终端）
- 🎨 **友好界面** - 使用 vim 风格的交互界面（j/k 移动，回车确认）

## 快速上手

```bash
# 1. 安装（需要本机已安装 Go 1.24+）
go install github.com/fijdemon/gssh@latest

# 2. 初始化配置文件（创建 ~/.gssh/config.yaml）
gssh init

# 3. 编辑配置文件，添加你的服务器
vim ~/.gssh/config.yaml

# 4. 打开交互式界面，从列表中选择并登录
gssh

# 或者直接通过名称登录
gssh <server-name>
```

## 安装

### 使用 go install

```bash
go install github.com/fijdemon/gssh@latest
```

### 系统依赖

`gssh` 本身是一个单一可执行文件，但运行时依赖以下系统命令（类 Unix / macOS 环境）：

- `ssh` / `scp`：用于实际建立 SSH 连接以及推送同步配置
- `expect`：用于在密码登录时自动输入密码（例如服务器登录时选择 `password` / `auto`）

> 如果只使用密钥认证，可以不安装 `expect`，但所有基于密码的自动登录将不可用，届时会退回到由 `ssh` 自己提示输入密码。

## 使用方法

### 命令一览

- `gssh`：打开交互式界面
- `gssh init`：初始化配置文件
- `gssh <server-name>`：按名称直接登录指定服务器
- `gssh pull`：从云端拉取配置（只更新 `servers`）
- `gssh push`：将本地服务器列表推送到云端
- `gssh version`：显示版本信息
- `gssh help`：显示帮助信息

### 初始化配置

首次使用需要初始化配置文件：

```bash
gssh init
```

这个命令会：
- 创建默认配置文件 `~/.gssh/config.yaml`
- 可选设置云端同步参数
- 引导你完成基本配置

### 交互式界面

直接运行 `gssh` 打开交互式界面：

```bash
gssh
```

**键位说明：**

- `j / k`：上下移动选择服务器
- `Enter`：登录当前选中服务器
- `/`：进入搜索模式（按名称 / 描述 / 标签过滤）
- `a`：添加服务器（表单方式）
- `e`：编辑当前选中服务器
- `d`：删除当前选中服务器（有二次确认）
- `q`：退出程序
- `Ctrl+C`：强制退出

**搜索模式：**

- 输入关键字即时过滤列表
- `Enter`：确认搜索并返回列表
- `Backspace` 且输入为空：退出搜索模式
- `Esc`：退出搜索模式

### 直接登录

通过服务器名称直接登录：

```bash
gssh <server-name>
```

### 配置同步

从云端拉取配置：

```bash
gssh pull
```

推送配置到云端：

```bash
gssh push
```

> **重要说明：**
> - 同步时**只同步 `servers` 部分**，`sync` 配置由各客户端自己维护
> - 每个客户端的同步服务器地址、认证方式等可以不同
> - 使用同步功能前，需要先通过 `gssh init` 设置同步参数，或手动编辑配置文件

## 配置文件

配置文件位于 `~/.gssh/config.yaml`

### 配置示例

```yaml
version: "1.0"
sync:
  enabled: true
  type: ssh
  ssh_host: your-sync-server.com
  ssh_user: your-username
  ssh_path: ~/.gssh/config.yaml
  ssh_key: ~/.ssh/id_rsa
  auto_sync: false

servers:
  - name: prod-web
    hostname: 192.168.1.100
    user: root
    port: 22
    description: 生产环境Web服务器
    tags: [production, web, nginx]
    group: production
    auth:
      type: auto  # auto|password|key
      password: your-password
      identity_file: ~/.ssh/id_rsa
    created_at: "2024-01-01T00:00:00Z"
```

### 认证类型说明

- `auto` - 根据配置自动选择合适方式：
  - 若配置了 `identity_file` 则使用密钥登录；
  - 否则若配置了 `password` 则使用密码登录；
  - 否则退回到由系统 `ssh` 自己处理（例如提示输入密码）。
- `key` - 仅使用密钥登录（忽略密码）
- `password` - 仅使用密码登录（忽略密钥）

## 云端同步设置

### SSH 方式同步

1. 确保可以 SSH 连接到同步服务器
2. 在配置文件中设置同步参数：
   - `ssh_host`: 同步服务器地址
   - `ssh_user`: SSH 用户名
   - `ssh_path`: 远程配置文件路径
   - `ssh_key`: SSH 密钥路径（可选）
   - `password`: SSH 密码（可选，与密钥二选一，用于程序化读取远程配置）

### 同步机制说明

- **只同步服务器列表**：`pull` 和 `push` 操作只同步 `servers` 部分
- **sync 配置独立**：每个客户端的 `sync` 配置（服务器地址、认证方式等）由本地维护，不会被同步覆盖
- **多终端支持**：不同终端可以配置不同的同步服务器，但共享相同的服务器列表

### 使用示例

```bash
# 拉取配置（只更新 servers 列表，保留本地 sync 配置）
gssh pull

# 推送配置（只推送 servers 列表，不推送 sync 配置）
gssh push
```

> 推送 (`gssh push`) 时：
> - 如果配置了 `ssh_key`，会使用 `scp -i` 无感推送；
> - 如果没有配置 `ssh_key`，则直接调用 `scp`，必要时由 `scp` 自己提示你输入密码；
> - 不再依赖 `sshpass`。

## 安全注意事项

- **明文密码存储**
  - `servers[*].auth.password` 和 `sync.password` 会以**明文**形式写入 `~/.gssh/config.yaml`。
  - 建议：
    - 尽量使用 SSH 密钥认证（配置 `identity_file` / `ssh_key`）；
    - 保证配置文件权限严格（推荐 `chmod 600 ~/.gssh/config.yaml`）；
    - 不要将配置文件提交到任何版本库。

- **主机密钥校验关闭**
  - 为了降低首次使用门槛，内部调用 `ssh` / `scp` 时统一使用：
    - `-o StrictHostKeyChecking=no`
    - `-o UserKnownHostsFile=/dev/null`
  - 这意味着不会校验远端主机指纹，在不可信网络环境中存在被中间人攻击（MITM）的风险。
  - 如果你有更高的安全要求，可以自行 fork 并移除这些选项。

- **本地环境假设**
  - `gssh` 设计为在**你控制的本机**上使用，不考虑多用户共享或不可信宿主环境。
  - 请确保本机账号本身是可信的，且终端历史不会泄露敏感信息。

## 依赖

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 解析
- [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) - SSH 客户端

## 开源协议

本项目使用 **MIT License**，详情见 `LICENSE` 文件。
