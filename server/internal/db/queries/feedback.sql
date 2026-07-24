-- name: InsertFeedback :one
INSERT INTO feedback (user_id, category, content, contact, status)
VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, category, content, contact, status, admin_reply, created_at, updated_at;

-- name: ListFeedbackByUser :many
SELECT id, user_id, category, content, contact, status, admin_reply, created_at, updated_at
FROM feedback
WHERE user_id = $1
ORDER BY created_at DESC;
