# 服务器数据库初始化（PostgreSQL）

以下命令中的占位符（`<SERVER_IP>`、`<STRONG_PASSWORD>` 等）需替换为实际值。不要把真实服务器凭据提交到仓库。

## 1. 安装并启动 PostgreSQL

Ubuntu / Debian：
```bash
ssh root@<SERVER_IP>

apt update
apt install -y postgresql
systemctl enable --now postgresql
```

CentOS / Rocky：
```bash
ssh root@<SERVER_IP>

dnf install -y postgresql-server postgresql-contrib
postgresql-setup --initdb
systemctl enable --now postgresql
```

## 2. 创建数据库角色和数据库

以 `postgres` 超级用户身份进入 psql：

```bash
sudo -u postgres psql
```

在 psql 提示符下执行：

```sql
CREATE ROLE smu_deal LOGIN PASSWORD '<STRONG_PASSWORD>';
CREATE DATABASE smu_deal OWNER smu_deal;
\q
```

**无需手动导入任何 SQL 文件**。Go 二进制启动时会自动把 `$MIGRATIONS_DIR`（默认 `/opt/smu-deal/migrations`）下的迁移文件按顺序应用到数据库：

- `0001_init.up.sql` — 建表结构（users、products、categories、messages、orders、reports 等）
- `0002_seed.up.sql` — 种子数据（9 个分类、admin / student001 账号）
- `0003_transit_seed.up.sql` — 摆渡车时刻表

每次重启服务时，二进制会检测并只执行尚未应用的迁移，已执行过的跳过。因此部署新版本只需重启服务即可，无需手动运行 `migrate` 工具。

## 3. 配置服务

将 `deploy/smu-deal.service` 复制到服务器，编辑其中的 `DB_URL`、`JWT_SECRET`、`CORS_ALLOWED_ORIGINS`，然后：

```bash
sudo cp smu-deal.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now smu-deal.service
```

服务启动后，应用会自动连接 PostgreSQL 并完成数据库迁移，监听 `PORT`（默认 8080）。

可用以下命令验证健康状态：

```bash
curl http://127.0.0.1:8080/healthz
# 预期返回 200 {"code":0,...}

systemctl status smu-deal.service
```

## 4. 图片存储说明

商品图片**不会**存入 PostgreSQL，而是上传后写入后端机器的 `UPLOAD_DIR` 目录（默认 `/data/uploads`），DB 只保存路径（如 `/uploads/2026-05-28/xxx.jpg`）。

```bash
# 创建并挂载数据盘（示例）
mkdir -p /data/uploads
# 在 smu-deal.service 中设置：
#   Environment=UPLOAD_DIR=/data/uploads
```

40 G 硬盘按单图 500 KB 估算可存约 8 万张。后期数据量更大时，可平滑替换为 MinIO（自建 S3）或 OSS / COS，业务代码无需改动，只改 upload 层实现。
