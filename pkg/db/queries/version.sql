-- name: GetCurrentVersion :one
SELECT version FROM db_version
ORDER BY version DESC
LIMIT 1;

-- name: GetVersionByNumber :one
SELECT * FROM db_version WHERE version = ?;

-- name: ListAllVersions :many
SELECT * FROM db_version ORDER BY version ASC;

-- name: InsertVersion :exec
INSERT INTO db_version (version, applied_at, description)
VALUES (?, ?, ?);