-- 校园二手交易平台 数据库初始化脚本

DROP DATABASE IF EXISTS smu_deal;
CREATE DATABASE smu_deal DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE smu_deal;

-- 用户表
CREATE TABLE user (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    student_no VARCHAR(32) NOT NULL,
    password_hash VARCHAR(128) NOT NULL,
    phone VARCHAR(32),
    college VARCHAR(64),
    campus VARCHAR(64),
    avatar VARCHAR(256),
    role VARCHAR(16) NOT NULL DEFAULT 'USER',
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_student_no (student_no)
) ENGINE=InnoDB;

-- 分类表
CREATE TABLE category (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    icon VARCHAR(128),
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- 商品表
CREATE TABLE product (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    seller_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    title VARCHAR(128) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    original_price DECIMAL(10,2),
    condition_level VARCHAR(32),
    trade_location VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'ON_SALE',
    view_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_seller (seller_id),
    KEY idx_category (category_id),
    KEY idx_status (status)
) ENGINE=InnoDB;

-- 商品图片表
CREATE TABLE product_image (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    product_id BIGINT NOT NULL,
    image_url VARCHAR(256) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_product (product_id)
) ENGINE=InnoDB;

-- 商品公开留言表
CREATE TABLE product_comment (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    product_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    content VARCHAR(300) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_product_created (product_id, created_at, id),
    KEY idx_user_created (user_id, created_at, id)
) ENGINE=InnoDB;

-- 收藏表
CREATE TABLE favorite (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_product (user_id, product_id)
) ENGINE=InnoDB;

-- 消息表
CREATE TABLE message (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    product_id BIGINT,
    content VARCHAR(1024) NOT NULL,
    is_read TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_receiver (receiver_id),
    KEY idx_pair (sender_id, receiver_id)
) ENGINE=InnoDB;

-- 交易表
CREATE TABLE trade_order (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    product_id BIGINT NOT NULL,
    buyer_id BIGINT NOT NULL,
    seller_id BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    meet_location VARCHAR(128),
    remark VARCHAR(256),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at DATETIME,
    KEY idx_buyer (buyer_id),
    KEY idx_seller (seller_id),
    KEY idx_product (product_id)
) ENGINE=InnoDB;

-- 举报表
CREATE TABLE report (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    reporter_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    reason VARCHAR(512) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    admin_remark VARCHAR(512),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- 意见箱
CREATE TABLE feedback (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT,
    category VARCHAR(32) NOT NULL DEFAULT '其他',
    content VARCHAR(2000) NOT NULL,
    contact VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    admin_reply VARCHAR(1000),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_user (user_id),
    KEY idx_status (status)
) ENGINE=InnoDB;

-- 初始分类
INSERT INTO category (name, sort_order) VALUES
('教材资料', 1),
('电子数码', 2),
('宿舍用品', 3),
('服装鞋包', 4),
('运动户外', 5),
('美妆个护', 6),
('交通工具', 7),
('票券卡券', 8),
('其他闲置', 9);

-- 默认账号会在后端首次启动时自动创建：
-- 管理员：admin / admin123
-- 演示用户：student001 / 123456
-- 见 com.smu.deal.config.DataInitializer
