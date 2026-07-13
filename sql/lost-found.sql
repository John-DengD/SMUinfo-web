CREATE TABLE IF NOT EXISTS lost_found (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    type VARCHAR(16) NOT NULL,
    title VARCHAR(80) NOT NULL,
    description VARCHAR(1000) NOT NULL,
    location VARCHAR(128),
    contact VARCHAR(128),
    status VARCHAR(16) NOT NULL DEFAULT 'OPEN',
    view_count INT NOT NULL DEFAULT 0,
    event_time DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_lost_found_type_status_created (type, status, created_at, id),
    KEY idx_lost_found_user_created (user_id, created_at, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lost_found_image (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    lost_found_id BIGINT NOT NULL,
    image_url VARCHAR(256) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_lost_found_image_item_sort (lost_found_id, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
