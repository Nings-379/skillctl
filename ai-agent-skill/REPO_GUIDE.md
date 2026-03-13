# Skillctl Repo Commands 使用指南

## 概述

`skillctl repo` 命令集提供了完整的远程仓库管理功能,允许用户:

- 添加、列出、删除远程技能仓库
- 设置默认仓库
- 同步远程仓库的技能到本地SQLite数据库
- 管理技能索引

## 数据库设计

### 数据库位置

- **Windows**: `%APPDATA%\skill\skill.db`
- **Linux/macOS**: `~/.cache/skill/skill.db`

### 数据库表结构

#### 1. repositories 表
存储远程仓库信息
- `id`: 主键
- `name`: 仓库名称 (唯一)
- `url`: 仓库URL
- `description`: 仓库描述
- `is_default`: 是否为默认仓库
- `priority`: 优先级
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `last_synced_at`: 最后同步时间
- `sync_status`: 同步状态
- `sync_error`: 同步错误信息

#### 2. skills 表
存储技能信息
- `id`: 主键
- `repository_id`: 所属仓库ID (外键)
- `name`: 技能名称
- `slug`: 技能标识符
- `description`: 技能描述
- `version`: 版本号
- `author`: 作者
- `repo_url`: 技能的Git仓库URL
- `repo_owner`: 仓库所有者
- `repo_name`: 仓库名称
- `stars`: GitHub stars数
- `forks`: forks数
- `downloads`: 下载次数
- `language`: 主要编程语言
- `license`: 许可证
- `homepage`: 主页URL
- `category`: 分类
- `security_status`: 安全状态
- `created_at`: 技能创建时间
- `updated_at`: 技能更新时间
- `indexed_at`: 索引时间

#### 3. tags 表
存储技能标签 (多对多关系)
- `id`: 主键
- `skill_id`: 关联的技能ID (外键)
- `tag`: 标签名称

#### 4. db_version 表
数据库版本管理
- `version`: 版本号
- `applied_at`: 应用时间
- `description`: 版本描述

## 命令使用说明

### 1. 添加远程仓库

```bash
skillctl repo add <url> [选项]
```

**选项:**
- `-n, --name <name>`: 仓库名称 (默认: 自动生成)
- `-d, --description <desc>`: 仓库描述
- `-D, --default`: 设置为默认仓库
- `-p, --priority <num>`: 优先级 (数字越大优先级越高)

**示例:**

```bash
# 添加仓库 (自动生成名称)
skillctl repo add https://github.com/Nings-379/ninskills

# 添加仓库并指定名称
skillctl repo add https://skillsllm.com/api/skills -n official

# 添加仓库并设置为默认
skillctl repo add https://github.com/user/repo --default

# 添加仓库并设置优先级
skillctl repo add https://github.com/user/repo --priority 10
```

### 2. 列出所有仓库

```bash
skillctl repo list
```

**输出信息包括:**
- 仓库名称
- 仓库URL
- 描述
- 是否为默认仓库
- 优先级
- 创建时间
- 最后同步时间
- 同步状态

**示例:**

```
📋 Remote Repositories:
============================================================

1. official
   URL: https://skillsllm.com/api/skills
   Description: Official skills repository
   Default: ✓   Priority: 0
   Added: 2026-03-11 13:20:00
   Last synced: 2026-03-11 13:25:00
   Sync status: ✅ success

2. ninskills
   URL: https://github.com/Nings-379/ninskills
   Default: ✗   Priority: 0
   Added: 2026-03-11 13:15:00

Total: 2 repository(ies)
```

### 3. 删除仓库

```bash
skillctl repo remove <name>
```

**说明:**
- 删除仓库会同时删除该仓库的所有技能索引
- 需要确认操作

**示例:**

```bash
skillctl repo remove ninskills
```

### 4. 设置默认仓库

```bash
skillctl repo default <name>
```

**说明:**
- 设置指定的仓库为默认仓库
- 默认仓库用于未指定仓库的操作

**示例:**

```bash
skillctl repo default official
```

### 5. 索引/同步仓库

```bash
skillctl repo index [name]
```

**说明:**
- 如果指定了仓库名称,则同步该仓库
- 如果未指定仓库名称,则同步默认仓库
- 从远程仓库获取技能信息并存储到本地数据库

**示例:**

```bash
# 同步默认仓库
skillctl repo index

# 同步指定仓库
skillctl repo index official

# 同步另一个仓库
skillctl repo index ninskills
```

**输出示例:**

```
📥 Using default repository: official
   URL: https://skillsllm.com/api/skills

🔄 Fetching skills from repository...
✓ Found 150 skill(s)

💾 Syncing skills to database...

✅ Index completed successfully!
   New skills: 50
   Updated skills: 100
   Total skills: 150
```

## 技术实现

### 使用的工具

- **sqlc**: 生成类型安全的数据库访问代码
- **modernc.org/sqlite**: 纯Go实现的SQLite驱动
- **Cobra**: CLI命令框架

### 项目结构

```
skillctl/
├── cmd/
│   ├── repo.go              # repo命令根命令
│   ├── repo_add.go          # repo add命令
│   ├── repo_list.go         # repo list命令
│   ├── repo_remove.go       # repo remove命令
│   ├── repo_default.go      # repo default命令
│   └── repo_index.go        # repo index命令
├── pkg/
│   └── db/
│       ├── database.go      # 数据库连接和初始化
│       ├── manager.go       # 数据库管理器
│       ├── queries/         # SQL查询文件
│       │   ├── repositories.sql
│       │   ├── skills.sql
│       │   ├── tags.sql
│       │   └── version.sql
│       ├── db.go            # sqlc生成的数据库接口
│       ├── models.go        # sqlc生成的模型
│       ├── querier.go       # sqlc生成的查询接口
│       ├── repositories.sql.go
│       ├── skills.sql.go
│       ├── tags.sql.go
│       └── version.sql.go
├── migrations/
│   └── 0001_initial_schema.sql  # 数据库迁移文件
└── sqlc.yaml                  # sqlc配置文件
```

### 数据库迁移

使用SQL迁移文件管理数据库版本:

```sql
-- migrations/0001_initial_schema.sql
-- 包含完整的数据库表结构定义
```

### 数据库操作示例

#### 添加仓库

```go
manager, err := db.NewManager()
if err != nil {
    return err
}
defer manager.Close()

params := db.CreateRepositoryParams{
    Name:        "official",
    Url:         "https://skillsllm.com/api/skills",
    Description: sql.NullString{String: "Official repository", Valid: true},
    IsDefault:   1,
    Priority:    0,
    CreatedAt:   time.Now().Format(time.RFC3339),
    UpdatedAt:   time.Now().Format(time.RFC3339),
}

err = manager.GetQueries().CreateRepository(ctx, params)
```

#### 同步技能

```go
// 从远程API获取技能
skills, err := fetchSkillsFromRepository(repo.Url)

// 同步到数据库
for _, skill := range skills {
    params := db.CreateSkillParams{
        RepositoryID: repo.ID,
        Name:        skill.Name,
        Slug:        skill.Slug,
        // ... 其他字段
    }
    
    skillID, err := manager.GetQueries().CreateSkill(ctx, params)
    
    // 添加标签
    manager.AddTagsToSkill(ctx, skillID, skill.Tags)
}
```

## 支持的远程仓库类型

### 1. REST API

标准的REST API返回JSON格式的技能列表:

```json
{
  "skills": [
    {
      "id": "1",
      "slug": "pdf",
      "name": "PDF",
      "description": "PDF processing skill",
      "repoUrl": "https://github.com/user/pdf-skill",
      "topics": ["pdf", "document"],
      "language": "Go",
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-15T00:00:00Z"
    }
  ]
}
```

### 2. GitHub仓库

支持从GitHub仓库获取技能信息。

## 高级功能

### 事务支持

数据库管理器支持事务操作:

```go
err := manager.Transaction(ctx, func(q *db.Queries) error {
    // 执行多个数据库操作
    if err := q.CreateRepository(ctx, params); err != nil {
        return err
    }
    // ... 其他操作
    return nil
})
```

### 批量操作

支持批量添加标签:

```go
manager.AddTagsToSkill(ctx, skillID, []string{"pdf", "document", "processing"})
```

### 查询优化

数据库包含以下索引以优化查询性能:
- 仓库默认状态索引
- 仓库优先级索引
- 技能仓库ID索引
- 技能名称索引
- 技能分类索引
- 技能作者索引
- 标签技能ID索引
- 标签名称索引

## 错误处理

所有命令都包含完善的错误处理:

- 数据库连接错误
- 仓库不存在错误
- 网络请求错误
- 数据验证错误

错误信息会以友好的方式显示给用户。

## 未来扩展

可以轻松扩展的功能:

1. **搜索功能**: 基于数据库的技能搜索
2. **过滤功能**: 按分类、标签、作者等过滤技能
3. **排序功能**: 按不同字段排序
4. **批量操作**: 批量添加/删除仓库
5. **仓库验证**: 添加仓库前验证URL有效性
6. **增量同步**: 只同步更新的技能
7. **技能详情**: 查看单个技能的详细信息

## 总结

`skillctl repo` 命令集提供了完整的远程仓库管理解决方案:

✅ 完整的CRUD操作 (创建、读取、更新、删除)
✅ SQLite数据库持久化
✅ 类型安全的数据库访问 (sqlc)
✅ 完善的错误处理
✅ 友好的用户界面
✅ 支持多种远程仓库类型
✅ 可扩展的架构设计

这个实现为skillctl项目提供了强大的仓库管理能力,用户可以轻松管理多个远程技能仓库,并保持本地数据库与远程仓库的同步。