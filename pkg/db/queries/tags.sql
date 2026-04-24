-- name: CreateTag :exec
INSERT INTO tags (skill_id, tag) VALUES (?, ?);

-- name: GetTagsBySkillID :many
SELECT * FROM tags WHERE skill_id = ?;

-- name: DeleteTagsBySkillID :exec
DELETE FROM tags WHERE skill_id = ?;

-- name: DeleteTag :exec
DELETE FROM tags WHERE skill_id = ? AND tag = ?;

-- name: SearchSkillsByTag :many
SELECT DISTINCT s.* FROM skills s
INNER JOIN tags t ON s.id = t.skill_id
WHERE t.tag = ?
ORDER BY s.name ASC;

-- name: GetPopularTags :many
SELECT tag, COUNT(*) as count 
FROM tags 
GROUP BY tag 
ORDER BY count DESC 
LIMIT ?;