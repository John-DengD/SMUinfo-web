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
    line_code VARCHAR(32) NOT NULL, line_name VARCHAR(64) NOT NULL,
    station_code VARCHAR(32) NOT NULL, station_name VARCHAR(64) NOT NULL,
    direction_code VARCHAR(32) NOT NULL, direction_name VARCHAR(64) NOT NULL,
    schedule_type VARCHAR(32) NOT NULL, schedule_type_name VARCHAR(64) NOT NULL,
    departure_time TIME NOT NULL,
    service_type VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
    service_label VARCHAR(64) NOT NULL DEFAULT '普通车',
    sort_order INT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_transit_upd BEFORE UPDATE ON transit_departure FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Performance indexes (ported from sql/performance-indexes.sql and TransitDataInitializer)
CREATE INDEX IF NOT EXISTS idx_product_list_created ON product (status, created_at, id);
CREATE INDEX IF NOT EXISTS idx_product_category_created ON product (category_id, status, created_at, id);
CREATE INDEX IF NOT EXISTS idx_product_seller_created ON product (seller_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_product_price ON product (status, price, id);
CREATE INDEX IF NOT EXISTS idx_product_hot ON product (status, view_count, id);
CREATE INDEX IF NOT EXISTS idx_product_condition ON product (condition_level, status, id);

-- Superset of idx_product_image_product; drop the redundant single-column index.
DROP INDEX IF EXISTS idx_product_image_product;
CREATE INDEX IF NOT EXISTS idx_product_image_product_sort ON product_image (product_id, sort_order, id);

CREATE INDEX IF NOT EXISTS idx_product_comment_product_created ON product_comment (product_id, created_at, id);

CREATE INDEX IF NOT EXISTS idx_message_user_created ON message (receiver_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_message_sender_created ON message (sender_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_message_pair_created ON message (sender_id, receiver_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_message_unread ON message (receiver_id, is_read, id);

CREATE INDEX IF NOT EXISTS idx_trade_order_buyer_created ON trade_order (buyer_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_trade_order_seller_created ON trade_order (seller_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_trade_order_product_status ON trade_order (product_id, status, id);

CREATE INDEX IF NOT EXISTS idx_report_status_created ON report (status, created_at, id);

CREATE INDEX IF NOT EXISTS idx_feedback_user_created ON feedback (user_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_feedback_status_created ON feedback (status, created_at, id);

CREATE INDEX IF NOT EXISTS idx_category_status_sort ON category (status, sort_order, id);

CREATE INDEX IF NOT EXISTS idx_transit_lookup ON transit_departure (line_code, schedule_type, direction_code, station_code, status, departure_time);
CREATE INDEX IF NOT EXISTS idx_transit_options ON transit_departure (line_code, status, sort_order);
