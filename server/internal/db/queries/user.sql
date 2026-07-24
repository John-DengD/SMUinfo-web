-- name: GetUserByStudentNo :one
SELECT * FROM "user" WHERE student_no = $1;

-- name: GetUserByID :one
SELECT * FROM "user" WHERE id = $1;

-- name: CountByStudentNo :one
SELECT count(*) FROM "user" WHERE student_no = $1;

-- name: InsertUser :one
INSERT INTO "user" (name, student_no, password_hash, phone, college, campus, role, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING *;

-- name: UpdateUserProfile :one
UPDATE "user" SET name=$2, phone=$3, college=$4, campus=$5, avatar=$6 WHERE id=$1 RETURNING *;
