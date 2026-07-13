package com.smu.deal.config;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.entity.TransitDeparture;
import com.smu.deal.mapper.TransitDepartureMapper;
import org.springframework.boot.CommandLineRunner;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

import java.time.LocalTime;
import java.util.ArrayList;
import java.util.List;

@Component
public class TransitDataInitializer implements CommandLineRunner {

    private static final String METRO_16 = "METRO_16";
    private static final String BUS_1077 = "BUS_1077";
    private static final String TO_LONGYANG = "TO_LONGYANG";
    private static final String TO_DISHUI = "TO_DISHUI";
    private static final String TO_LINGANG_AVE = "TO_LINGANG_AVE";
    private static final String TO_SHARED = "TO_SHARED";
    private static final String NORMAL = "NORMAL";
    private static final String EXPRESS_DIRECT = "EXPRESS_DIRECT";

    private final JdbcTemplate jdbcTemplate;
    private final TransitDepartureMapper transitDepartureMapper;

    public TransitDataInitializer(JdbcTemplate jdbcTemplate, TransitDepartureMapper transitDepartureMapper) {
        this.jdbcTemplate = jdbcTemplate;
        this.transitDepartureMapper = transitDepartureMapper;
    }

    @Override
    public void run(String... args) {
        createTable();
        Long count = transitDepartureMapper.selectCount(new LambdaQueryWrapper<TransitDeparture>());
        if (count != null && count > 0) return;

        List<TransitDeparture> rows = new ArrayList<>();
        seedMetro(rows);
        seedBus1077(rows);
        rows.forEach(transitDepartureMapper::insert);
    }

    private void createTable() {
        jdbcTemplate.execute("""
                CREATE TABLE IF NOT EXISTS transit_departure (
                  id BIGINT PRIMARY KEY AUTO_INCREMENT,
                  line_code VARCHAR(32) NOT NULL,
                  line_name VARCHAR(64) NOT NULL,
                  station_code VARCHAR(64) NOT NULL,
                  station_name VARCHAR(64) NOT NULL,
                  direction_code VARCHAR(32) NOT NULL,
                  direction_name VARCHAR(64) NOT NULL,
                  schedule_type VARCHAR(32) NOT NULL,
                  schedule_type_name VARCHAR(64) NOT NULL,
                  departure_time TIME NOT NULL,
                  service_type VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
                  service_label VARCHAR(64) NOT NULL DEFAULT '普通车',
                  sort_order INT NOT NULL DEFAULT 0,
                  status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
                  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
                  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                  KEY idx_transit_lookup (line_code, schedule_type, direction_code, station_code, status, departure_time),
                  KEY idx_transit_options (line_code, status, sort_order)
                ) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
                """);
        ensureColumn("service_type", "ALTER TABLE transit_departure ADD COLUMN service_type VARCHAR(32) NOT NULL DEFAULT 'NORMAL' AFTER departure_time");
        ensureColumn("service_label", "ALTER TABLE transit_departure ADD COLUMN service_label VARCHAR(64) NOT NULL DEFAULT '普通车' AFTER service_type");
        jdbcTemplate.execute("""
                UPDATE transit_departure
                SET service_type = 'EXPRESS_DIRECT', service_label = '大站/直达'
                WHERE line_code = 'METRO_16'
                  AND sort_order BETWEEN 1000 AND 2199
                """);
    }

    private void seedMetro(List<TransitDeparture> rows) {
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_LONGYANG, "往龙阳路", "罗山路", "罗山路", EXPRESS_DIRECT, "大站/直达", 1000,
                "07:42 10:41 11:41 12:41 13:41 14:41 15:41 16:41 17:41 19:41 20:41 21:41");
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_LONGYANG, "往龙阳路", "龙阳路", "龙阳路", EXPRESS_DIRECT, "大站/直达", 1100,
                "07:48 10:46 11:46 12:46 13:46 14:46 15:46 16:46 17:46 17:54 18:34 19:46 20:46 21:46");
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_LONGYANG, "往龙阳路", "临港大道", "临港大道", EXPRESS_DIRECT, "大站/直达", 1200,
                "07:03 10:03 11:03 12:03 13:03 14:03 15:03 16:03 17:03 19:03 20:03 21:03");
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_DISHUI, "往临港大道/滴水湖", "罗山路", "罗山路", EXPRESS_DIRECT, "大站/直达", 1300,
                "07:06 10:06 11:06 12:06 13:06 14:06 15:06 16:06 17:06 18:06 19:06 20:06 21:06");
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_DISHUI, "往临港大道/滴水湖", "龙阳路", "龙阳路", EXPRESS_DIRECT, "大站/直达", 1400,
                "07:00 07:30 07:45 10:00 11:00 12:00 13:00 14:00 15:00 16:00 17:00 18:00 19:00 20:00 21:00");
        add(rows, METRO_16, "16号线", "WEEKDAY", "周一至周五", TO_DISHUI, "往临港大道/滴水湖", "临港大道", "临港大道", EXPRESS_DIRECT, "大站/直达", 1500,
                "07:43 10:43 11:43 12:43 13:43 14:43 15:43 16:43 17:43 18:43 19:43 20:43 21:43");

        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_LONGYANG, "往龙阳路", "罗山路", "罗山路", EXPRESS_DIRECT, "大站/直达", 1600,
                "07:41 08:41 09:41 10:41 11:41 12:41 13:41 14:41 15:41 16:41 17:41 18:41 19:41 20:41 21:41");
        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_LONGYANG, "往龙阳路", "龙阳路", "龙阳路", EXPRESS_DIRECT, "大站/直达", 1700,
                "07:46 08:46 09:46 10:46 11:46 12:46 13:46 14:46 15:46 16:46 17:46 18:46 19:46 20:46 21:46");
        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_LONGYANG, "往龙阳路", "临港大道", "临港大道", EXPRESS_DIRECT, "大站/直达", 1800,
                "07:03 08:03 09:03 10:03 11:03 12:03 13:03 14:03 15:03 16:03 17:03 18:03 19:03 20:03 21:03");
        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_DISHUI, "往临港大道/滴水湖", "罗山路", "罗山路", EXPRESS_DIRECT, "大站/直达", 1900,
                "07:06 08:06 09:06 10:06 11:06 12:06 13:06 14:06 15:06 16:06 17:06 18:06 19:06 20:06 21:06");
        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_DISHUI, "往临港大道/滴水湖", "龙阳路", "龙阳路", EXPRESS_DIRECT, "大站/直达", 2000,
                "07:00 08:00 09:00 10:00 11:00 12:00 13:00 14:00 15:00 16:00 17:00 18:00 19:00 20:00 21:00");
        add(rows, METRO_16, "16号线", "WEEKEND", "周末/节假日", TO_DISHUI, "往临港大道/滴水湖", "临港大道", "临港大道", EXPRESS_DIRECT, "大站/直达", 2100,
                "07:43 08:43 09:43 10:43 11:43 12:43 13:43 14:43 15:43 16:43 17:43 18:43 19:43 20:43 21:43");
    }

    private void seedBus1077(List<TransitDeparture> rows) {
        add(rows, BUS_1077, "1077路", "BUS_NORMAL", "周一至周四/周六", TO_LINGANG_AVE, "往临港大道", "临港共享区枢纽站", "共享区枢纽站", NORMAL, "普通车", 3000,
                "05:40 05:55 06:10 06:20 06:30 06:40 06:50 07:00 07:10 07:20 07:30 07:40 07:50 08:00 08:10 08:22 08:34 08:46 08:58 09:10 09:22 09:35 09:50 10:05 10:20 10:35 10:50 11:05 11:20 11:35 11:45 11:55 12:05 12:15 12:25 12:35 12:45 12:55 13:05 13:15 13:25 13:35 13:45 13:55 14:05 14:15 14:25 14:35 14:45 14:55 15:05 15:15 15:25 15:35 15:45 15:55 16:05 16:15 16:25 16:36 16:48 17:00 17:12 17:24 17:36 17:48 18:00 18:12 18:24 18:36 18:48 19:00 19:12 19:25 19:40 19:55 20:10 20:25 20:40 20:55 21:10 21:25 21:40 21:55 22:10 22:30 22:50 23:10 23:30");
        add(rows, BUS_1077, "1077路", "BUS_NORMAL", "周一至周四/周六", TO_SHARED, "往共享区", "临港大道枢纽站", "临港大道枢纽站", NORMAL, "普通车", 3100,
                "06:00 06:20 06:35 06:50 07:00 07:10 07:20 07:30 07:40 07:50 08:00 08:10 08:20 08:30 08:40 08:52 09:04 09:16 09:28 09:40 09:52 10:05 10:20 10:35 10:50 11:05 11:20 11:35 11:50 12:00 12:10 12:20 12:30 12:40 12:50 13:00 13:10 13:20 13:30 13:40 13:50 14:00 14:10 14:20 14:30 14:40 14:50 15:00 15:10 15:20 15:30 15:40 15:50 16:00 16:10 16:20 16:30 16:40 16:50 17:02 17:14 17:26 17:38 17:50 18:02 18:14 18:26 18:38 18:50 19:02 19:15 19:28 19:41 19:55 20:10 20:25 20:40 20:55 21:10 21:25 21:40 21:55 22:10 22:25 22:40 23:00 23:20 23:40 23:54");

        add(rows, BUS_1077, "1077路", "BUS_FRIDAY", "周五", TO_LINGANG_AVE, "往临港大道", "临港共享区枢纽站", "共享区枢纽站", NORMAL, "普通车", 3200,
                "05:40 05:55 06:10 06:20 06:30 06:40 06:50 07:00 07:10 07:18 07:25 07:32 07:40 07:50 08:00 08:10 08:20 08:28 08:36 08:46 08:58 09:10 09:20 09:28 09:40 09:55 10:10 10:20 10:28 10:40 10:52 11:05 11:20 11:35 11:45 11:55 12:05 12:15 12:25 12:33 12:40 12:48 12:56 13:05 13:15 13:25 13:35 13:42 13:50 13:58 14:06 14:15 14:25 14:32 14:40 14:48 14:56 15:05 15:15 15:25 15:35 15:42 15:50 15:58 16:06 16:15 16:25 16:36 16:48 17:00 17:12 17:24 17:36 17:48 18:00 18:12 18:24 18:36 18:48 19:00 19:12 19:25 19:40 19:55 20:10 20:25 20:40 20:55 21:10 21:25 21:40 21:55 22:10 22:30 22:50 23:10 23:30");
        add(rows, BUS_1077, "1077路", "BUS_FRIDAY", "周五", TO_SHARED, "往共享区", "临港大道枢纽站", "临港大道枢纽站", NORMAL, "普通车", 3300,
                "06:00 06:20 06:35 06:50 07:00 07:10 07:20 07:30 07:40 07:50 07:57 08:04 08:12 08:20 08:30 08:40 08:50 08:58 09:06 09:16 09:28 09:40 09:50 09:58 10:10 10:25 10:40 10:50 10:58 11:10 11:22 11:35 11:50 12:00 12:10 12:20 12:30 12:40 12:50 12:58 13:05 13:13 13:21 13:30 13:40 13:50 14:00 14:07 14:15 14:23 14:30 14:40 14:50 14:57 15:05 15:13 15:21 15:30 15:40 15:50 16:00 16:07 16:15 16:23 16:31 16:40 16:50 17:02 17:14 17:26 17:38 17:50 18:02 18:14 18:26 18:38 18:50 19:02 19:15 19:28 19:41 19:55 20:10 20:25 20:40 20:55 21:10 21:25 21:40 21:55 22:10 22:25 22:40 23:00 23:20 23:40 23:54");

        add(rows, BUS_1077, "1077路", "BUS_SUNDAY", "周日", TO_LINGANG_AVE, "往临港大道", "临港共享区枢纽站", "共享区枢纽站", NORMAL, "普通车", 3400,
                "05:40 06:00 06:15 06:30 06:45 07:00 07:15 07:30 07:40 07:50 08:00 08:10 08:20 08:30 08:40 08:50 09:00 09:12 09:24 09:35 09:50 10:05 10:20 10:35 10:50 11:05 11:20 11:35 11:40 11:50 12:00 12:10 12:20 12:30 12:40 12:50 13:00 13:10 13:20 13:30 13:35 13:45 13:58 14:00 14:08 14:15 14:25 14:35 14:45 14:58 15:10 15:20 15:28 15:35 15:42 15:50 16:00 16:10 16:20 16:30 16:40 16:50 16:58 17:05 17:12 17:20 17:30 17:40 17:50 17:58 18:00 18:10 18:20 18:30 18:40 18:50 18:58 19:00 19:12 19:20 19:30 19:40 19:50 20:00 20:10 20:20 20:30 20:45 20:50 21:00 21:10 21:20 21:30 21:50 22:10 22:30 22:50 23:10 23:30");
        add(rows, BUS_1077, "1077路", "BUS_SUNDAY", "周日", TO_SHARED, "往共享区", "临港大道枢纽站", "临港大道枢纽站", NORMAL, "普通车", 3500,
                "06:00 06:20 06:35 06:50 07:05 07:20 07:35 07:50 08:05 08:20 08:35 08:50 09:05 09:20 09:35 09:50 10:05 10:20 10:35 10:50 11:05 11:20 11:35 11:50 12:00 12:10 12:20 12:30 12:40 12:50 13:00 13:10 13:20 13:30 13:40 13:50 13:58 14:05 14:12 14:20 14:30 14:40 14:50 15:00 15:10 15:20 15:30 15:40 15:50 16:00 16:10 16:20 16:28 16:35 16:42 16:50 16:58 17:05 17:12 17:20 17:28 17:35 17:42 17:50 18:00 18:10 18:20 18:28 18:35 18:42 18:50 19:00 19:10 19:20 19:30 19:40 19:50 19:58 20:05 20:12 20:20 20:30 20:40 20:50 21:00 21:10 21:20 21:30 21:40 21:50 22:00 22:20 22:40 23:00 23:20 23:40 23:54");
    }

    private void add(List<TransitDeparture> rows, String lineCode, String lineName, String scheduleType,
                     String scheduleTypeName, String directionCode, String directionName, String stationCode,
                     String stationName, String serviceType, String serviceLabel, int sortBase, String times) {
        String[] parts = times.split("\\s+");
        for (int i = 0; i < parts.length; i++) {
            TransitDeparture row = new TransitDeparture();
            row.setLineCode(lineCode);
            row.setLineName(lineName);
            row.setStationCode(stationCode);
            row.setStationName(stationName);
            row.setDirectionCode(directionCode);
            row.setDirectionName(directionName);
            row.setScheduleType(scheduleType);
            row.setScheduleTypeName(scheduleTypeName);
            row.setDepartureTime(LocalTime.parse(parts[i]));
            row.setServiceType(serviceType);
            row.setServiceLabel(serviceLabel);
            row.setSortOrder(sortBase + i);
            row.setStatus("ACTIVE");
            rows.add(row);
        }
    }

    private void ensureColumn(String columnName, String sql) {
        Integer count = jdbcTemplate.queryForObject("""
                SELECT COUNT(*)
                FROM information_schema.columns
                WHERE table_schema = DATABASE()
                  AND table_name = 'transit_departure'
                  AND column_name = ?
                """, Integer.class, columnName);
        if (count == null || count == 0) {
            jdbcTemplate.execute(sql);
        }
    }
}
