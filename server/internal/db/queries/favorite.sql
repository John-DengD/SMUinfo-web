-- name: InsertFavorite :exec
INSERT INTO favorite (user_id, product_id)
VALUES ($1, $2);

-- name: DeleteFavorite :exec
DELETE FROM favorite WHERE user_id = $1 AND product_id = $2;

-- name: CountFavorite :one
SELECT count(*) FROM favorite WHERE user_id = $1 AND product_id = $2;

-- name: ListFavoriteProductIDs :many
SELECT product_id FROM favorite
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListProductsByIDs :many
SELECT id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at
FROM product WHERE id = ANY(sqlc.arg('ids')::bigint[]);
