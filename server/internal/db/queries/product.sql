-- name: GetProduct :one
SELECT id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at
FROM product WHERE id = $1;

-- name: InsertProduct :one
INSERT INTO product (seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at;

-- name: UpdateProduct :one
UPDATE product
SET title = $2, description = $3, category_id = $4, price = $5, original_price = $6,
    condition_level = $7, trade_location = $8, status = $9
WHERE id = $1
RETURNING id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at;

-- name: IncrementProductView :exec
UPDATE product SET view_count = view_count + 1 WHERE id = $1;

-- name: SetProductStatus :exec
UPDATE product SET status = $2 WHERE id = $1;

-- name: CountProducts :one
SELECT count(*)
FROM product
WHERE (sqlc.narg('keyword')::text IS NULL
        OR title LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR description LIKE '%' || sqlc.narg('keyword')::text || '%')
  AND (sqlc.narg('category_id')::bigint IS NULL OR category_id = sqlc.narg('category_id')::bigint)
  AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
  AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric)
  AND (sqlc.narg('condition_level')::text IS NULL OR condition_level = sqlc.narg('condition_level')::text)
  AND (sqlc.narg('campus')::text IS NULL OR trade_location LIKE '%' || sqlc.narg('campus')::text || '%')
  AND (sqlc.narg('seller_id')::bigint IS NULL OR seller_id = sqlc.narg('seller_id')::bigint)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT sqlc.arg('apply_default_status')::bool OR status IN ('ON_SALE', 'RESERVED'));

-- name: ListProducts :many
SELECT id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at
FROM product
WHERE (sqlc.narg('keyword')::text IS NULL
        OR title LIKE '%' || sqlc.narg('keyword')::text || '%'
        OR description LIKE '%' || sqlc.narg('keyword')::text || '%')
  AND (sqlc.narg('category_id')::bigint IS NULL OR category_id = sqlc.narg('category_id')::bigint)
  AND (sqlc.narg('min_price')::numeric IS NULL OR price >= sqlc.narg('min_price')::numeric)
  AND (sqlc.narg('max_price')::numeric IS NULL OR price <= sqlc.narg('max_price')::numeric)
  AND (sqlc.narg('condition_level')::text IS NULL OR condition_level = sqlc.narg('condition_level')::text)
  AND (sqlc.narg('campus')::text IS NULL OR trade_location LIKE '%' || sqlc.narg('campus')::text || '%')
  AND (sqlc.narg('seller_id')::bigint IS NULL OR seller_id = sqlc.narg('seller_id')::bigint)
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (NOT sqlc.arg('apply_default_status')::bool OR status IN ('ON_SALE', 'RESERVED'))
ORDER BY
  CASE WHEN sqlc.arg('sort_by')::text = 'price_asc' THEN price END ASC,
  CASE WHEN sqlc.arg('sort_by')::text = 'price_desc' THEN price END DESC,
  CASE WHEN sqlc.arg('sort_by')::text = 'hot' THEN view_count END DESC,
  CASE WHEN sqlc.arg('sort_by')::text NOT IN ('price_asc', 'price_desc', 'hot') THEN created_at END DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: ListProductImages :many
SELECT id, product_id, image_url, sort_order, created_at
FROM product_image
WHERE product_id = ANY(sqlc.arg('product_ids')::bigint[])
ORDER BY sort_order ASC;

-- name: InsertProductImage :exec
INSERT INTO product_image (product_id, image_url, sort_order)
VALUES ($1, $2, $3);

-- name: DeleteProductImages :exec
DELETE FROM product_image WHERE product_id = $1;

-- name: ListUsersByIDs :many
SELECT id, name, campus, student_no FROM "user" WHERE id = ANY(sqlc.arg('ids')::bigint[]);

-- name: ListCategoriesByIDs :many
SELECT id, name FROM category WHERE id = ANY(sqlc.arg('ids')::bigint[]);

-- name: ListFavoritedProductIDs :many
SELECT product_id FROM favorite
WHERE user_id = sqlc.arg('user_id') AND product_id = ANY(sqlc.arg('product_ids')::bigint[]);

-- name: ListProductComments :many
SELECT id, product_id, user_id, content, created_at
FROM product_comment
WHERE product_id = $1
ORDER BY created_at ASC, id ASC;

-- name: InsertProductComment :one
INSERT INTO product_comment (product_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id, product_id, user_id, content, created_at;
