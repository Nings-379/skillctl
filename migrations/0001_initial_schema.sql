-- 初始数据库结构
-- 创建仓库表
CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    description TEXT,
    is_default INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_synced_at TEXT,
    sync_status TEXT DEFAULT 'pending',
    sync_error TEXT
);

-- 创建技能表
CREATE TABLE IF NOT EXISTS skills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    version TEXT,
    author TEXT,
    repo_url TEXT,
    repo_owner TEXT,
    repo_name TEXT,
    stars INTEGER DEFAULT 0,
    forks INTEGER DEFAULT 0,
    downloads INTEGER DEFAULT 0,
    language TEXT,
    license TEXT,
    homepage TEXT,
    category TEXT,
    security_status TEXT,
    created_at TEXT,
    updated_at TEXT,
    indexed_at TEXT NOT NULL,
    UNIQUE(repository_id, slug),
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE
);

-- 创建标签表
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id INTEGER NOT NULL,
    tag TEXT NOT NULL,
    UNIQUE(skill_id, tag),
    FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_repositories_is_default ON repositories(is_default);
CREATE INDEX IF NOT EXISTS idx_repositories_priority ON repositories(priority);
CREATE INDEX IF NOT EXISTS idx_skills_repository_id ON skills(repository_id);
CREATE INDEX IF NOT EXISTS idx_skills_name ON skills(name);
CREATE INDEX IF NOT EXISTS idx_skills_category ON skills(category);
CREATE INDEX IF NOT EXISTS idx_skills_author ON skills(author);
CREATE INDEX IF NOT EXISTS idx_tags_skill_id ON tags(skill_id);
CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);