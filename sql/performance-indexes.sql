-- Performance indexes for common list/search/message/order queries.
-- Safe to run repeatedly on MySQL 8+.

USE smu_deal;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_list_created') = 0, 'CREATE INDEX idx_product_list_created ON product (status, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_category_created') = 0, 'CREATE INDEX idx_product_category_created ON product (category_id, status, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_seller_created') = 0, 'CREATE INDEX idx_product_seller_created ON product (seller_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_price') = 0, 'CREATE INDEX idx_product_price ON product (status, price, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_hot') = 0, 'CREATE INDEX idx_product_hot ON product (status, view_count, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product' AND index_name = 'idx_product_condition') = 0, 'CREATE INDEX idx_product_condition ON product (condition_level, status, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product_image' AND index_name = 'idx_product_image_product_sort') = 0, 'CREATE INDEX idx_product_image_product_sort ON product_image (product_id, sort_order, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'product_comment' AND index_name = 'idx_product_comment_product_created') = 0, 'CREATE INDEX idx_product_comment_product_created ON product_comment (product_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'message' AND index_name = 'idx_message_user_created') = 0, 'CREATE INDEX idx_message_user_created ON message (receiver_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'message' AND index_name = 'idx_message_sender_created') = 0, 'CREATE INDEX idx_message_sender_created ON message (sender_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'message' AND index_name = 'idx_message_pair_created') = 0, 'CREATE INDEX idx_message_pair_created ON message (sender_id, receiver_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'message' AND index_name = 'idx_message_unread') = 0, 'CREATE INDEX idx_message_unread ON message (receiver_id, is_read, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'trade_order' AND index_name = 'idx_trade_order_buyer_created') = 0, 'CREATE INDEX idx_trade_order_buyer_created ON trade_order (buyer_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'trade_order' AND index_name = 'idx_trade_order_seller_created') = 0, 'CREATE INDEX idx_trade_order_seller_created ON trade_order (seller_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'trade_order' AND index_name = 'idx_trade_order_product_status') = 0, 'CREATE INDEX idx_trade_order_product_status ON trade_order (product_id, status, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'report' AND index_name = 'idx_report_status_created') = 0, 'CREATE INDEX idx_report_status_created ON report (status, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'feedback' AND index_name = 'idx_feedback_user_created') = 0, 'CREATE INDEX idx_feedback_user_created ON feedback (user_id, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'feedback' AND index_name = 'idx_feedback_status_created') = 0, 'CREATE INDEX idx_feedback_status_created ON feedback (status, created_at, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @s = IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'category' AND index_name = 'idx_category_status_sort') = 0, 'CREATE INDEX idx_category_status_sort ON category (status, sort_order, id)', 'SELECT 1');
PREPARE stmt FROM @s; EXECUTE stmt; DEALLOCATE PREPARE stmt;
