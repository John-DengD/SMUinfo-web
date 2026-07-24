-- name: GetActiveAnnouncement :one
SELECT id, title, content, status, created_by, created_at, updated_at
FROM announcement
WHERE status = 'ACTIVE'
ORDER BY created_at DESC
LIMIT 1;
