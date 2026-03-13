# Skill - Agent Skill Management Tool

A powerful AI Agent skill management command-line tool for searching, installing, managing, and deploying skill packages.

## Overview

Skill is a command-line tool developed in Go to help AI Agent manage skill packages. Similar to how Git manages code repositories, Skill manages the lifecycle of skill packages. Supports adding skills to projects from Git repositories, local paths, or installed skills.

**Main Features:**

- ✅ Install skills from Git repositories (supports entire repository or individual skill subdirectories)
- ✅ List local, project, or remote skills
- ✅ Add skills to projects
- ✅ Search skills (supports local, project directories)
- ✅ Remove skills (supports local and project)
- ✅ Display detailed skill information (including installation time)
- ✅ Support multiple skill directory structures

## Available Commands

- `repo` - Manage remote skill repositories (add, list, remove, set default, index)
- `add` - Add a skill from local path or installed skill name to project
- `install` - Install a skill from Git repository to local (supports entire repository or single subdirectory)
- `list` - List installed skills, skills in project, or remote skills
- `push` - Upload skill to remote repository
- `remove` - Remove skill from local or project directory
- `search` - Search for skills in local directory, project directory, or installed skills
- `status` - List all installed skills

**Command Flags:**

- `-l, --local` - Search/list locally installed skills (~/.skills)
- `-p, --project` - List skills in project directory
- `-r, --repo string` - Specify repository name (for install command to install from specific repository, or for list command to list skills from database)
- `-s, --skills-dir string` - Scan skills from a specific directory

## Installation

### Build from Source

**Windows:**

```cmd
git clone https://code.cnworkshop.xyz/sought/ai-agent-skill.git
cd .\ai-agent-skill
go build -o skillctl.exe .
```

**Linux/macOS:**

```bash
git clone https://code.cnworkshop.xyz/sought/ai-agent-skill.git
cd ai-agent-skill
go build -o skillctl .
```

### Global Installation (Recommended)

```bash
go install
```

This will install `skillctl` to `~/go/bin/` directory (already in PATH), making it available anywhere.

On Linux, `go/bin` may not be in PATH. If not, you need to add it manually:

```shell
# Edit configuration file
vim ~/.bashrc
# Add the following at the end of the file

# Go environment
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
# Make configuration effective
source ~/.bashrc

# Check
go env GOPATH
ls /root/go/bin
```

## Usage

### 0. Remote Repository Management

#### Add Remote Repository

```bash

# Add repository (auto-generate name)
skillctl repo add https://github.com/user/repo

# Add repository and specify name
skillctl repo add https://github.com/user/repo -n my-repo

# Add repository and set as default
skillctl repo add https://github.com/user/repo -n my-repo --default

# Add repository and set priority
skillctl repo add https://github.com/user/repo -p 10
```

#### List All Repositories

```bash
skillctl repo list
```

Display all configured repositories and their details, including:

- Repository name
- Repository URL
- Description
- Whether it's the default repository
- Priority
- Creation time
- Last sync time
- Sync status

#### Set Default Repository

```bash
skillctl repo default <repository-name>
```

Example:

```bash
skillctl repo default my-repo
```

#### Remove Repository

```bash
skillctl repo remove <repository-name>
```

**Note**: Removing a repository will also delete all skill indexes for that repository.

#### Index/Sync Repository

```bash

# Sync default repository
skillctl repo index

# Sync specific repository
skillctl repo index <repository-name>
```

This fetches skill information from the remote repository and stores it in the local database. Supports:

- Git repository URL (uses git clone)
- REST API URL (returns JSON format skill list)

**Database location**:

- Windows: `%APPDATA%\skill\skill.db`
- Linux/macOS: `~/.cache/skill/skill.db`

### 1. List Skills

#### List Installed Skills

```bash
skillctl list
```

Display all skills installed in the `~/.skills` directory, including:

- Skill name
- Description
- Installation time (ISO 8601 format)
- Path
- Tags

**Example Output:**

```
1. pdf
   Description: Use this skill whenever the user wants to do anything with PDF files.
   Path: C:\Users\user\.skills\pdf
   Installed: 2026-03-09T10:30:00Z
```

#### List Remote Skills

**List skills from specific repository in database:**

```bash
# List all skills in specified repository
skillctl list --repo <repository-name>

# Example
skillctl list -r my-repo
```

List all skills from the specified repository in the indexed database.

**List skills in Git repository (directly from URL):**

```bash
# List all skills in the skills directory of a GitHub repository
skillctl list https://github.com/user/my-skill
skillctl list https://github.com/github/awesome-copilot
```

Automatically detects GitHub repository URLs, prioritizes using GitHub API to get skill list (fast), falls back to git clone if it fails.

For non-GitHub repositories (like Gitee), it will use git clone directly with minimal output.

**Note**: The `list` command does not support the `-R/--remote` flag. Remote URLs should be passed directly as positional arguments.

#### List Skills in Project

```bash
skillctl list --project
```

Scan all skill directories in the current project directory (`.github/skills`, `.agents/skills`, `.claude/skills`, `.copilot/skills`, `examples/skills`).

#### List Skills in Specified Directory

```bash
skillctl list --skills-dir examples/skills
skillctl list --skills-dir ~/.skill
```

Scan skills in the specified directory.

### 2. Install Skills

#### Install Entire Repository from Git URL

```bash
skillctl install https://github.com/user/my-skill
```

Clone the Git repository and install it to the `~/.skills` directory.

#### Install Single Skill from Git URL (Subdirectory)

```bash
# Only install a specific skill from the repository
skillctl install https://github.com/user/my-skill pdf
skillctl install https://github.com/user/my-skill xlsx
```

Uses Git sparse-checkout feature to only clone the `skills/<skill-name>` subdirectory from the repository, saving bandwidth and space.

**Supported Repository Structure:**

```
my-skill/
└── skills/
    ├── pdf/
    ├── xlsx/
    └── docx/
```

#### Install Skills from Database

```bash
# Install skill from default repository
skillctl install <skill-name>

# Install skill from specific repository
skillctl install -r <repo-name> <skill-name>

# Examples
skillctl install pdf
skillctl install -r my-repo unit-test-generator
```

Install skills from the indexed database. Requires using `skillctl repo index` to sync repositories first.

**Installation Process:**

1. Check if skill is already installed
2. Search for skill in database (supports exact match and fuzzy match)
3. If `-r` parameter is specified, search only in the specified repository
4. Use sparse-checkout to download only the target skill folder (not the entire project)
5. Parse skill metadata
6. Add to local index

**Note:**

- When installing from database, only the content of `skills/<skill-name>` directory is downloaded, not other files from the entire project
- After installation, directory structure: `~/.skills/<skill-name>/` (directly contains skill files, without `skills/` parent directory)
- If skill name doesn't exist in database, similar suggestions will be provided

### 3. Add Skills to Project

#### Add from Installed Skills

```bash
# First check installed skills
skillctl list

# Add skill to project
skillctl add xlsx -p agents
skillctl add pdf -p claude
skillctl add docx -p github
```

Finds and copies skills from `~/.skills`, project directory, or `~/.skill` directory to the specified location.

#### Add from Local Path

```bash
# Add from relative path
skillctl add ./local-skill -p agents

# Add from absolute path
skillctl add C:\path\to\skill -p claude
```

#### Add from Git URL to Project

```bash
# Add directly to project from Git URL
skillctl add https://github.com/user/my-skill -p agents
```

#### Supported Target Locations

- `github` → `.github/skills`
- `agents` → `.agents/skills`
- `claude` → `.claude/skills`
- `copilot` → `.copilot/skills`
- Custom path

#### Add to Project Directory

```bash
# Add to skills directory of current project
skillctl add my-skill --project

# Add to skills directory of specified project
skillctl add my-skill --project --dir /path/to/project
```

### 4. Skill Repository Management

#### Create Global Skill Repository

```bash
# Create .skill folder in user's home directory
mkdir ~/.skill

# Copy common skills to global repository
cp -r examples/skills/xlsx ~/.skill/xlsx
cp -r examples/skills/pdf ~/.skill/pdf
```

#### Use Global Skill Repository

```bash
# Use global skills in any project (search from ~/.skill by default)
cd /path/to/any-project
skillctl add xlsx -p agents
skillctl add pdf -p claude
```

#### View Global Skill Repository

```bash
skillctl list --skills-dir ~/.skill
```

#### Search Skills

**Search locally installed skills (~/.skills):**

```bash
skillctl search --local pdf
skillctl search -l document
```

Search for skills in the `~/.skills` directory, including installed skills in the index.

**Search skills in project:**

```bash
skillctl search --project pdf
skillctl search -p document
```

Search in skill directories under the project directory (`.agents/skills`, `.github/skills`, `.claude/skills`, `.copilot/skills`, `examples/skills`).

Search for skills in the `./skills` folder in the current directory.

**Search Scope:**

- Name
- Description
- Tags

### 6. Push Skills

**Push to local installed skill directory (default):**

```bash
skillctl push pdf
```

Finds skills from the project directory (`.github/skills`, `.agents/skills`, `.copilot/skills`, `examples/skills`) and pushes them to the `~/.skills` directory, updating the index.

**Push to remote GitHub repository:**

```bash
skillctl push pdf --remote --repo https://github.com/user/repo
```

Or use short flags:

```bash
skillctl push pdf -r -R https://github.com/user/repo
```

Upload skills to the specified remote repository using GitHub API.

**Push Query Directories:**

The `push` command searches for skills in the project directory in the following order:

1. `.github/skills`
2. `.agents/skills`
3. `.copilot/skills`
4. `examples/skills`

**Remote Push Authentication:**

Pushing to remote repositories requires setting up a GitHub Token. Here are configuration methods for different operating systems:

Test repository and token currently in use:
url: https://github.com/Nings-379/ninskills
token: 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

### Windows

**Temporary setting (current session):**

```powershell
# PowerShell
$env:GITHUB_TOKEN="your_github_token"
```

```cmd
# CMD
set GITHUB_TOKEN=your_github_token
```

**Permanent setting:**

```powershell
# PowerShell - Set user-level environment variable
[System.Environment]::SetEnvironmentVariable('GITHUB_TOKEN', 'your_github_token', 'User')

# Restart PowerShell to take effect
```

Or add manually in system settings:

1. Right-click "This PC" → Properties → Advanced system settings → Environment variables
2. Create new in "User variables":
   - Variable name: `GITHUB_TOKEN`
   - Variable value: `your_github_token`

### Linux/macOS

**Temporary setting (current session):**

```bash
# Bash/Zsh
export GITHUB_TOKEN="your_github_token"
```

**Permanent setting:**

```bash
# Bash - Edit ~/.bashrc
echo 'export GITHUB_TOKEN="your_github_token"' >> ~/.bashrc
source ~/.bashrc

# Zsh - Edit ~/.zshrc
echo 'export GITHUB_TOKEN="your_github_token"' >> ~/.zshrc
source ~/.zshrc
```

**In macOS System Preferences (recommended):**

1. Open "System Preferences" → Users & Groups → Login Items
2. Add environment variable configuration, or use shell configuration file

Token requires the following permissions:

- `repo` - Full repository access

**Usage Example:**

```bash
# Push from .agents/skills to local
skillctl push claude-api

# Push from examples/skills to remote
skillctl push pdf -r -R https://github.com/Nings-379/ninskills
```

#### Remove Skills

**Remove installed skills (~/.skills):**

```bash
skillctl remove pdf
```

Remove skills from the `~/.skills` directory and update the index.

**Remove skills in project:**

```bash
skillctl remove --project pdf
```

Remove skills from skill directories under the current project directory (`.agents/skills`, `.github/skills`, `.claude/skills`, `.copilot/skills`, `examples/skills`).

Deleting project skills does not update the index, as the index only tracks globally installed skills.

#### View Status

```bash
skillctl status
```

List all installed skills.

## Skill Search Priority

When using `skillctl add <skill-name>`, the system searches in the following order:

1. **Directory specified by `--from`** (default is `~/.skill`)
2. **Installed skills** (`~/.skills/index.json`)
3. **Local path** (like `./my-skill`)
4. **Git URL** (explicit URL format)
5. **`~/.skills` directory** (even if not in index)
6. **Skill directories in project directory**:
   - `examples/skills`
   - `.github/skills`
   - `.agents/skills`
   - `.claude/skills`
   - `.copilot/skills`
   - `.skill`

## Storage Locations

- **Installed skills**: `~/.skills/` - Global installed skill repository
- **Skill index**: `~/.skills/index.json` - Skill metadata index
- **Project skill directories**:
  - `.github/skills/` - GitHub Actions skills
  - `.agents/skills/` - AI Agent skills
  - `.claude/skills/` - Claude skills
  - `.copilot/skills/` - GitHub Copilot skills
  - `examples/skills/` - Example skills

## Skill Package Structure

A standard skill package should contain:

```
my-skill/
├── SKILL.md         # Skill metadata (recommended)
├── skill.yaml       # Skill metadata (alternative)
├── README.md        # Documentation
└── [other files]    # Skill implementation and resources
```

## VS Code Integration

After adding skills, VS Code automatically refreshes the workspace, making skills immediately available. Supports VS Code recognized skill directories:

- `.github/skills`
- `.agents/skills`
- `.claude/skills`
- `.copilot/skills`
- `examples/skills`

## Recommended Workflows

### Solution 1: Push Skills from Project to Local (Recommended)

```bash
# 1. Develop skills in project
cd /path/to/project

# 2. Push skills to local installed directory
skillctl push my-skill

# 3. Skills can now be used in any project
skillctl add my-skill -p agents
```

### Solution 2: In-Project Skill Management

```bash
# 1. View skills in project
skillctl list --project

# 2. Add skills in project
skillctl add xlsx -p agents    # Add from ~/.skills to project
```

### Solution 3: Push to Remote Repository

```bash
# 1. Set GitHub Token
$env:GITHUB_TOKEN="your_github_token_here"

# 2. Push skills to remote repository
skillctl push my-skill -r -R https://github.com/user/repo

# 3. Others can install
skillctl install https://github.com/user/repo my-skill
```

## Environment Variables

- `GITHUB_TOKEN` - GitHub Personal Access Token (for remote push)
  - Get it from: https://github.com/settings/tokens
  - Required permission: `repo`
- `SKILLS_DIR`: Skill storage directory (default: `~/.skills`)

## FAQ

### Q: skillctl add can't find skills?

A: Please check:

1. Use `skillctl list` to see installed skills
2. Use `skillctl list --project` to see skills in project
3. If neither, use full path: `skillctl add ./path/to/skill -l agents`

### Q: How to update skills?

A: Reinstall:

```bash
skillctl install https://github.com/user/my-skill
```

### Q: How to install only one skill from a repository?

A: Use skill name parameter:

```bash
skillctl install https://github.com/user/my-skill pdf
```

This only downloads the `skills/pdf` subdirectory, saving bandwidth and space.

### Q: How to install skills from database?

A: Use skill name or specify repository:

```bash
# Install from default repository
skillctl install pdf

# Install from specific repository
skillctl install -r my-repo unit-test-generator
```

When installing from database, sparse-checkout is used to download only the target skill folder, not the entire project.

### Q: How to view indexed repositories?

A: Use repo list command:

```bash
skillctl repo list
```

This displays all configured repositories and their details.

### Q: How to see what skills are in a GitHub repository?

A: Use the remote feature of the list command:

```bash
skillctl list https://github.com/user/my-skill
```

### Q: How to delete skills in project?

A: Use -p parameter:

```bash
skillctl remove --project pdf
```

This deletes skills from the project directory (`.agents/skills`, etc.), not from `~/.skills`.

### Q: How to push skills from project?

A: Use push command:

```bash
# Push to local installed directory
skillctl push pdf

# Push to remote repository
skillctl push pdf -r -R https://github.com/user/repo
```

### Q: Where are skills installed?

A: By default in the `.skills` folder in the user's home directory:

- Windows: `C:\Users\<username>\.skills\`
- Linux/macOS: `~/.skills/`

### Q: How to use skills in other projects?

A: Use `skillctl add` command to add installed skills to project:

```bash
skillctl add xlsx -p agents
```

### Q: Remote push fails with 401 error?

A: Need to set GitHub Token. Choose one of the following methods based on your operating system:

**Windows:**

```powershell
# PowerShell temporary setting
$env:GITHUB_TOKEN="your_github_token_here"

# PowerShell permanent setting
[System.Environment]::SetEnvironmentVariable('GITHUB_TOKEN', 'your_github_token_here', 'User')
```

**Linux/macOS:**

```bash
# Temporary setting
export GITHUB_TOKEN="your_github_token_here"

# Permanent setting (Bash)
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.bashrc
source ~/.bashrc

# Permanent setting (Zsh)
echo 'export GITHUB_TOKEN="your_github_token_here"' >> ~/.zshrc
source ~/.zshrc
```

Ensure token has `repo` permission. Get Token: https://github.com/settings/tokens

## Contributing

Issues and Pull Requests are welcome!

## License

MIT License
