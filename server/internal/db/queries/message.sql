-- name: InsertMessage :one
INSERT INTO message (sender_id, receiver_id, product_id, content, is_read)
VALUES ($1, $2, $3, $4, false)
RETURNING id, sender_id, receiver_id, product_id, content, is_read, created_at;

-- name: GetUserByIDForMessage :one
SELECT id, name, avatar FROM "user" WHERE id = $1;

-- name: ListMessagesByUser :many
-- All messages where the user is sender or receiver, newest first.
SELECT id, sender_id, receiver_id, product_id, content, is_read, created_at
FROM message
WHERE sender_id = $1 OR receiver_id = $1
ORDER BY created_at DESC, id DESC;

-- name: ListMessagesBetween :many
-- Full thread between two users, oldest first.
SELECT id, sender_id, receiver_id, product_id, content, is_read, created_at
FROM message
WHERE (sender_id = $1 AND receiver_id = $2)
   OR (sender_id = $2 AND receiver_id = $1)
ORDER BY created_at ASC, id ASC;

-- name: MarkMessagesRead :exec
-- Mark unread messages sent by $2 to $1 as read.
UPDATE message
SET is_read = true
WHERE receiver_id = $1 AND sender_id = $2 AND is_read = false;

-- name: CountUnreadMessages :one
SELECT count(*) FROM message
WHERE receiver_id = $1 AND is_read = false;

-- name: GetProductTitleForMessage :one
SELECT title FROM product WHERE id = $1;
