-- name: CreateSkillDownload :exec
INSERT INTO skill_downloads (
    skill_name,
    installed_as,
    skill_version,
    source_type,
    source_name,
    source_url,
    downloader,
    downloader_host,
    downloaded_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListSkillDownloads :many
SELECT * FROM skill_downloads
ORDER BY downloaded_at DESC, id DESC;

-- name: ListSkillDownloadsBySkill :many
SELECT * FROM skill_downloads
WHERE skill_name = ? OR installed_as = ?
ORDER BY downloaded_at DESC, id DESC;