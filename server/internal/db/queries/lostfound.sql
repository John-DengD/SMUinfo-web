-- name: GetLostFound :one
SELECT id, user_id, type, title, description, location, contact, status, view_count, event_time, created_at, updated_at
FROM lost_found WHERE id = $1;

-- name: InsertLostFound :one
INSERT INTO lost_found (user_id, type, title, description, location, contact, status, view_count, event_time)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, user_id, type, title, description, location, contact, status, view_count, event_time, created_at, updated_at;

-- name: IncrementLostFoundView :exec
UPDATE lost_found SET view_count = view_count + 1 WHERE id = $1;

-- name: SetLostFoundStatus :exec
UPDATE lost_found SET status = $2 WHERE id = $1;

-- name: CountLostFound :one
SELECT count(*)
FROM lost_found
WHERE status = 'OPEN'
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text)
  AND (sqlc.narg('keyword')::text IS NULL
        OR title LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR description LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR location LIKE '%' || sqlc.narg('keyword')::text || '%');

-- name: ListLostFound :many
SELECT id, user_id, type, title, description, location, contact, status, view_count, event_time, created_at, updated_at
FROM lost_found
WHERE status = 'OPEN'
  AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text)
  AND (sqlc.narg('keyword')::text IS NULL
        OR title LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR description LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR location LIKE '%' || sqlc.narg('keyword')::text || '%')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: ListLostFoundImages :many
SELECT id, lost_found_id, image_url, sort_order, created_at
FROM lost_found_image
WHERE lost_found_id = ANY(sqlc.arg('lost_found_ids')::bigint[])
ORDER BY sort_order ASC, id ASC;

-- name: InsertLostFoundImage :exec
INSERT INTO lost_found_image (lost_found_id, image_url, sort_order)
VALUES ($1, $2, $3);

-- name: DeleteLostFoundImages :exec
DELETE FROM lost_found_image WHERE lost_found_id = $1;
