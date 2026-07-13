package com.smu.deal.config;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.UserMapper;
import org.springframework.boot.CommandLineRunner;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Component;

@Component
public class DataInitializer implements CommandLineRunner {

    private final UserMapper userMapper;
    private final PasswordEncoder passwordEncoder;
    private final JdbcTemplate jdbcTemplate;

    public DataInitializer(UserMapper userMapper, PasswordEncoder passwordEncoder, JdbcTemplate jdbcTemplate) {
        this.userMapper = userMapper;
        this.passwordEncoder = passwordEncoder;
        this.jdbcTemplate = jdbcTemplate;
    }

    @Override
    public void run(String... args) {
        ensureAnnouncementTable();
        ensureLostFoundTables();
        ensureUser("admin", "系统管理员", "admin123", "ADMIN", null, null);
        ensureUser("student001", "张同学", "123456", "USER", "计算机学院", "主校区");
    }

    private void ensureAnnouncementTable() {
        jdbcTemplate.execute("""
                CREATE TABLE IF NOT EXISTS announcement (
                    id BIGINT PRIMARY KEY AUTO_INCREMENT,
                    title VARCHAR(80) NOT NULL,
                    content VARCHAR(500) NOT NULL,
                    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
                    created_by BIGINT,
                    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                    KEY idx_announcement_status_created (status, created_at, id)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
                """);
    }

    private void ensureLostFoundTables() {
        jdbcTemplate.execute("""
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
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
                """);
        jdbcTemplate.execute("""
                CREATE TABLE IF NOT EXISTS lost_found_image (
                    id BIGINT PRIMARY KEY AUTO_INCREMENT,
                    lost_found_id BIGINT NOT NULL,
                    image_url VARCHAR(256) NOT NULL,
                    sort_order INT NOT NULL DEFAULT 0,
                    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    KEY idx_lost_found_image_item_sort (lost_found_id, sort_order, id)
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
                """);
    }

    private void ensureUser(String studentNo, String name, String password, String role, String college, String campus) {
        Long count = userMapper.selectCount(new LambdaQueryWrapper<User>().eq(User::getStudentNo, studentNo));
        if (count > 0) return;
        User u = new User();
        u.setStudentNo(studentNo);
        u.setName(name);
        u.setPasswordHash(passwordEncoder.encode(password));
        u.setRole(role);
        u.setStatus("ACTIVE");
        u.setCollege(college);
        u.setCampus(campus);
        userMapper.insert(u);
    }
}
