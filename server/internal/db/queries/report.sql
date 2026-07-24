-- name: InsertReport :one
INSERT INTO report (reporter_id, product_id, reason, status)
VALUES ($1, $2, $3, $4) RETURNING id, reporter_id, product_id, reason, status, admin_remark, created_at, updated_at;

-- name: GetProductByID :one
SELECT id, seller_id, category_id, title, description, price, original_price, condition_level, trade_location, status, view_count, created_at, updated_at
FROM product WHERE id = $1;
