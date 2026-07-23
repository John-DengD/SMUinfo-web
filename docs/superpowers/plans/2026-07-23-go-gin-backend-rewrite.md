# SMU Deal Go(Gin) 后端重写 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 Go(Gin)+PostgreSQL 重写 SMU Deal 后端，作为现有 Spring Boot/MySQL API 的逐字节兼容替代品，微信小程序零改动即可切换。

**Architecture:** 分域垂直切分（auth/product/category/favorite/order/message/report/feedback/announcement/lostfound/transit/upload/admin）。每域 `handler.go`(Gin+DTO) → `service.go`(业务规则) → sqlc 生成查询。pgxpool 连接 PostgreSQL；golang-migrate 管理 schema；对外统一 `{code,message,data}` 信封 + camelCase 字段。

**Tech Stack:** Go 1.22 / Gin / pgx v5 / sqlc / golang-migrate / golang-jwt v5 / x/crypto/bcrypt。

**权威契约来源（关键）：** 这是一次 1:1 端口，不是新设计。每个域的对外 HTTP 契约（路由、请求/响应 JSON 字段名、状态流转、校验规则、错误码/文案）以 `backend/src/main/java/com/smu/deal/` 下对应的 `controller` + `service` + `dto` 为**唯一权威**。实现每个域时必须先读这三份 Java 文件，逐字段转写；不得凭空设计字段名或行为。JSON 字段名一律 camelCase（对齐现有 Jackson 输出）。

**兼容性验收门（每个域完成后必查）：**
1. 路由前缀/方法与 Java `@RequestMapping`/`@GetMapping` 等完全一致。
2. 响应信封 `{"code":0,"message":"ok","data":...}`，业务失败 `code!=0` 且 HTTP 200（对齐 `GlobalExceptionHandler`：BusinessException→200+code，鉴权失败→401，越权→403）。
3. 分页 `{"total":N,"records":[...]}`。
4. 响应 JSON 字段名逐个对照 Java DTO 的字段（camelCase）。
5. 公开/受保护划分对齐 `SecurityConfig`（见 Task 7）。

---

## 路由保护映射（来自 SecurityConfig，Task 7 据此实现）

- `OPTIONS /**` → 放行
- 公开：`/api/auth/**`、`GET /api/categories`、`/api/transit/**`、`GET /api/announcements/active`、`/uploads/**`、`GET /api/products`、`GET /api/products/*`、`GET /api/products/*/comments`、`GET /api/lost-found`、`GET /api/lost-found/*`
- `/api/admin/**` → 需 `role=ADMIN`
- 其余所有 → 需登录

## 文件结构

```
server/
├── go.mod  go.sum  sqlc.yaml  Makefile  .env.example
├── cmd/smudeal/main.go
└── internal/
    ├── config/config.go
    ├── httpx/{envelope.go, errors.go, middleware.go, context.go}
    ├── db/
    │   ├── pool.go
    │   ├── migrate.go
    │   ├── migrations/{0001_init.up.sql, 0001_init.down.sql, 0002_seed.up.sql, 0002_seed.down.sql}
    │   ├── queries/{user.sql, product.sql, category.sql, ...}
    │   └── gen/            (sqlc 输出)
    ├── auth/{handler.go, service.go, validate.go, service_test.go}
    ├── product/{handler.go, service.go, service_test.go}
    ├── category/{handler.go, service.go}
    ├── favorite/{handler.go, service.go}
    ├── order/{handler.go, service.go}
    ├── message/{handler.go, service.go}
    ├── report/{handler.go, service.go}
    ├── feedback/{handler.go, service.go}
    ├── announcement/{handler.go, service.go}
    ├── lostfound/{handler.go, service.go}
    ├── transit/{handler.go, service.go}
    ├── upload/{handler.go, service.go}
    └── admin/{handler.go}
```

---

## Task 0: Go module 与工具脚手架

**Files:**
- Create: `server/go.mod`, `server/Makefile`, `server/sqlc.yaml`, `server/.env.example`, `server/.gitignore`

- [ ] **Step 1: 初始化 module**

```bash
cd /Users/new/sch-projects/smu-deal
mkdir -p server && cd server
go mod init github.com/John-DengD/smu-deal/server
go get github.com/gin-gonic/gin@latest
go get github.com/jackc/pgx/v5@latest
go get github.com/golang-jwt/jwt/v5@latest
go get golang.org/x/crypto/bcrypt
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/golang-migrate/migrate/v4/database/pgx/v5@latest
go get github.com/golang-migrate/migrate/v4/source/file@latest
```

- [ ] **Step 2: 写 sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "internal/db/migrations"
    queries: "internal/db/queries"
    gen:
      go:
        package: "gen"
        out: "internal/db/gen"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_pointers_for_null_types: true
```

- [ ] **Step 3: 写 Makefile**

```makefile
.PHONY: sqlc build test run
sqlc:
	sqlc generate
build:
	CGO_ENABLED=0 go build -o bin/smudeal ./cmd/smudeal
test:
	go test ./...
run:
	go run ./cmd/smudeal
```

- [ ] **Step 4: 写 .env.example**

```
PORT=8080
DB_URL=postgres://smu_deal:password@localhost:5432/smu_deal?sslmode=disable
JWT_SECRET=local-development-only-secret-key-change-before-deploy-123456
UPLOAD_DIR=./uploads
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
MAX_FILE_SIZE=5242880
```

- [ ] **Step 5: 写 .gitignore**（`bin/`、`uploads/`、`.env`），提交

```bash
git add server && git commit -m "chore(server): init go module and tooling"
```

---

## Task 1: 配置加载

**Files:** Create `server/internal/config/config.go`, `server/internal/config/config_test.go`

- [ ] **Step 1: 写失败测试**

```go
package config

import ("os"; "testing")

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("DB_URL", "postgres://x")
	os.Setenv("JWT_SECRET", "s")
	c := Load()
	if c.Port != "8080" { t.Fatalf("want 8080 got %s", c.Port) }
	if c.UploadDir != "./uploads" { t.Fatalf("upload dir default") }
	if c.MaxFileSize != 5242880 { t.Fatalf("max file size default") }
}
```

- [ ] **Step 2: 运行确认失败** `cd server && go test ./internal/config/` → 编译失败（Load 未定义）

- [ ] **Step 3: 实现**

```go
package config

import ("os"; "strconv"; "strings")

type Config struct {
	Port           string
	DBURL          string
	JWTSecret      string
	JWTExpireHours int
	UploadDir      string
	URLPrefix      string
	AllowedOrigins []string
	MaxFileSize    int64
}

func Load() Config {
	return Config{
		Port:           env("PORT", "8080"),
		DBURL:          os.Getenv("DB_URL"),
		JWTSecret:      env("JWT_SECRET", "local-development-only-secret-key-change-before-deploy-123456"),
		JWTExpireHours: 168,
		UploadDir:      env("UPLOAD_DIR", "./uploads"),
		URLPrefix:      "/uploads",
		AllowedOrigins: strings.Split(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ","),
		MaxFileSize:    int64(envInt("MAX_FILE_SIZE", 5242880)),
	}
}

func env(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }
func envInt(k string, def int) int { if v := os.Getenv(k); v != "" { if n, err := strconv.Atoi(v); err == nil { return n } }; return def }
```

- [ ] **Step 4: 运行确认通过** `go test ./internal/config/` → PASS
- [ ] **Step 5: 提交** `git add server/internal/config && git commit -m "feat(server): config loader"`

---

## Task 2: PostgreSQL schema migration（0001_init）

**Files:** Create `server/internal/db/migrations/0001_init.up.sql` + `.down.sql`

移植全部 14 张表 + 索引（对照 `sql/init.sql`、`sql/announcement.sql`、`sql/lost-found.sql`、`sql/product-comment.sql`、`sql/performance-indexes.sql`）。MySQL→PG 转换规则见 spec。

- [ ] **Step 1: 写 `0001_init.up.sql`**（完整）

```sql
-- updated_at 触发器函数
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN NEW.updated_at = now(); RETURN NEW; END; $$ LANGUAGE plpgsql;

CREATE TABLE "user" (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    student_no VARCHAR(32) NOT NULL UNIQUE,
    password_hash VARCHAR(128) NOT NULL,
    phone VARCHAR(32), college VARCHAR(64), campus VARCHAR(64), avatar VARCHAR(256),
    role VARCHAR(16) NOT NULL DEFAULT 'USER',
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_user_upd BEFORE UPDATE ON "user" FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(64) NOT NULL, icon VARCHAR(128),
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_category_upd BEFORE UPDATE ON category FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE product (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seller_id BIGINT NOT NULL, category_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL, description TEXT,
    price NUMERIC(10,2) NOT NULL, original_price NUMERIC(10,2),
    condition_level VARCHAR(32), trade_location VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'ON_SALE',
    view_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_product_upd BEFORE UPDATE ON product FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_product_seller ON product(seller_id);
CREATE INDEX idx_product_category ON product(category_id);
CREATE INDEX idx_product_status ON product(status);

CREATE TABLE product_image (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL, image_url VARCHAR(256) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_product_image_product ON product_image(product_id);

CREATE TABLE product_comment (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL, user_id BIGINT NOT NULL,
    content VARCHAR(300) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_pc_product_created ON product_comment(product_id, created_at, id);
CREATE INDEX idx_pc_user_created ON product_comment(user_id, created_at, id);

CREATE TABLE favorite (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL, product_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, product_id)
);

CREATE TABLE message (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sender_id BIGINT NOT NULL, receiver_id BIGINT NOT NULL, product_id BIGINT,
    content VARCHAR(1024) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_message_receiver ON message(receiver_id);
CREATE INDEX idx_message_pair ON message(sender_id, receiver_id);

CREATE TABLE trade_order (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL, buyer_id BIGINT NOT NULL, seller_id BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    meet_location VARCHAR(128), remark VARCHAR(256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
CREATE TRIGGER trg_order_upd BEFORE UPDATE ON trade_order FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_order_buyer ON trade_order(buyer_id);
CREATE INDEX idx_order_seller ON trade_order(seller_id);
CREATE INDEX idx_order_product ON trade_order(product_id);

CREATE TABLE report (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reporter_id BIGINT NOT NULL, product_id BIGINT NOT NULL,
    reason VARCHAR(512) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    admin_remark VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_report_upd BEFORE UPDATE ON report FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE feedback (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT,
    category VARCHAR(32) NOT NULL DEFAULT '其他',
    content VARCHAR(2000) NOT NULL, contact VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    admin_reply VARCHAR(1000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_feedback_upd BEFORE UPDATE ON feedback FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_feedback_user ON feedback(user_id);
CREATE INDEX idx_feedback_status ON feedback(status);

CREATE TABLE announcement (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title VARCHAR(80) NOT NULL, content VARCHAR(500) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_announcement_upd BEFORE UPDATE ON announcement FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_announcement_status_created ON announcement(status, created_at, id);

CREATE TABLE lost_found (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL, type VARCHAR(16) NOT NULL,
    title VARCHAR(80) NOT NULL, description VARCHAR(1000) NOT NULL,
    location VARCHAR(128), contact VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'OPEN',
    view_count INT NOT NULL DEFAULT 0, event_time TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_lf_upd BEFORE UPDATE ON lost_found FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE INDEX idx_lf_type_status_created ON lost_found(type, status, created_at, id);
CREATE INDEX idx_lf_user_created ON lost_found(user_id, created_at, id);

CREATE TABLE lost_found_image (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lost_found_id BIGINT NOT NULL, image_url VARCHAR(256) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_lfi_item_sort ON lost_found_image(lost_found_id, sort_order, id);

CREATE TABLE transit_departure (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    line_code VARCHAR(32), line_name VARCHAR(64),
    station_code VARCHAR(32), station_name VARCHAR(64),
    direction_code VARCHAR(32), direction_name VARCHAR(64),
    schedule_type VARCHAR(32), schedule_type_name VARCHAR(64),
    departure_time TIME, service_type VARCHAR(32), service_label VARCHAR(64),
    sort_order INT, status VARCHAR(16) DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_transit_upd BEFORE UPDATE ON transit_departure FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

> 实现者注意：`transit_departure` 各列的确切类型/可空性需对照 `entity/TransitDeparture.java` 与 `config/TransitDataInitializer.java` 复核后再定稿。

- [ ] **Step 2: 写 `0001_init.down.sql`**：按逆序 `DROP TABLE IF EXISTS ... CASCADE;` 全部 14 张表 + `DROP FUNCTION IF EXISTS set_updated_at();`
- [ ] **Step 3: 提交** `git add server/internal/db/migrations && git commit -m "feat(server): postgres schema migration"`

---

## Task 3: 种子数据 migration（0002_seed）

**Files:** Create `server/internal/db/migrations/0002_seed.up.sql` + `.down.sql`

替代 Java `DataInitializer` 与 `TransitDataInitializer`。

- [ ] **Step 1: 预生成 bcrypt 哈希**

```bash
cd server && cat > /tmp/hash.go <<'EOF'
package main
import ("fmt"; "golang.org/x/crypto/bcrypt")
func main(){ for _,p:=range []string{"admin123","123456"}{ h,_:=bcrypt.GenerateFromPassword([]byte(p),10); fmt.Printf("%s => %s\n",p,h) } }
EOF
go run /tmp/hash.go
```

记录输出的两个 `$2a$10$...` 哈希，填入下一步。

- [ ] **Step 2: 写 `0002_seed.up.sql`**（把 `<HASH_ADMIN>`/`<HASH_STUDENT>` 换成上一步真实哈希）

```sql
INSERT INTO category (name, sort_order) VALUES
('教材资料',1),('电子数码',2),('宿舍用品',3),('服装鞋包',4),('运动户外',5),
('美妆个护',6),('交通工具',7),('票券卡券',8),('其他闲置',9);

INSERT INTO "user" (name, student_no, password_hash, role, status, college, campus) VALUES
('系统管理员','admin','<HASH_ADMIN>','ADMIN','ACTIVE',NULL,NULL),
('张同学','student001','<HASH_STUDENT>','USER','ACTIVE','计算机学院','主校区');
```

> transit 班车种子数据：对照 `config/TransitDataInitializer.java` 逐行转写成 `INSERT INTO transit_departure (...)`。若该初始化器数据量大，可在实现时用脚本从 Java 数组转 SQL。

- [ ] **Step 3: 写 `0002_seed.down.sql`**：`DELETE FROM category; DELETE FROM "user" WHERE student_no IN ('admin','student001'); DELETE FROM transit_departure;`
- [ ] **Step 4: 提交** `git commit -am "feat(server): seed data migration"`

---

## Task 4: DB 连接池与迁移执行

**Files:** Create `server/internal/db/pool.go`, `server/internal/db/migrate.go`

- [ ] **Step 1: 写 `pool.go`**

```go
package db

import ("context"; "github.com/jackc/pgx/v5/pgxpool")

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}
```

- [ ] **Step 2: 写 `migrate.go`**（用 golang-migrate 的 iofs/file source，程序启动时执行 up）

```go
package db

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn, dir string) error {
	m, err := migrate.New("file://"+dir, dsn)
	if err != nil { return err }
	if err := m.Up(); err != nil && err != migrate.ErrNoChange { return err }
	return nil
}
```

- [ ] **Step 3: 提交** `git commit -am "feat(server): db pool and migration runner"`

---

## Task 5: HTTP 信封、错误、上下文

**Files:** Create `server/internal/httpx/{envelope.go, errors.go, context.go}`, `envelope_test.go`

- [ ] **Step 1: 写失败测试 `envelope_test.go`**

```go
package httpx
import ("encoding/json"; "testing")
func TestOKEnvelope(t *testing.T){
	b,_ := json.Marshal(OK(map[string]int{"a":1}))
	if string(b) != `{"code":0,"message":"ok","data":{"a":1}}` { t.Fatalf("got %s", b) }
}
func TestFailEnvelope(t *testing.T){
	b,_ := json.Marshal(Fail(400,"bad"))
	if string(b) != `{"code":400,"message":"bad","data":null}` { t.Fatalf("got %s", b) }
}
```

- [ ] **Step 2: 运行确认失败** `go test ./internal/httpx/`
- [ ] **Step 3: 实现 `envelope.go`**

```go
package httpx

type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
func OK(data any) R   { return R{Code: 0, Message: "ok", Data: data} }
func Fail(code int, msg string) R { return R{Code: code, Message: msg, Data: nil} }

type Page struct {
	Total   int64 `json:"total"`
	Records any   `json:"records"`
}
```

- [ ] **Step 4: 实现 `errors.go`**（BizError + gin 中间件，对齐 GlobalExceptionHandler：BizError→200+code；panic→200+500）

```go
package httpx

import ("net/http"; "github.com/gin-gonic/gin")

type BizError struct { Code int; Msg string }
func (e BizError) Error() string { return e.Msg }
func NewBiz(code int, msg string) BizError { return BizError{code, msg} }
func Biz(msg string) BizError { return BizError{400, msg} }

// Abort 用于 handler 内主动返回业务错误
func Abort(c *gin.Context, e BizError) { c.JSON(http.StatusOK, Fail(e.Code, e.Msg)); c.Abort() }

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				if be, ok := r.(BizError); ok {
					c.JSON(http.StatusOK, Fail(be.Code, be.Msg))
				} else {
					c.JSON(http.StatusOK, Fail(500, "服务器错误"))
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
```

- [ ] **Step 5: 实现 `context.go`**（从 gin.Context 取当前用户）

```go
package httpx

import "github.com/gin-gonic/gin"

const CtxUserID = "uid"
const CtxRole = "role"

func UserID(c *gin.Context) int64 { v, _ := c.Get(CtxUserID); id, _ := v.(int64); return id }
func Role(c *gin.Context) string  { v, _ := c.Get(CtxRole); s, _ := v.(string); return s }
func RequireUserID(c *gin.Context) int64 {
	id := UserID(c)
	if id == 0 { panic(NewBiz(401, "未登录")) }
	return id
}
```

- [ ] **Step 6: 运行确认通过** `go test ./internal/httpx/`
- [ ] **Step 7: 提交** `git commit -am "feat(server): http envelope, biz errors, context helpers"`

---

## Task 6: JWT 与鉴权中间件

**Files:** Create `server/internal/httpx/jwt.go`, `server/internal/httpx/middleware.go`, `jwt_test.go`

对照 `security/JwtUtil.java`（HMAC，`sub=userId`，`role` claim，168h）与 `JwtAuthenticationFilter.java`（Bearer 解析，解析失败静默放行→后续 auth 拦截）。

- [ ] **Step 1: 写失败测试 `jwt_test.go`**

```go
package httpx
import "testing"
func TestJWTRoundTrip(t *testing.T){
	m := NewJWT("test-secret-32-bytes-xxxxxxxxxxxxxx", 168)
	tok, err := m.Generate(42, "ADMIN")
	if err != nil { t.Fatal(err) }
	uid, role, err := m.Parse(tok)
	if err != nil || uid != 42 || role != "ADMIN" { t.Fatalf("uid=%d role=%s err=%v", uid, role, err) }
}
```

- [ ] **Step 2: 运行确认失败**
- [ ] **Step 3: 实现 `jwt.go`**（用 golang-jwt/v5，HS256）

```go
package httpx

import ("strconv"; "time"; "github.com/golang-jwt/jwt/v5")

type JWT struct { secret []byte; expH int }
func NewJWT(secret string, expH int) JWT { return JWT{[]byte(secret), expH} }

func (j JWT) Generate(uid int64, role string) (string, error) {
	now := time.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.FormatInt(uid, 10), "role": role,
		"iat": now.Unix(), "exp": now.Add(time.Duration(j.expH) * time.Hour).Unix(),
	})
	return t.SignedString(j.secret)
}

func (j JWT) Parse(tok string) (int64, string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) { return j.secret, nil })
	if err != nil { return 0, "", err }
	sub, _ := claims["sub"].(string)
	uid, _ := strconv.ParseInt(sub, 10, 64)
	role, _ := claims["role"].(string)
	return uid, role, nil
}
```

- [ ] **Step 4: 实现 `middleware.go`**（AuthOptional 解析 Bearer 注入 context；RequireAuth 未登录→401 信封；RequireAdmin→403；CORS）

```go
package httpx

import ("net/http"; "strings"; "github.com/gin-gonic/gin")

func AuthParse(j JWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if strings.HasPrefix(h, "Bearer ") {
			if uid, role, err := j.Parse(h[7:]); err == nil {
				c.Set(CtxUserID, uid); c.Set(CtxRole, role)
			}
		}
		c.Next()
	}
}
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserID(c) == 0 { c.JSON(http.StatusUnauthorized, Fail(401, "登录已过期，请重新登录")); c.Abort(); return }
		c.Next()
	}
}
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if UserID(c) == 0 { c.JSON(http.StatusUnauthorized, Fail(401, "登录已过期，请重新登录")); c.Abort(); return }
		if !strings.EqualFold(Role(c), "ADMIN") { c.JSON(http.StatusForbidden, Fail(403, "无权访问")); c.Abort(); return }
		c.Next()
	}
}
func CORS(origins []string) gin.HandlerFunc { /* 反射 Origin 命中 origins 则回写头；OPTIONS 直接 204 */
	set := map[string]bool{}; for _, o := range origins { set[strings.TrimSpace(o)] = true }
	return func(c *gin.Context) {
		o := c.GetHeader("Origin")
		if set[o] { c.Header("Access-Control-Allow-Origin", o); c.Header("Access-Control-Allow-Credentials", "true") }
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")
		if c.Request.Method == http.MethodOptions { c.AbortWithStatus(http.StatusNoContent); return }
		c.Next()
	}
}
```

- [ ] **Step 5: 运行确认通过** `go test ./internal/httpx/`
- [ ] **Step 6: 提交** `git commit -am "feat(server): jwt and auth/cors middleware"`

---

## Task 7: main 装配 + 路由骨架 + 静态目录

**Files:** Create `server/cmd/smudeal/main.go`

此任务先搭起可启动的空骨架：加载 config → 迁移 → pgxpool → 注册全局中间件（CORS、AuthParse、Recovery）→ 静态 `/uploads` → 健康检查 `/healthz`。各域路由分组先建好、handler 在后续任务填充。路由分组与保护严格对照本文件顶部“路由保护映射”。

- [ ] **Step 1: 写 `main.go`**

```go
package main

import (
	"context"; "log/slog"; "os"
	"github.com/gin-gonic/gin"
	"github.com/John-DengD/smu-deal/server/internal/config"
	"github.com/John-DengD/smu-deal/server/internal/db"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

func main() {
	cfg := config.Load()
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" || true {
		mdir := httpxOr(os.Getenv("MIGRATIONS_DIR"), "internal/db/migrations")
		if err := db.RunMigrations(cfg.DBURL, mdir); err != nil { slog.Error("migrate", "err", err); os.Exit(1) }
	}
	pool, err := db.NewPool(context.Background(), cfg.DBURL)
	if err != nil { slog.Error("db", "err", err); os.Exit(1) }
	defer pool.Close()

	_ = os.MkdirAll(cfg.UploadDir, 0o755)
	jwt := httpx.NewJWT(cfg.JWTSecret, cfg.JWTExpireHours)

	r := gin.New()
	r.Use(httpx.CORS(cfg.AllowedOrigins), httpx.AuthParse(jwt), httpx.Recovery())
	r.Static(cfg.URLPrefix, cfg.UploadDir)
	r.GET("/healthz", func(c *gin.Context){ c.JSON(200, httpx.OK("ok")) })

	api := r.Group("/api")
	// 各域在后续任务注册：auth.Register(api, ...) 等
	_ = api; _ = pool; _ = jwt

	if err := r.Run(":" + cfg.Port); err != nil { slog.Error("run", "err", err); os.Exit(1) }
}

func httpxOr(v, def string) string { if v != "" { return v }; return def }
```

- [ ] **Step 2: 生成 sqlc（先建至少一个 query 以免空目录报错，见 Task 8）后** `go build ./...`
- [ ] **Step 3: 提交** `git commit -am "feat(server): main wiring, static uploads, health"`

> 说明：随着后续域任务完成，逐个在 `main.go` 的 api 分组下注册路由。每个域任务的最后一步都包含“在 main.go 注册本域路由并 go build”。

---

## Task 8: auth 域（注册/登录/我的资料）— 完整实现（作为所有域的模板）

**权威契约：** `controller/AuthController.java`、`service/AuthService.java`（已读，见下）、`dto/AuthDTO.java`。
**路由：** `POST /api/auth/register`、`POST /api/auth/login`、`GET /api/users/me`(需登录)、`PUT /api/users/me`(需登录)。

**Files:** Create `server/internal/db/queries/user.sql`, `server/internal/auth/{service.go, validate.go, handler.go, service_test.go}`；Modify `cmd/smudeal/main.go`

- [ ] **Step 1: 写 `queries/user.sql`（sqlc）**

```sql
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
```

运行 `cd server && sqlc generate`，确认 `internal/db/gen` 生成。

- [ ] **Step 2: 写失败测试 `service_test.go`**（复刻 AuthService 校验：学号 12 位纯数字、假学号黑名单、姓名规则）

```go
package auth
import "testing"
func TestValidateStudentNo(t *testing.T){
	if _, err := validateStudentNo("12345"); err == nil { t.Fatal("short should fail") }
	if _, err := validateStudentNo("123456789012"); err == nil { t.Fatal("fake blocklist should fail") }
	if v, err := validateStudentNo(" 202312345678 "); err != nil || v != "202312345678" { t.Fatalf("trim/ok got %q %v", v, err) }
}
func TestValidateName(t *testing.T){
	if _, err := validateName("张"); err == nil { t.Fatal("too short") }
	if _, err := validateName("张3"); err == nil { t.Fatal("digit not allowed") }
	if v, err := validateName("  张 三  "); err != nil || v != "张 三" { t.Fatalf("got %q %v", v, err) }
}
```

- [ ] **Step 3: 运行确认失败** `go test ./internal/auth/`
- [ ] **Step 4: 实现 `validate.go`**（正则：学号 `^\d{12}$`；姓名 `^[\p{Han}A-Za-z·.\- ]+$` 且 2–20；黑名单 `000000000000/111111111111/123456789012`；姓名先 `strings.Fields`+`join(" ")` 归一空白）
- [ ] **Step 5: 实现 `service.go`**（Register/Login/GetMe/UpdateMe，bcrypt 校验，禁用账号拦截，签发 JWT。错误用 `httpx.Biz`/`httpx.NewBiz`）
- [ ] **Step 6: 实现 `handler.go`**（Gin 处理器 + 请求/响应 DTO，字段名对照 `AuthDTO`：如 `studentNo/name/password/phone/college/campus`，登录响应 `{token, user:{id,name,studentNo,phone,college,campus,avatar,role,status}}`；`Register(rg *gin.RouterGroup, svc *Service)` 注册子路由）
- [ ] **Step 7: 运行确认通过** `go test ./internal/auth/`
- [ ] **Step 8: 在 `main.go` 注册**：构造 `auth.NewService(gen.New(pool), jwt)` 与 `auth.Register(api, svc)`，`go build ./...`
- [ ] **Step 9: 提交** `git commit -am "feat(server): auth domain (register/login/profile)"`

---

## Task 9–19: 其余业务域（同 Task 8 模板逐域实现）

每个域重复 Task 8 的步骤序列：`queries/*.sql` → sqlc generate → service 单测（若原 Java 有对应 `*ServiceTest` 则复刻其用例）→ service 实现 → handler+DTO 实现 → main 注册 → build → 提交。每域完成后跑“兼容性验收门”5 项自查。

- [ ] **Task 9: category** — 权威：`CategoryController/CategoryService`。路由 `GET /api/categories`（公开，返回 ACTIVE 分类按 sort_order）。
- [ ] **Task 10: product** — 权威：`ProductController/ProductService/ProductDTO`。路由 `GET /api/products`（多条件搜索/分类/排序/分页，公开）、`GET /api/products/{id}`（公开，view_count++）、`POST /api/products`、`PUT /api/products/{id}`、`DELETE /api/products/{id}`（需登录且属主）、`GET /api/products/{id}/comments`（公开）、`POST /api/products/{id}/comments`（需登录）。含 product_image 关联写入。**这是最复杂域，实现前完整通读三份 Java。**
- [ ] **Task 11: favorite** — 权威：`FavoriteController/FavoriteService`。`GET /api/favorites`、`POST /api/favorites/{productId}`、`DELETE /api/favorites/{productId}`（均需登录；唯一约束去重）。
- [ ] **Task 12: order** — 权威：`OrderController/OrderService`。`POST /api/orders`、`GET /api/orders?role=buyer|seller`、`PUT /api/orders/{id}/confirm|finish|cancel`（需登录）。状态机 PENDING→CONFIRMED→...→COMPLETED/CANCELLED 严格对照 Java；finish 写 completed_at。
- [ ] **Task 13: message** — 权威：`MessageController/MessageService`。`POST /api/messages`、`GET /api/messages`（会话列表）、`GET /api/messages/conversation/{userId}`（拉取并标记已读）、`GET /api/messages/unread-count`（需登录）。
- [ ] **Task 14: report** — 权威：`ReportController/ReportService`。`POST /api/reports`（需登录）。
- [ ] **Task 15: feedback** — 权威：`FeedbackController/FeedbackService`。`POST /api/feedback`（user_id 可空/匿名）、`GET /api/feedback/mine`（需登录）。
- [ ] **Task 16: announcement** — 权威：`AnnouncementController/AnnouncementService`。`GET /api/announcements/active`（公开，ACTIVE 按 created_at desc）。
- [ ] **Task 17: lostfound** — 权威：`LostFoundController/LostFoundService`。`GET /api/lost-found`（公开，type/status 筛选分页）、`GET /api/lost-found/{id}`（公开，view_count++）、`POST /api/lost-found`、`DELETE /api/lost-found/{id}`（需登录且属主）。含 lost_found_image。
- [ ] **Task 18: transit** — 权威：`TransitController/TransitService/TransitDeparture`。`GET /api/transit/next`（公开，按当前时间/线路/方向返回下几班）。查询参数与响应字段严格对照 Java。
- [ ] **Task 19: upload** — 权威：`UploadController/UploadService`。`POST /api/upload/image`（需登录，multipart 表单字段名对照 Java；按 `UPLOAD_DIR/yyyy-MM-dd/<uuid>.<ext>` 落盘；返回 `{url:"/uploads/..."}`；校验 MAX_FILE_SIZE 与图片扩展名）。

---

## Task 20: admin 域（管理后台聚合）

**权威：** `controller/AdminController.java` + 各被聚合 Service（`AdminUserService` 等）。全部路由挂在 `/api/admin` 分组下并加 `RequireAdmin()`。

- [ ] **Step 1–N:** 逐路由实现（对照 AdminController 已读的 mapping）：
  - `GET /api/admin/users`、`PUT /api/admin/users/{id}/status`
  - `GET /api/admin/products`、`PUT /api/admin/products/{id}/status`（强制下架/恢复）
  - `GET/POST /api/admin/categories`、`PUT/DELETE /api/admin/categories/{id}`
  - `GET /api/admin/reports`、`PUT /api/admin/reports/{id}`（驳回/下架处理）
  - `GET /api/admin/feedback`、`PUT /api/admin/feedback/{id}`（回复）
  - `GET/POST /api/admin/announcements`、`PUT/DELETE /api/admin/announcements/{id}`
- [ ] **末步:** main 注册 admin 分组、build、`git commit -am "feat(server): admin domain"`

---

## Task 21: 端到端兼容性集成测试

**Files:** Create `server/internal/e2e/e2e_test.go`（用 testcontainers-go 起临时 PG）

- [ ] **Step 1:** 起容器 → 跑 migrations → 起 gin engine（httptest.Server）。
- [ ] **Step 2:** 覆盖核心链路并断言信封/字段名：注册→登录→拿 token→发布商品→列表搜索→详情→收藏→下单→confirm/finish→发消息→未读数→失物发布→公告 active→举报→反馈→管理员改用户状态。
- [ ] **Step 3:** 断言每个响应顶层为 `{"code":0,"message":"ok","data":...}`，分页为 `{"total","records"}`，关键字段名 camelCase（`studentNo/createdAt/viewCount` 等）。
- [ ] **Step 4:** `go test ./internal/e2e/` 全绿后提交。

---

## Task 22: 部署与文档切换

**Files:** Modify `.github/workflows/deploy.yml`；Create `deploy/smu-deal.service`；Modify `README.md`、`sql/server-setup.md`

- [ ] **Step 1:** 改 `deploy.yml`：移除 JDK/Maven；`actions/setup-go`；`cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o smudeal ./cmd/smudeal`；scp `server/smudeal` + `server/internal/db/migrations` 到服务器 `/tmp`；SSH：备份旧二进制→安装到 `$APP_DIR/smudeal`→拷贝 migrations 到 `$APP_DIR/migrations`→重启 `$SERVICE_NAME`→`curl -f localhost:8080/healthz` 健康检查。复用现有 Secrets。
- [ ] **Step 2:** 写 `deploy/smu-deal.service`（`ExecStart=$APP_DIR/smudeal`，`WorkingDirectory=$APP_DIR`，`Environment=MIGRATIONS_DIR=$APP_DIR/migrations` 及 DB_URL/JWT_SECRET/UPLOAD_DIR 等，`Restart=always`）。
- [ ] **Step 3:** 更新 `sql/server-setup.md`：改成 PostgreSQL 建库/建账号命令（`CREATE DATABASE smu_deal; CREATE USER ...; GRANT ...`），说明 CI 只跑 migration。
- [ ] **Step 4:** 更新 `README.md`：技术栈改为 Go(Gin)+PostgreSQL；启动步骤 `cd server && make run`；环境变量表更新（去掉 CLEANUP_*，DB_URL 改 PG DSN）。
- [ ] **Step 5:** 提交 `git commit -am "ci: deploy go binary; docs: postgres setup"`

---

## Task 23: 清理旧后端

- [ ] **Step 1:** 确认新后端 e2e 全绿、小程序冒烟通过后，`git rm -r backend/` 及 MySQL 专用 SQL（保留 spec/plan 记录）。
- [ ] **Step 2:** 提交 `git commit -m "chore: remove legacy java backend"`

> 注意：此任务在真实切换验证后再执行；执行计划时若尚未验证，保留旧 backend 并跳过本任务。

---

## Self-Review 结论

- **Spec 覆盖**：14 表→Task 2；种子/默认账号→Task 3；信封/分页→Task 5；JWT/bcrypt/CORS→Task 6；路由保护映射→Task 7 + 各域；全部端点→Task 8–20；兼容性验收→Task 21；部署/环境变量/去 CLEANUP→Task 22；去资源清理任务→全程不实现（YAGNI）。
- **端到端字段兼容**：每个域任务以 Java controller/service/dto 为权威契约并列入验收门，Task 21 断言字段名。
- **类型一致性**：`httpx.R/Page/BizError/JWT`、`httpx.RequireUserID/UserID/Role`、`config.Config` 字段在 Task 5/6/1 定义，后续域任务统一引用。
- **已知待实现者复核点**（非占位符，是端口的必然）：transit 表列定稿（Task 2 Step1 注记）、transit 种子转写（Task 3）、各域 DTO 字段以 Java 为准（Task 9–20）。
