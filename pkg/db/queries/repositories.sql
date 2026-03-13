-- name: CreateRepository :exec
INSERT INTO repositories (name, url, description, is_default, priority, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetRepositoryByName :one
SELECT * FROM repositories
WHERE name = ?;

-- name: GetRepositoryByID :one
SELECT * FROM repositories
WHERE id = ?;

-- name: ListAllRepositories :many
SELECT * FROM repositories
ORDER BY priority DESC, created_at ASC;

-- name: GetDefaultRepository :one
SELECT * FROM repositories
WHERE is_default = 1
LIMIT 1;

-- name: UpdateRepository :exec
UPDATE repositories
SET url = ?, description = ?, priority = ?, updated_at = ?
WHERE id = ?;

-- name: SetRepositoryAsDefault :exec
UPDATE repositories
SET is_default = 0;

-- name: UpdateRepositoryDefault :exec
UPDATE repositories
SET is_default = 1, updated_at = ?
WHERE id = ?;

-- name: DeleteRepository :exec
DELETE FROM repositories
WHERE id = ?;

-- name: UpdateRepositorySyncStatus :exec
UPDATE repositories
SET sync_status = ?, last_synced_at = ?, sync_error = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateRepositoryURL :exec
UPDATE repositories
SET url = ?, updated_at = ?
WHERE id = ?;

-- name: CountRepositories :one
SELECT COUNT(*) as count FROM repositories;

-- name: ListRepositoriesByStatus :many
SELECT * FROM repositories
WHERE sync_status = ?
ORDER BY priority DESC, created_at ASC;