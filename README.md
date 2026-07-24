# 校园二手交易平台

按照 `campus-secondhand-platform-plan.md` 的方案实现的 MVP 版本：注册/登录、商品发布与浏览、搜索、收藏、站内消息、线下交易流程、举报、管理后台。

## 维护约定

后续产品功能只维护微信小程序端，Web 端不再作为业务功能维护入口。新增或调整管理员能力时，也优先在小程序内实现管理员入口，例如公告发布、商品审核、用户管理等。

平台不接入任何线上支付，所有交易在线下完成，平台仅做信息撮合。

## 技术栈

- 后端：Go + Gin
- 数据库：PostgreSQL
- ORM / 查询：sqlc + golang-migrate
- 鉴权：JWT + bcrypt
- 图片：本地文件上传，DB 只存 `/uploads/...` 路径

## 目录结构

```
smu-deal/
├── server/                  Go 后端（Gin + PostgreSQL）
│   ├── cmd/smudeal/         程序入口
│   ├── internal/            各业务域
│   └── internal/db/migrations/  自动迁移 SQL 文件
├── deploy/smu-deal.service  systemd 服务单元模板
├── sql/server-setup.md      服务器 PostgreSQL 初始化说明
└── campus-secondhand-platform-plan.md
```

## 本地启动

需要本地 PostgreSQL，先创建数据库和角色：

```bash
sudo -u postgres psql
CREATE ROLE smu_deal LOGIN PASSWORD 'dev_password';
CREATE DATABASE smu_deal OWNER smu_deal;
\q
```

然后启动后端：

```bash
cd server
cp .env.example .env   # 按需编辑 DB_URL、JWT_SECRET 等
make run               # 或：go run ./cmd/smudeal
```

启动时会自动应用 `internal/db/migrations/` 下的所有迁移（幂等，已执行过的跳过）。

服务监听 `http://localhost:8080`，健康检查：`GET /healthz`。

### 默认账号

| 角色   | 学号        | 密码     |
| ------ | ----------- | -------- |
| 管理员 | admin       | admin123 |
| 学生   | student001  | 123456   |

### 环境变量

| 变量                    | 默认值                        | 说明                                      |
| ----------------------- | ----------------------------- | ----------------------------------------- |
| `PORT`                  | `8080`                        | HTTP 监听端口                             |
| `DB_URL`                | —                             | PostgreSQL DSN，例如 `postgres://user:pass@localhost:5432/smu_deal?sslmode=disable` |
| `JWT_SECRET`            | —                             | 至少 32 字节的随机字符串                  |
| `UPLOAD_DIR`            | `./uploads`                   | 图片落地目录                              |
| `CORS_ALLOWED_ORIGINS`  | —                             | 允许的跨域来源，例如 `https://your-domain` |
| `MAX_FILE_SIZE`         | `5242880`（5 MB）             | 单张图片最大字节数                        |
| `MIGRATIONS_DIR`        | `internal/db/migrations`      | SQL 迁移文件目录                          |

## 自动部署

仓库已配置 GitHub Actions：push 到 `main` 分支后，CI 会交叉编译 Linux 静态二进制，通过 SSH 上传到服务器，替换旧二进制，并重启 `smu-deal.service`。服务重启后，二进制自动把新迁移应用到 PostgreSQL，无需在服务器上安装任何数据库工具。

### 服务器前置条件

1. 安装 PostgreSQL，创建 `smu_deal` 角色和数据库（见 `sql/server-setup.md`）。
2. 将 `deploy/smu-deal.service` 复制到 `/etc/systemd/system/`，编辑 `DB_URL`、`JWT_SECRET`、`CORS_ALLOWED_ORIGINS`，然后 `systemctl daemon-reload && systemctl enable --now smu-deal.service`。
3. 在 GitHub 仓库的 `Settings → Secrets and variables → Actions` 中配置以下 secret：

| Secret           | 必填 | 说明                                        |
| ---------------- | ---- | ------------------------------------------- |
| `DEPLOY_HOST`    | 是   | 服务器 IP 或域名                            |
| `DEPLOY_SSH_KEY` | 是   | 可登录服务器的 SSH 私钥                     |
| `DEPLOY_USER`    | 否   | SSH 用户，默认 `root`                       |
| `DEPLOY_PORT`    | 否   | SSH 端口，默认 `22`                         |
| `APP_DIR`        | 否   | 服务器应用目录，默认 `/opt/smu-deal`        |
| `SERVICE_NAME`   | 否   | systemd 服务名，默认 `smu-deal.service`     |

### 关于图片存储

商品图片不存入 PostgreSQL，而是落地到 `UPLOAD_DIR`（生产建议挂数据盘到 `/data/uploads`），DB 只保留 `/uploads/2026-05-28/xxx.jpg` 格式的路径。后期可平滑替换为 MinIO / OSS。

## 功能清单

普通用户：
- 注册、登录（JWT 鉴权）
- 浏览首页 / 分类筛选 / 多条件搜索 / 排序
- 商品详情、收藏 / 取消收藏
- 发布闲置、上传图片、编辑、上架 / 下架
- 联系卖家（站内消息）、未读消息提示
- 创建线下交易请求（我想要 → 待确认 → 已预约 → 已完成）
- 举报违规商品
- 我的资料 / 我的发布 / 我的收藏 / 我的交易 / 消息中心

管理员：
- 用户管理（启用 / 禁用）
- 商品管理（搜索、强制下架 / 恢复）
- 分类管理（增删改）
- 举报管理（驳回 / 下架处理）

## 后续可扩展

- 真实校园身份认证、邮箱/手机验证码
- WebSocket 实时聊天
- 商品推荐 / 浏览历史
- 卖家信用、交易评价
- 阿里云 OSS / MinIO 图片存储
