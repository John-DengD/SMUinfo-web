# SMU Deal 后端 Go(Gin) 重写设计

- 日期：2026-07-23
- 作者：Junlong Deng（协同 Claude）
- 状态：已批准，进入实现

## 背景与动机

现有后端为 Spring Boot 3 + Spring Security + JWT + MyBatis Plus + MySQL 8（约 4100 行 Java，84 文件，14 张表，13 个业务域）。JVM 常驻内存与 MySQL 一起占用服务器资源过多。目标：用 Go(Gin) 重写后端，并把数据库并入服务器上已有的 PostgreSQL，显著降低资源占用。

微信小程序（`miniprogram/`）是唯一在线维护的客户端。已删除的 Vue Web 前端不恢复。

## 已确定的决策

1. **数据**：全新开始（fresh start）。PostgreSQL 建空表 + 种子数据，不迁移旧 MySQL 数据，不写迁移工具。
2. **功能范围**：复刻全部业务功能，**去掉**资源兜底清理任务（`ResourcePressureCleanupJob` 一族）。
3. **数据访问**：sqlc（写 SQL → 生成类型安全 Go 代码）。
4. **构建部署**：GitHub Actions 交叉编译 Linux 静态二进制，scp 到服务器；golang-migrate 跑建表；PG 的库/账号由人工在服务器预先创建。

## 最高优先级硬约束：小程序逐字节兼容

Go 后端必须是现有 API 的**逐字节兼容替代品**，小程序代码零改动即可切换：

- 相同路由前缀 `/api/...`、相同 HTTP 方法。
- 相同响应信封：`{"code":0,"message":"ok","data":...}`，`code=0` 成功、非 0 失败。
- 相同分页结构：`{"total":N,"records":[...]}`。
- **相同 JSON 字段名（camelCase）**：如 `studentNo`、`createdAt`、`viewCount`。每个对外响应结构体用显式 `json:"..."` tag 精确对齐现有 Jackson 输出。
- 相同鉴权头 `Authorization: Bearer <jwt>`；JWT 用同一 HMAC 密钥、`sub=userId`、`role` claim、168h 过期。
- 相同图片 URL 前缀 `/uploads/...`，由 Go 直接托管静态目录。

**验收**：小程序连新后端，全流程无改动跑通（注册/登录/发布/搜索/收藏/下单/消息/失物招领/公告/举报/反馈/班车/管理后台）。

## 技术栈

| 关注点 | 选型 |
|---|---|
| Web 框架 | Gin |
| DB 驱动 | pgx/v5 + pgxpool |
| 查询层 | sqlc |
| 迁移 | golang-migrate（`.up.sql`/`.down.sql`） |
| JWT | golang-jwt/v5 |
| 密码 | golang.org/x/crypto/bcrypt（兼容 Spring BCrypt `$2a$` 格式） |
| 校验 | go-playground/validator + 手写规则 |
| 配置 | 环境变量（标准库） |
| 日志 | log/slog |

## 项目结构

新后端放在仓库 `server/` 目录，与旧 `backend/` 并存，切换完成后再删除旧目录。

```
server/
├── cmd/smudeal/main.go        入口：装配依赖、启动 Gin
├── internal/
│   ├── config/                环境变量加载
│   ├── db/
│   │   ├── migrations/        golang-migrate SQL
│   │   ├── queries/           手写 .sql（sqlc 输入）
│   │   └── gen/               sqlc 生成代码（提交进仓库）
│   ├── httpx/                 R 信封、分页、错误中间件、JWT 中间件、CORS
│   ├── auth/ product/ category/ favorite/ order/ message/
│   ├── report/ feedback/ announcement/ lostfound/ transit/
│   ├── upload/                本地文件上传
│   └── admin/                 管理后台聚合
├── sqlc.yaml
├── go.mod
└── Makefile
```

每域内部分层：`handler.go`（Gin 处理器 + 请求/响应 DTO）→ `service.go`（业务规则）→ sqlc 查询。域间只经 service 接口交互。

## 数据层：MySQL → PostgreSQL

14 张表：`user`、`category`、`product`、`product_image`、`product_comment`、`favorite`、`message`、`trade_order`、`report`、`feedback`、`announcement`、`lost_found`、`lost_found_image`、`transit_departure`。

关键类型转换：

| MySQL | PostgreSQL |
|---|---|
| `BIGINT AUTO_INCREMENT` | `BIGINT GENERATED ALWAYS AS IDENTITY` |
| `DATETIME` + `ON UPDATE CURRENT_TIMESTAMP` | `TIMESTAMPTZ` + `updated_at` 触发器 |
| `TINYINT(1)`（is_read） | `BOOLEAN` |
| `DECIMAL(10,2)` | `NUMERIC(10,2)` |
| `KEY idx_x` | `CREATE INDEX` |
| `LocalTime`（transit departure_time） | `TIME` |

- 索引按现有 `sql/performance-indexes.sql` 一并移植。
- Migration 1：建全部表 + 索引 + `updated_at` 触发器函数。
- Migration 2：种子数据 —— 9 个初始分类、admin/student001 账号（预生成 bcrypt 哈希）、transit 班车数据。**替代原 `DataInitializer`/`TransitDataInitializer`**。
- sqlc 生成结构体仅内部使用；对外响应用各域自己的 DTO 保证 camelCase 精确可控。

## 鉴权与安全

- 逐条复刻 `AuthService`：学号 12 位纯数字、假学号黑名单（`000000000000`/`111111111111`/`123456789012`）、姓名 2–20 字符且字符集校验、bcrypt 校验、禁用账号拦截。
- JWT 中间件：解析 `Bearer`，注入 `userId`/`role` 到 context；管理接口叠加 `role=ADMIN` 校验。
- 逐个对照旧 `SecurityConfig`，复刻公开/受保护路由划分（如 `GET /api/products` 公开，写操作需登录）。
- CORS 沿用 `CORS_ALLOWED_ORIGINS`。

## 配置（环境变量）

- 保留：`DB_USER`、`DB_PASSWORD`、`JWT_SECRET`、`UPLOAD_DIR`、`CORS_ALLOWED_ORIGINS`、`MAX_FILE_SIZE`。
- 调整：`DB_URL` → PG DSN（`postgres://user:pass@host:5432/smu_deal?sslmode=disable`）；新增 `PORT`（默认 8080）。
- 去掉：所有 `CLEANUP_*`。

## 部署（GitHub Actions 交叉编译）

改写 `.github/workflows/deploy.yml`：

1. `setup-go` → `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build` 出静态二进制（去掉 JDK/Maven）。
2. scp 二进制 + `migrations/` 到服务器 `/tmp`。
3. SSH：备份旧二进制 → 部署到 `/opt/smu-deal/smudeal` → 跑 `migrate up` → 重启 systemd → 健康检查。
4. 提供新 `smu-deal.service`（`ExecStart` 指向二进制）+ 更新 `sql/server-setup.md`（改成 PG 建库/建账号命令，CI 只跑 migration，不碰建库权限）。

复用现有 GitHub Secrets（`DEPLOY_HOST`/`DEPLOY_SSH_KEY`/`DEPLOY_USER`/`DEPLOY_PORT`/`APP_DIR`/`SERVICE_NAME`）。

## 测试

- 每个 service 的业务规则写单元测试，复刻现有 `*ServiceTest` 覆盖点（auth 校验、lost-found、announcement 等）。
- 关键端点用 `httptest` 做集成测试，断言 JSON 信封与字段名精确匹配。集成测试用 Docker 化 PG（testcontainers-go，用户确认本机可用 Docker）。
- 交付前用真实小程序连本地新后端跑一轮冒烟。

## 明确不做（YAGNI）

- 资源兜底清理任务（`ResourcePressureCleanupJob` 等）。
- 不改微信小程序代码；不恢复已删除的 Vue 前端。
- 不引入 Docker（运行时）、Redis、消息队列等新组件。

## 端点清单（须逐一复刻，保持信封/字段兼容）

```
POST   /api/auth/register            GET    /api/lost-found
POST   /api/auth/login               GET    /api/lost-found/{id}
GET    /api/users/me                 POST   /api/lost-found
PUT    /api/users/me                 DELETE /api/lost-found/{id}
GET    /api/categories               GET    /api/favorites
GET    /api/products                 POST   /api/favorites/{productId}
GET    /api/products/{id}            DELETE /api/favorites/{productId}
POST   /api/products                 POST   /api/messages
PUT    /api/products/{id}            GET    /api/messages
DELETE /api/products/{id}            GET    /api/messages/conversation/{userId}
GET    /api/products/{id}/comments   GET    /api/messages/unread-count
POST   /api/products/{id}/comments   POST   /api/reports
POST   /api/upload/image             GET    /api/announcements/active
POST   /api/orders                   GET    /api/transit/next
GET    /api/orders?role=buyer|seller POST   /api/feedback
PUT    /api/orders/{id}/confirm      GET    /api/feedback/mine
PUT    /api/orders/{id}/finish
PUT    /api/orders/{id}/cancel

管理后台（需 role=ADMIN）：
GET/PUT   /api/admin/users, /api/admin/users/{id}/status
GET/PUT   /api/admin/products, /api/admin/products/{id}/status
GET/POST/PUT/DELETE /api/admin/categories[/{id}]
GET/PUT   /api/admin/reports[/{id}]
GET/PUT   /api/admin/feedback[/{id}]
GET/POST/PUT/DELETE /api/admin/announcements[/{id}]
```
