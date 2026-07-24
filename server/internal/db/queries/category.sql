-- name: ListActiveCategories :many
SELECT id, name, icon, sort_order, status, created_at, updated_at
FROM category
WHERE status = 'ACTIVE'
ORDER BY sort_order ASC;
