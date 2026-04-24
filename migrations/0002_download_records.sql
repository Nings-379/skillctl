-- 删除旧的数据库版本表
DROP TABLE IF EXISTS db_version;

-- 记录本地工具安装技能的下载/安装历史
CREATE TABLE IF NOT EXISTS skill_downloads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_name TEXT NOT NULL,
    installed_as TEXT NOT NULL,
    skill_version TEXT,
    source_type TEXT NOT NULL,
    source_name TEXT,
    source_url TEXT,
    downloader TEXT NOT NULL,
    downloader_host TEXT,
    downloaded_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_skill_downloads_skill_name ON skill_downloads(skill_name);
CREATE INDEX IF NOT EXISTS idx_skill_downloads_installed_as ON skill_downloads(installed_as);
CREATE INDEX IF NOT EXISTS idx_skill_downloads_downloaded_at ON skill_downloads(downloaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_skill_downloads_downloader ON skill_downloads(downloader);