# Skill - Agent Skill Management Tool

一个功能强大的 AI Agent 技能管理命令行工具，用于搜索、安装、管理和部署技能包。

## 简介

Skill 是一个用 Go 语言开发的命令行工具，帮助 AI Agent 管理技能包。类似于 Git 管理代码仓库，Skill 管理技能包的生命周期。支持从 Git 仓库、本地路径或已安装技能中添加技能到项目。

**主要功能：**

- ✅ 从 Git 仓库安装技能（支持整个仓库或单个技能子目录）
- ✅ 列出本地、项目或远程技能
- ✅ 添加技能到项目
- ✅ 搜索技能（支持本地、项目目录）
- ✅ 移除技能（支持本地和项目）
- ✅ 显示技能详细信息（包括安装时间）
- ✅ 支持多种技能目录结构

## 可用命令

- `repo` - 管理远程技能仓库（添加、列表、删除、设置默认、索引）
- `add` - 从本地路径或已安装技能名称添加技能到项目
- `install` - 从 Git 仓库安装技能到本地（支持整个仓库或单个子目录）
- `update` - 从远程仓库更新已安装的技能
- `list` - 列出已安装的技能、项目中的技能或远程技能
- `push` - 上传技能到远程仓库
- `remove` - 从本地或项目目录移除技能
- `search` - 在本地目录、项目目录或已安装技能中搜索技能
- `status` - 列出所有已安装的技能

**命令参数：**

- `-l, --local` - 更新本地目录中的技能
- `-p, --project` - 列出项目目录中的技能或更新项目中的技能
- `-r, --repo string` - 指定仓库名称（用于 install 命令从指定仓库安装技能，用于 update 命令从指定仓库更新技能，或用于 list 命令从数据库列出指定仓库的技能）
- `-s, --skills-dir string` - 扫描指定目录中的技能

## 安装

### 从源码构建

**Windows:**

```cmd
git clone https://code.cnworkshop.xyz/sought/ai-agent-skill.git
cd .\ai-agent-skill
go build -o skillctl.exe .
```

**Linux/macOS:**
系统默认有“skill”命令，如果在创建skill会冲突，需要换一下名称。

```bash
git clone https://code.cnworkshop.xyz/sought/ai-agent-skill.git
cd ai-agent-skill
go build -o skillctl .
```

### 全局安装（推荐）

```bash
go install
```

这将把 `skill.exe` 安装到 `~/go/bin/` 目录（已在 PATH 中），可以在任何地方使用。
Linux一般不会把go/bin加到path中，如果没有需要手动添加

```shell
#编辑配置文件
vim ~/.bashrc
#在文件最后添加

# Go environment
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
#让配置生效
source ~/.bashrc

#检查
go env GOPATH
ls /root/go/bin
```

## 使用方式

### 0. 远程仓库管理

#### 添加远程仓库

```bash

# 添加仓库（自动生成名称）
skillctl repo add https://github.com/user/repo

# 添加仓库并指定名称
skillctl repo add https://github.com/user/repo -n my-repo

# 添加仓库并设置为默认
skillctl repo add https://github.com/user/repo -n my-repo --default

# 添加仓库并设置优先级
skillctl repo add https://github.com/user/repo -p 10
```

#### 列出所有仓库

```bash
skillctl repo list
```

显示所有已配置的仓库及其详细信息，包括：

- 仓库名称
- 仓库 URL
- 描述
- 是否为默认仓库
- 优先级
- 创建时间
- 最后同步时间
- 同步状态

#### 设置默认仓库

```bash
skillctl repo default <仓库名称>
```

例如：

```bash
skillctl repo default my-repo
```

#### 删除仓库

```bash
skillctl repo remove <仓库名称>
```

**注意**: 删除仓库会同时删除该仓库的所有技能索引。

#### 索引/同步仓库

```bash
#如果index导入报错128

# 同步默认仓库
skillctl repo index

# 同步指定仓库
skillctl repo index <仓库名称>
```

这会从远程仓库获取技能信息并存储到本地数据库。支持：

- Git 仓库 URL（使用 git clone）
- REST API URL（返回 JSON 格式的技能列表）

**数据库位置**:

- Windows: `%APPDATA%\skill\skill.db`
- Linux/macOS: `~/.cache/skill/skill.db`

### 1. 列出技能

#### 列出已安装的技能

```bash
skillctl list
```

显示 `~/.skills` 目录中已安装的所有技能，包括：

- 技能名称
- 描述
- 安装时间（ISO 8601 格式）
- 路径
- 标签

**示例输出：**

```
1. pdf
   Description: Use this skill whenever the user wants to do anything with PDF files.
   Path: C:\Users\user\.skills\pdf
   Installed: 2026-03-09T10:30:00Z
```

#### 列出远程技能

**从数据库列出指定仓库的技能：**

```bash
# 列出指定仓库中的所有技能
skillctl list --repo <仓库名称>

# 示例
skillctl list -r my-repo
```

从已索引的数据库中列出指定仓库的所有技能。

**列出 Git 仓库中的技能（直接从 URL）：**

```bash
# 列出 GitHub 仓库中 skills 目录下的所有技能
skillctl list -R https://github.com/user/my-skill
skillctl list -R https://github.com/github/awesome-copilot
```

#### 列出项目中的技能

```bash
skillctl list --project
```

扫描当前项目目录下的所有技能目录（`.github/skills`、`.agents/skills`、`.claude/skills`、`.copilot/skills`、`examples/skills`）。

#### 列出指定目录的技能

```bash
skillctl list --skills-dir examples/skills
skillctl list --skills-dir ~/.skill
```

扫描指定目录下的技能。

### 2. 安装技能

#### 从 Git URL 安装整个仓库

```bash
skillctl install https://github.com/user/my-skill
```

克隆 Git 仓库并安装到 `~/.skills` 目录。

#### 从 Git URL 安装单个技能（子目录）

```bash
# 只安装仓库中的特定技能
skillctl install https://github.com/user/my-skill pdf
skillctl install https://github.com/user/my-skill xlsx
```

使用 Git sparse-checkout 功能，只克隆仓库中 `skills/<skill-name>` 子目录，节省带宽和空间。

**支持的仓库结构：**

```
my-skill/
└── skills/
    ├── pdf/
    ├── xlsx/
    └── docx/
```

#### 从数据库安装技能

```bash
# 从默认仓库安装技能
skillctl install <skill-name>

# 从指定仓库安装技能
skillctl install -r <repo-name> <skill-name>

# 示例
skillctl install pdf
skillctl install -r my-repo unit-test-generator
```

从已索引的数据库中安装技能。需要先使用 `skillctl repo index` 同步仓库。

**安装流程：**

1. 检查技能是否已安装
2. 在数据库中搜索技能（支持精确匹配和模糊匹配）
3. 如果指定了 `-r` 参数，只在指定仓库中搜索
4. 使用 sparse-checkout 只下载目标技能文件夹（不包含整个项目）
5. 解析技能元数据
6. 添加到本地索引

**注意：**

- 从数据库安装时，只会下载 `skills/<skill-name>` 目录的内容，不会包含整个项目的其他文件
- 安装后的目录结构：`~/.skills/<skill-name>/` （直接包含技能文件，不包含 `skills/` 父目录）
- 如果技能名称在数据库中不存在，会提供相似的建议

### 3. 更新技能

#### 更新本地技能

```bash
# 更新本地已安装的技能（默认）
skillctl update <skill-name>

# 明确指定更新本地技能
skillctl update <skill-name> -l
```

从远程仓库下载最新版本的技能并与当前版本对比，如果有新版本则自动更新。

#### 更新项目技能

```bash
# 更新项目中的技能
skillctl update <skill-name> -p
```

更新项目目录中的技能（`.agents/skills`、`.github/skills` 等）。

#### 从指定仓库更新技能

```bash
# 从指定仓库更新技能
skillctl update <skill-name> -r <repo-name>

# 示例
skillctl update pdf -r my-repo
skillctl update unit-test-generator -r ningskills
```

从指定仓库检查并更新技能。

**更新流程：**

1. 在数据库中搜索技能（支持指定仓库）
2. 显示当前版本信息
3. 从远程仓库下载最新版本（使用 sparse-checkout 只下载目标技能）
4. 对比版本号
5. 如果版本相同，跳过更新
6. 如果有新版本，删除旧版本并安装新版本
7. 更新元数据（版本、作者、描述等）
8. 显示更新结果

**输出示例：**

```
🔍 Searching for skill 'pdf' in repository...
📦 Found skill in database: pdf
   📝 Use this skill whenever the user wants to do anything with PDF files.
   📌 Current version: 1.0.0
   🔗 https://github.com/user/repo.git

📍 Target location: C:\Users\user\.skills\pdf

📥 Downloading latest version from: https://github.com/user/repo.git
✓ Latest version downloaded successfully

📋 New skill metadata parsed
   New version: 1.1.0

📤 Updating from version 1.0.0 to 1.1.0

✓ New version installed
✓ Skill metadata updated

✅ Skill 'pdf' has been successfully updated!
   Version: 1.1.0
   Author: Test Author
   Description: Use this skill whenever the user wants to do anything with PDF files.
   Location: C:\Users\user\.skills\pdf
```

### 4. 添加技能到项目

#### 从已安装技能添加

```bash
# 先查看已安装的技能
skillctl list

# 添加技能到项目
skillctl add xlsx -p agents
skillctl add pdf -p claude
skillctl add docx -p github
```

从 `~/.skills`、项目目录或 `~/.skill` 目录中查找技能并复制到指定位置。

#### 从本地路径添加

```bash
# 从相对路径添加
skillctl add ./local-skill -p agents

# 从绝对路径添加
skillctl add C:\path\to\skill -p claude
```

#### 从 Git URL 添加到项目

```bash
# 从 Git URL 直接添加到项目
skillctl add https://github.com/user/my-skill -p agents
```

#### 支持的目标位置

- `github` → `.github/skills`
- `agents` → `.agents/skills`
- `claude` → `.claude/skills`
- `copilot` → `.copilot/skills`
- 自定义路径

#### 添加到项目目录

```bash
# 添加到当前项目的 skills 目录
skillctl add my-skill --project

# 添加到指定项目的 skills 目录
skillctl add my-skill --project --dir /path/to/project
```

### 5. 技能仓库管理

#### 创建全局技能仓库

```bash
# 在用户主目录创建 .skill 文件夹
mkdir ~/.skill

# 将常用技能复制到全局仓库
cp -r examples/skills/xlsx ~/.skill/xlsx
cp -r examples/skills/pdf ~/.skill/pdf
```

#### 使用全局技能仓库

```bash
# 在任何项目中使用全局技能（默认从 ~/.skill 查找）
cd /path/to/any-project
skillctl add xlsx -p agents
skillctl add pdf -p claude
```

#### 查看全局技能仓库

```bash
skillctl list --skills-dir ~/.skill
```

#### 搜索技能

**搜索本地已安装的技能（~/.skills）：**

```bash
skillctl search --local pdf
skillctl search -l document
```

在 `~/.skills` 目录中搜索技能，包括索引中的已安装技能。

**搜索项目中的技能：**

```bash
skillctl search --project pdf
skillctl search -p document
```

在项目目录下的技能目录中搜索（`.agents/skills`、`.github/skills`、`.claude/skills`、`.copilot/skills`、`examples/skills`）。

在当前目录的 `./skills` 文件夹中搜索技能。

**搜索范围：**

- 名称
- 描述
- 标签

### 6. 推送技能

**推送到本地已安装技能目录（默认）：**

```bash
skillctl push pdf
```

从项目目录（`.github/skills`、`.agents/skills`、`.copilot/skills`、`examples/skills`）查找技能，并推送到 `~/.skills` 目录，同时更新索引。

**推送到远程 GitHub 仓库：**

```bash
skillctl push pdf --remote --repo https://github.com/user/repo
```

或使用短参数：

```bash
skillctl push pdf -r -R https://github.com/user/repo
```

使用 GitHub API 上传技能到指定的远程仓库。

**推送查询目录：**

`push` 命令会按以下顺序在项目目录中查找技能：

1. `.github/skills`
2. `.agents/skills`
3. `.copilot/skills`
4. `examples/skills`

**远程推送认证：**

推送到远程仓库需要设置 GitHub Token。以下是不同操作系统的配置方法：

### Windows

**临时设置（当前会话）：**

```powershell
# PowerShell
$env:GITHUB_TOKEN="your_github_token"
```

```cmd
# CMD
set GITHUB_TOKEN=your_github_token
```

**永久设置：**

```powershell
# PowerShell - 设置用户级环境变量
[System.Environment]::SetEnvironmentVariable('GITHUB_TOKEN', 'your_github_token', 'User')

# 重启 PowerShell 使其生效
```

或在系统设置中手动添加：

1. 右键"此电脑" → 属性 → 高级系统设置 → 环境变量
2. 在"用户变量"中新建：
   - 变量名：`GITHUB_TOKEN`
   - 变量值：`your_github_token`

### Linux/macOS

**临时设置（当前会话）：**

```bash
# Bash/Zsh
export GITHUB_TOKEN="your_github_token"
```

**永久设置：**

```bash
# Bash - 编辑 ~/.bashrc
echo 'export GITHUB_TOKEN="your_github_token"' >> ~/.bashrc
source ~/.bashrc

# Zsh - 编辑 ~/.zshrc
echo 'export GITHUB_TOKEN="your_github_token"' >> ~/.zshrc
source ~/.zshrc
```

**在 macOS 系统偏好设置中（推荐）：**

1. 打开"系统偏好设置" → 用户与群组 → 登录项
2. 添加环境变量配置，或使用 shell 配置文件

Token 需要以下权限：

- `repo` - 完整仓库访问权限

**使用示例：**

```bash
# 从 .agents/skills 推送到本地
skillctl push claude-api

# 从 examples/skills 推送到远程
skillctl push pdf -r -R https://github.com/Nings-379/ninskills
```

#### 移除技能

**移除已安装的技能（~/.skills）：**

```bash
skillctl remove pdf
```

从 `~/.skills` 目录移除技能，并更新索引。

**移除项目中的技能：**

```bash
skillctl remove --project pdf
```

从当前项目目录下的技能目录中移除技能（`.agents/skills`、`.github/skills`、`.claude/skills`、`.copilot/skills`、`examples/skills`）。

项目技能的删除不会更新索引，因为索引只跟踪全局安装的技能。

#### 查看状态

```bash
skillctl status
```

列出所有已安装的技能。

#### 查看版本

```bash
skillctl version
```

显示版本信息。

## 技能查找优先级

当使用 `skillctl add <skill-name>` 时，系统会按以下顺序查找：

1. **`--from` 指定的目录**（默认为 `~/.skill`）
2. **已安装技能**（`~/.skills/index.json`）
3. **本地路径**（如 `./my-skill`）
4. **Git URL**（明确的 URL 格式）
5. **`~/.skills` 目录**（即使不在索引中）
6. **项目目录中的技能目录**：
   - `examples/skills`
   - `.github/skills`
   - `.agents/skills`
   - `.claude/skills`
   - `.copilot/skills`
   - `.skill`

## 存储位置

- **已安装技能**：`~/.skills/` - 全局安装的技能仓库
- **技能索引**：`~/.skills/index.json` - 技能元数据索引
- **项目技能目录**：
  - `.github/skills/` - GitHub Actions 技能
  - `.agents/skills/` - AI Agent 技能
  - `.claude/skills/` - Claude 技能
  - `.copilot/skills/` - GitHub Copilot 技能
  - `examples/skills/` - 示例技能

## 技能包结构

一个标准的技能包应包含：

```
my-skill/
├── SKILL.md         # 技能元数据（推荐）
├── skill.yaml       # 技能元数据（备选）
├── README.md        # 文档
└── [其他文件]       # 技能实现和资源
```

## VS Code 集成

添加技能后，VS Code 会自动刷新工作区，技能立即可用。支持 VS Code 识别的技能目录：

- `.github/skills`
- `.agents/skills`
- `.claude/skills`
- `.copilot/skills`
- `examples/skills`

## 推荐工作流

### 方案1：从项目推送技能到本地（推荐）

```bash
# 1. 在项目中开发技能
cd /path/to/project

# 2. 推送技能到本地已安装目录
skillctl push my-skill

# 3. 技能现在可以在任何项目中使用
skillctl add my-skill -p agents
```

### 方案2：项目内技能管理

```bash
# 1. 查看项目中的技能
skillctl list --project

# 2. 在项目内添加技能
skillctl add xlsx -p agents    # 从 ~/.skills 添加到项目
```

### 方案3：推送到远程仓库

```bash
# 1. 设置 GitHub Token
$env:GITHUB_TOKEN="your_github_token_here"

# 2. 推送技能到远程仓库
skillctl push my-skill -r -R https://github.com/user/repo

# 3. 其他人可以安装
skillctl install https://github.com/user/repo my-skill
```

## 环境变量

- `GITHUB_TOKEN` - GitHub Personal Access Token（用于远程推送）
  - 获取方式：https://github.com/settings/tokens
  - 需要权限：`repo`
- `SKILLS_DIR`: 技能存储目录（默认：`~/.skills`）

## 常见问题

### Q: skillctl add 找不到技能？

A: 请检查：

1. 使用 `skillctl list` 查看已安装的技能
2. 使用 `skillctl list --project` 查看项目中的技能
3. 如果都不在，请使用完整路径：`skillctl add ./path/to/skill -l agents`

### Q: 如何更新技能？

A: 使用 update 命令更新技能：

```bash
# 更新本地技能
skillctl update pdf

# 更新项目技能
skillctl update pdf -p

# 从指定仓库更新技能
skillctl update pdf -r my-repo
```

update 命令会：

- 从远程仓库下载最新版本
- 对比当前版本和新版本
- 如果版本相同则跳过更新
- 如果有新版本则自动更新
- 更新技能元数据

### Q: 如何只安装仓库中的一个技能？

A: 使用技能名称参数：

```bash
skillctl install https://github.com/user/my-skill pdf
```

这只会下载 `skills/pdf` 子目录，节省带宽和空间。

### Q: 如何从数据库安装技能？

A: 使用技能名称或指定仓库：

```bash
# 从默认仓库安装
skillctl install pdf

# 从指定仓库安装
skillctl install -r my-repo unit-test-generator
```

从数据库安装时会使用 sparse-checkout，只下载目标技能文件夹，不会包含整个项目。

### Q: 如何查看已索引的仓库？

A: 使用 repo list 命令：

```bash
skillctl repo list
```

这会显示所有已配置的仓库及其详细信息。

### Q: 如何查看 GitHub 仓库中有哪些技能？

A: 使用 list 命令的远程功能：

```bash
skillctl list --remote https://github.com/user/my-skill
```

### Q: 如何删除项目中的技能？

A: 使用 -p 参数：

```bash
skillctl remove --project pdf
```

这会从项目目录（`.agents/skills` 等）中删除技能，而不是从 `~/.skills` 删除。

### Q: 如何从项目推送技能？

A: 使用 push 命令：

```bash
# 推送到本地已安装目录
skillctl push pdf

# 推送到远程仓库
skillctl push pdf -r -R https://github.com/user/repo
```

### Q: 如何更新已安装的技能？

A: 使用 update 命令：

```bash
# 更新本地技能
skillctl update pdf

# 更新项目技能
skillctl update pdf -p

# 从指定仓库更新技能
skillctl update pdf -r my-repo
```

update 命令会自动检测版本变化，只在有新版本时才进行更新。

### Q: 技能安装在哪里？

A: 默认安装在用户主目录下的 `.skills` 文件夹中：

- Windows: `C:\Users\<username>\.skills\`
- Linux/macOS: `~/.skills/`

### Q: 如何在其他项目中使用技能？

A: 使用 `skillctl add` 命令将已安装的技能添加到项目：

```bash
skillctladd xlsx -p agents
```

### Q: 远程推送失败，提示 401 错误？

A: 需要设置 GitHub Token。请根据你的操作系统选择以下方式：

**Windows:**

```powershell
# PowerShell 临时设置
$env:GITHUB_TOKEN="your_github_token_here"

# PowerShell 永久设置
[System.Environment]::SetEnvironmentVariable('GITHUB_TOKEN', 'your_github_token_here', 'User')
```

**Linux/macOS:**

```bash
# 临时设置
export GITHUB_TOKEN="your_github_token_here"

# 永久设置（Bash）
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.bashrc
source ~/.bashrc

# 永久设置（Zsh）
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.zshrc
source ~/.zshrc
```

确保 token 有 `repo` 权限。获取 Token：https://github.com/settings/tokens

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
