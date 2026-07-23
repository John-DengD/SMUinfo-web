INSERT INTO category (name, sort_order) VALUES
('教材资料',1),('电子数码',2),('宿舍用品',3),('服装鞋包',4),('运动户外',5),
('美妆个护',6),('交通工具',7),('票券卡券',8),('其他闲置',9);

INSERT INTO "user" (name, student_no, password_hash, role, status, college, campus) VALUES
('系统管理员','admin','$2a$10$0NKkYmBLtPi5vbn6EORTBemqYlgVVE17eUzAi1HNDgZqACKW4txr2','ADMIN','ACTIVE',NULL,NULL),
('张同学','student001','$2a$10$N0zb4JMiELDnothog5rJy.kZEIIeqJKucizuClSCRQU2rmIpS.6Ue','USER','ACTIVE','计算机学院','主校区');
