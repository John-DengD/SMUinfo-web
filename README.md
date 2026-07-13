# 校园二手交易平台

按照 `campus-secondhand-platform-plan.md` 的方案实现的 MVP 版本：注册/登录、商品发布与浏览、搜索、收藏、站内消息、线下交易流程、举报、管理后台。

## 维护约定

后续产品功能只维护微信小程序端，Web 端不再作为业务功能维护入口。新增或调整管理员能力时，也优先在小程序内实现管理员入口，例如公告发布、商品审核、用户管理等。

平台不接入任何线上支付，所有交易在线下完成，平台仅做信息撮合。

## 技术栈

- 前端：Vue 3 + Vite + Vue Router + Pinia + Axios + Element Plus
- 后端：Spring Boot 3 + Spring Security + JWT + MyBatis Plus
- 数据库：MySQL 8
- 图片：本地文件上传，存储到 `backend/uploads/`

## 目录结构

```
smu-deal/
├── backend/             Spring Boot 后端
├── frontend/            Vue 前端
├── sql/init.sql         数据库建表与初始化脚本
└── campus-secondhand-platform-plan.md
```

## 启动步骤

## 自动部署

仓库已配置 GitHub Actions：push 到 `main` 分支后会自动打包 `backend` 和 `frontend`，通过 SSH 上传到服务器，替换后端 JAR、发布前端静态文件，并重启 `smu-deal.service`、reload Nginx。

需要在 GitHub 仓库的 `Settings -> Secrets and variables -> Actions` 里配置：

| Secret | 必填 | 说明 |
| ------ | ---- | ---- |
| `DEPLOY_HOST` | 是 | 服务器 IP 或域名 |
| `DEPLOY_SSH_KEY` | 是 | 可登录服务器的 SSH 私钥 |
| `DEPLOY_USER` | 否 | SSH 用户，默认 `root` |
| `DEPLOY_PORT` | 否 | SSH 端口，默认 `22` |
| `APP_DIR` | 否 | 服务器应用目录，默认 `/opt/smu-deal` |
| `WEB_DIR` | 否 | 前端 Nginx 站点目录，默认 `/var/www/smu-deal` |
| `SERVICE_NAME` | 否 | systemd 服务名，默认 `smu-deal.service` |

服务器端需提前准备好 Java 17、数据库环境、Nginx 和 `smu-deal.service`。Actions 会把新后端包部署为 `/opt/smu-deal/smu-deal.jar`，并在替换前备份旧包；前端默认发布到 `/var/www/smu-deal`，发布前会备份原 `index.html` 所在目录内容。

### 1. 准备 MySQL

默认连接本机 `localhost:3306` 的 `smu_deal` 数据库，用户名为 `root`，密码为空。部署环境请通过环境变量提供数据库凭据和 JWT 密钥，不要把真实凭据写入仓库。

首次使用需要先导入数据库结构：

```bash
mysql -h localhost -uroot -p < sql/init.sql
```

连接远程或带密码的 MySQL 时，通过环境变量覆盖：
```bash
DB_URL='jdbc:mysql://<DB_HOST>:3306/smu_deal?useSSL=false&serverTimezone=Asia/Shanghai&characterEncoding=utf-8&allowPublicKeyRetrieval=true' \
DB_USER='<DB_USER>' \
DB_PASSWORD='<DB_PASSWORD>' \
JWT_SECRET='<AT_LEAST_32_BYTE_RANDOM_SECRET>' \
mvn spring-boot:run
```

### 关于图片存储

**为什么不直接把图片塞进 MySQL？** 二进制图片放进数据库会让 DB 体积暴涨、备份变慢、缓存命中率下降。业界标准做法是 **MySQL 只存图片 URL，图片本体放磁盘文件系统或 MinIO/OSS**。本项目已经这么做：上传图片接口把文件落地到 `uploads/` 目录，DB 里仅保留 `/uploads/2026-05-28/xxx.jpg` 这样的路径。

服务器 40G 硬盘按单图 500KB 估算可存约 8 万张图。生产环境只需把后端启动时的环境变量改成挂载盘：
```bash
UPLOAD_DIR=/data/uploads mvn spring-boot:run
```

### 资源兜底清理

后端内置资源兜底任务，默认每 5 分钟检查一次。磁盘可用率低于 15% 或 JVM 堆内存使用率高于 90% 时，会分批清理 7 天以前的已售出、已下架商品，并同步删除商品图片文件和相关收藏、留言、私信、订单、举报记录。

可通过环境变量调整：
```bash
CLEANUP_ENABLED=true
CLEANUP_MIN_DISK_FREE_RATIO=0.15
CLEANUP_MAX_HEAP_USED_RATIO=0.90
CLEANUP_BATCH_SIZE=100
CLEANUP_MIN_AGE_DAYS=7
CLEANUP_CHECK_INTERVAL_MS=300000
```


### 2. 启动后端

需要 JDK 17+ 和 Maven。

```bash
cd backend
mvn spring-boot:run
```

后端服务监听 `http://localhost:8080`。首次启动会自动创建以下账号：

| 角色   | 学号        | 密码     |
| ------ | ----------- | -------- |
| 管理员 | admin       | admin123 |
| 学生   | student001  | 123456   |

### 3. 启动前端

需要 Node.js 18+。

```bash
cd frontend
npm install
npm run dev
```

前端开发服务 `http://localhost:5173`，已在 `vite.config.js` 中代理 `/api` 与 `/uploads` 到后端。

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

## 主要接口

```
POST   /api/auth/register
POST   /api/auth/login
GET    /api/users/me
PUT    /api/users/me
GET    /api/categories
GET    /api/products
GET    /api/products/{id}
POST   /api/products
PUT    /api/products/{id}
DELETE /api/products/{id}
POST   /api/upload/image
GET    /api/lost-found
GET    /api/lost-found/{id}
POST   /api/lost-found
DELETE /api/lost-found/{id}
GET    /api/favorites
POST   /api/favorites/{productId}
DELETE /api/favorites/{productId}
POST   /api/messages
GET    /api/messages
GET    /api/messages/conversation/{userId}
GET    /api/messages/unread-count
POST   /api/orders
GET    /api/orders?role=buyer|seller
PUT    /api/orders/{id}/confirm
PUT    /api/orders/{id}/finish
PUT    /api/orders/{id}/cancel
POST   /api/reports
GET    /api/admin/users
PUT    /api/admin/users/{id}/status
GET    /api/admin/products
PUT    /api/admin/products/{id}/status
GET    /api/admin/categories
POST   /api/admin/categories
PUT    /api/admin/categories/{id}
DELETE /api/admin/categories/{id}
GET    /api/admin/reports
PUT    /api/admin/reports/{id}
```

## 后续可扩展

- 真实校园身份认证、邮箱/手机验证码
- WebSocket 实时聊天
- 商品推荐 / 浏览历史
- 卖家信用、交易评价
- 阿里云 OSS / MinIO 图片存储
- Nginx + JAR 部署脚本
