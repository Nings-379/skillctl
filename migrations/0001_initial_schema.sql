-- 初始数据库结构
-- 创建仓库表
CREATE TABLE repositories (
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
CREATE TABLE skills (
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
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id INTEGER NOT NULL,
    tag TEXT NOT NULL,
    UNIQUE(skill_id, tag),
    FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE CASCADE
);

-- 创建数据库版本表
CREATE TABLE db_version (
    version INTEGER NOT NULL PRIMARY KEY,
    applied_at TEXT NOT NULL,
    description TEXT
);

-- 插入初始版本记录
INSERT INTO db_version (version, applied_at, description) 
VALUES (1, datetime('now'), 'Initial schema');

-- 创建索引
CREATE INDEX idx_repositories_is_default ON repositories(is_default);
CREATE INDEX idx_repositories_priority ON repositories(priority);
CREATE INDEX idx_skills_repository_id ON skills(repository_id);
CREATE INDEX idx_skills_name ON skills(name);
CREATE INDEX idx_skills_category ON skills(category);
CREATE INDEX idx_skills_author ON skills(author);
CREATE INDEX idx_tags_skill_id ON tags(skill_id);
CREATE INDEX idx_tags_tag ON tags(tag);