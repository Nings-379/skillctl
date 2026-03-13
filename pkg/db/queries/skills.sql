-- name: CreateSkill :exec
INSERT INTO skills (
    repository_id, name, slug, description, version, author, 
    repo_url, repo_owner, repo_name, stars, forks, downloads, 
    language, license, homepage, category, security_status, 
    created_at, updated_at, indexed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSkillByID :one
SELECT * FROM skills WHERE id = ?;

-- name: GetSkillByRepositoryAndSlug :one
SELECT * FROM skills
WHERE repository_id = ? AND slug = ?;

-- name: ListSkillsByRepository :many
SELECT * FROM skills
WHERE repository_id = ?
ORDER BY name ASC;

-- name: ListAllSkills :many
SELECT * FROM skills
ORDER BY repository_id, name ASC;

-- name: SearchSkillsByName :many
SELECT * FROM skills
WHERE name LIKE ?
ORDER BY name ASC;

-- name: SearchSkillsByCategory :many
SELECT * FROM skills
WHERE category = ?
ORDER BY name ASC;

-- name: SearchSkillsByAuthor :many
SELECT * FROM skills
WHERE author LIKE ?
ORDER BY name ASC;

-- name: UpdateSkill :exec
UPDATE skills
SET description = ?, version = ?, author = ?, repo_url = ?, 
    repo_owner = ?, repo_name = ?, stars = ?, forks = ?, downloads = ?, 
    language = ?, license = ?, homepage = ?, category = ?, 
    security_status = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteSkill :exec
DELETE FROM skills WHERE id = ?;

-- name: DeleteSkillsByRepository :exec
DELETE FROM skills WHERE repository_id = ?;

-- name: CountSkills :one
SELECT COUNT(*) as count FROM skills;

-- name: CountSkillsByRepository :one
SELECT COUNT(*) as count FROM skills WHERE repository_id = ?;

-- name: UpsertSkill :exec
INSERT OR REPLACE INTO skills (
    id, repository_id, name, slug, description, version, author,
    repo_url, repo_owner, repo_name, stars, forks, downloads,
    language, license, homepage, category, security_status,
    created_at, updated_at, indexed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);