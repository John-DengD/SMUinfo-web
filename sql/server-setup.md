# 服务器数据库初始化

以下命令中的 `<SERVER_IP>`、`<DB_USER>` 和 `<STRONG_DB_PASSWORD>` 需要替换为实际值。不要把真实服务器凭据提交到仓库。

## 1. 登录服务器，安装并启动 MySQL（如未安装）

Ubuntu / Debian：
```bash
ssh root@<SERVER_IP>

apt update
apt install -y mysql-server
systemctl enable --now mysql
```

CentOS / Rocky：
```bash
ssh root@<SERVER_IP>

yum install -y mysql-server
systemctl enable --now mysqld
```

## 2. 允许外网访问（开发期方便，生产建议改 SSH 隧道或安全组限制 IP）

```bash
# 进入 MySQL
mysql -uroot

# 创建仅用于本项目的数据库账号
CREATE USER '<DB_USER>'@'%' IDENTIFIED BY '<STRONG_DB_PASSWORD>';
GRANT ALL PRIVILEGES ON smu_deal.* TO '<DB_USER>'@'%';
FLUSH PRIVILEGES;
exit;
```

修改 `/etc/mysql/mysql.conf.d/mysqld.cnf`（或 `/etc/my.cnf`），把 `bind-address = 127.0.0.1` 改成 `bind-address = 0.0.0.0`，然后：
```bash
systemctl restart mysql
```

开放 3306 端口（云厂商控制台安全组也要放行）：
```bash
ufw allow 3306/tcp     # Ubuntu
firewall-cmd --permanent --add-port=3306/tcp && firewall-cmd --reload  # CentOS
```

## 3. 导入建表 SQL

在本机执行（无需登录服务器）：
```bash
mysql -h <SERVER_IP> -u<DB_USER> -p < sql/init.sql
```

或登录服务器后：
```bash
ssh root@<SERVER_IP>
mysql -u<DB_USER> -p < /path/to/init.sql
```

## 4. 验证

```bash
mysql -h <SERVER_IP> -u<DB_USER> -p -e "USE smu_deal; SHOW TABLES;"
```

应当看到 `user / category / product / product_image / favorite / message / trade_order / report` 八张表。

## 5. 图片存储说明

商品图片**不会**存到 MySQL，而是上传后写入后端机器的 `uploads/` 目录，DB 只保存路径（如 `/uploads/2026-05-28/xxx.jpg`）。

- 如果后端也部署在该服务器上：可以直接挂数据盘到 `/data/uploads/`，并设环境变量 `UPLOAD_DIR=/data/uploads` 启动后端。40G 硬盘按单图 500KB 估算约 8 万张。
- 如果后端在本地开发：图片就存在本地 `backend/uploads/` 下，不上传到服务器。
- 后期数据量更大时，可平滑替换成 MinIO（自建 S3）或 OSS / COS，业务代码无需改动，只改 `UploadService.upload()` 的实现。
