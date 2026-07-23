-- name: CountAdminUsers :one
SELECT count(*)
FROM "user"
WHERE (sqlc.narg('keyword')::text IS NULL
        OR name LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR student_no LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR phone LIKE '%' || sqlc.narg('keyword')::text || '%');

-- name: ListAdminUsers :many
SELECT id, name, student_no, phone, college, campus, avatar, role, status
FROM "user"
WHERE (sqlc.narg('keyword')::text IS NULL
        OR name LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR student_no LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR phone LIKE '%' || sqlc.narg('keyword')::text || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: SetUserStatus :exec
UPDATE "user" SET status = $2 WHERE id = $1;

-- name: InsertCategory :one
INSERT INTO category (name, icon, sort_order, status)
VALUES ($1, $2, $3, $4)
RETURNING id, name, icon, sort_order, status, created_at, updated_at;

-- name: UpdateCategory :one
UPDATE category
SET name = $2, icon = $3, sort_order = $4, status = $5
WHERE id = $1
RETURNING id, name, icon, sort_order, status, created_at, updated_at;

-- name: GetCategory :one
SELECT id, name, icon, sort_order, status, created_at, updated_at
FROM category WHERE id = $1;

-- name: DeleteCategory :exec
DELETE FROM category WHERE id = $1;

-- name: ListReports :many
SELECT id, reporter_id, product_id, reason, status, admin_remark, created_at, updated_at
FROM report
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC;

-- name: GetReport :one
SELECT id, reporter_id, product_id, reason, status, admin_remark, created_at, updated_at
FROM report WHERE id = $1;

-- name: UpdateReport :exec
UPDATE report SET status = $2, admin_remark = $3 WHERE id = $1;

-- name: ListProductTitlesByIDs :many
SELECT id, title FROM product WHERE id = ANY(sqlc.arg('ids')::bigint[]);

-- name: ListAllFeedback :many
SELECT id, user_id, category, content, contact, status, admin_reply, created_at, updated_at
FROM feedback
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC;

-- name: GetFeedback :one
SELECT id, user_id, category, content, contact, status, admin_reply, created_at, updated_at
FROM feedback WHERE id = $1;

-- name: UpdateFeedbackReply :exec
UPDATE feedback SET status = $2, admin_reply = $3 WHERE id = $1;

-- name: ListAllAnnouncements :many
SELECT id, title, content, status, created_by, created_at, updated_at
FROM announcement
ORDER BY created_at DESC;

-- name: GetAnnouncement :one
SELECT id, title, content, status, created_by, created_at, updated_at
FROM announcement WHERE id = $1;

-- name: InsertAnnouncement :one
INSERT INTO announcement (title, content, status, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, title, content, status, created_by, created_at, updated_at;

-- name: UpdateAnnouncement :one
UPDATE announcement
SET title = $2, content = $3, status = $4
WHERE id = $1
RETURNING id, title, content, status, created_by, created_at, updated_at;

-- name: DeleteAnnouncement :exec
DELETE FROM announcement WHERE id = $1;

-- name: DisableOtherActiveAnnouncements :exec
UPDATE announcement SET status = 'INACTIVE'
WHERE status = 'ACTIVE' AND id <> $1;
