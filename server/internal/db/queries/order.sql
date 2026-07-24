-- name: GetOrder :one
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order WHERE id = $1;

-- name: InsertOrder :one
INSERT INTO trade_order (product_id, buyer_id, seller_id, status, meet_location, remark)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at;

-- name: ListOrdersByBuyer :many
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order WHERE buyer_id = $1
ORDER BY created_at DESC, id DESC;

-- name: ListOrdersBySeller :many
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order WHERE seller_id = $1
ORDER BY created_at DESC, id DESC;

-- name: ListOrdersByUser :many
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order WHERE buyer_id = $1 OR seller_id = $1
ORDER BY created_at DESC, id DESC;

-- name: CountActiveOrdersByBuyerProduct :one
SELECT count(*) FROM trade_order
WHERE product_id = $1 AND buyer_id = $2 AND status IN ('PENDING', 'RESERVED');

-- name: CountReservedOrdersOtherThan :one
SELECT count(*) FROM trade_order
WHERE product_id = $1 AND id <> $2 AND status = 'RESERVED';

-- name: CountActiveOrdersOtherThan :one
SELECT count(*) FROM trade_order
WHERE product_id = $1 AND id <> $2 AND status IN ('PENDING', 'RESERVED');

-- name: ListPendingOrdersOtherThan :many
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order
WHERE product_id = $1 AND id <> $2 AND status = 'PENDING';

-- name: ListActiveOrdersOtherThan :many
SELECT id, product_id, buyer_id, seller_id, status, meet_location, remark, created_at, updated_at, completed_at
FROM trade_order
WHERE product_id = $1 AND id <> $2 AND status IN ('PENDING', 'RESERVED');

-- name: SetOrderStatus :exec
UPDATE trade_order SET status = $2 WHERE id = $1;

-- name: SetOrderCompleted :exec
UPDATE trade_order SET status = 'COMPLETED', completed_at = now() WHERE id = $1;
