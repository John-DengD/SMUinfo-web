-- 仅新增意见箱表（不影响已有数据）
USE smu_deal;

CREATE TABLE IF NOT EXISTS feedback (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
