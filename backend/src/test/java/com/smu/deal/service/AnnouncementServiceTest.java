package com.smu.deal.service;

import com.smu.deal.dto.AnnouncementDTO;
import com.smu.deal.entity.Announcement;
import com.smu.deal.mapper.AnnouncementMapper;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Proxy;
import java.time.LocalDateTime;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

class AnnouncementServiceTest {

    @Test
    void activeReturnsNullWhenNoAnnouncementIsActive() {
        InMemoryAnnouncements store = new InMemoryAnnouncements();
        AnnouncementService service = new AnnouncementService(store.mapper());

        assertThat(service.active()).isNull();
    }

    @Test
    void creatingActiveAnnouncementDisablesPreviousActiveAnnouncement() {
        InMemoryAnnouncements store = new InMemoryAnnouncements();
        AnnouncementService service = new AnnouncementService(store.mapper());

        AnnouncementDTO.SaveReq first = req("第一条", "旧公告", "ACTIVE");
        AnnouncementDTO.Item firstCreated = service.create(1L, first);

        AnnouncementDTO.SaveReq second = req("第二条", "新公告", "ACTIVE");
        AnnouncementDTO.Item secondCreated = service.create(1L, second);

        assertThat(store.find(firstCreated.getId()).getStatus()).isEqualTo("INACTIVE");
        assertThat(store.find(secondCreated.getId()).getStatus()).isEqualTo("ACTIVE");
        assertThat(service.active().getId()).isEqualTo(secondCreated.getId());
        assertThat(service.active().getContent()).isEqualTo("新公告");
    }

    @Test
    void updatingAnnouncementToActiveDisablesOtherActiveAnnouncement() {
        InMemoryAnnouncements store = new InMemoryAnnouncements();
        AnnouncementService service = new AnnouncementService(store.mapper());

        AnnouncementDTO.Item firstCreated = service.create(1L, req("第一条", "旧公告", "ACTIVE"));
        AnnouncementDTO.Item secondCreated = service.create(1L, req("第二条", "新公告", "INACTIVE"));

        service.update(secondCreated.getId(), req("第二条", "新公告更新", "ACTIVE"));

        assertThat(store.find(firstCreated.getId()).getStatus()).isEqualTo("INACTIVE");
        assertThat(store.find(secondCreated.getId()).getStatus()).isEqualTo("ACTIVE");
        assertThat(service.active().getContent()).isEqualTo("新公告更新");
    }

    private AnnouncementDTO.SaveReq req(String title, String content, String status) {
        AnnouncementDTO.SaveReq req = new AnnouncementDTO.SaveReq();
        req.setTitle(title);
        req.setContent(content);
        req.setStatus(status);
        return req;
    }

    private static class InMemoryAnnouncements {
        private final Map<Long, Announcement> rows = new LinkedHashMap<>();
        private long nextId = 1L;
        private long tick = 1L;

        AnnouncementMapper mapper() {
            return (AnnouncementMapper) Proxy.newProxyInstance(
                    AnnouncementMapper.class.getClassLoader(),
                    new Class[]{AnnouncementMapper.class},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "insert" -> insert((Announcement) args[0]);
                        case "selectOne" -> active();
                        case "selectList" -> rows.values().stream()
                                .map(InMemoryAnnouncements::copy)
                                .sorted(Comparator.comparing(Announcement::getCreatedAt).reversed())
                                .toList();
                        case "selectById" -> find(((Number) args[0]).longValue());
                        case "updateById" -> updateById((Announcement) args[0]);
                        case "update" -> disableOtherActive((Announcement) args[0], args[1]);
                        case "deleteById" -> rows.remove(((Number) args[0]).longValue()) == null ? 0 : 1;
                        default -> throw new UnsupportedOperationException(method.getName());
                    });
        }

        Announcement find(Long id) {
            Announcement row = rows.get(id);
            return row == null ? null : copy(row);
        }

        private int insert(Announcement row) {
            row.setId(nextId++);
            row.setCreatedAt(LocalDateTime.of(2026, 1, 1, 0, 0).plusNanos(tick++));
            row.setUpdatedAt(row.getCreatedAt());
            rows.put(row.getId(), copy(row));
            return 1;
        }

        private Announcement active() {
            return rows.values().stream()
                    .filter(row -> "ACTIVE".equals(row.getStatus()))
                    .max(Comparator.comparing(Announcement::getCreatedAt))
                    .map(InMemoryAnnouncements::copy)
                    .orElse(null);
        }

        private int updateById(Announcement patch) {
            Announcement row = rows.get(patch.getId());
            if (row == null) return 0;
            row.setTitle(patch.getTitle());
            row.setContent(patch.getContent());
            row.setStatus(patch.getStatus());
            if (patch.getCreatedBy() != null) row.setCreatedBy(patch.getCreatedBy());
            row.setUpdatedAt(LocalDateTime.of(2026, 1, 1, 0, 0).plusNanos(tick++));
            return 1;
        }

        private int disableOtherActive(Announcement patch, Object wrapper) {
            Long activeId = rows.values().stream()
                    .filter(row -> "ACTIVE".equals(row.getStatus()))
                    .max(Comparator.comparing(Announcement::getUpdatedAt))
                    .map(Announcement::getId)
                    .orElse(null);
            int changed = 0;
            for (Announcement row : rows.values()) {
                if (!"ACTIVE".equals(row.getStatus())) continue;
                if (row.getId().equals(activeId)) continue;
                row.setStatus(patch.getStatus());
                row.setUpdatedAt(LocalDateTime.of(2026, 1, 1, 0, 0).plusNanos(tick++));
                changed++;
            }
            return changed;
        }

        private static Announcement copy(Announcement src) {
            Announcement dst = new Announcement();
            dst.setId(src.getId());
            dst.setTitle(src.getTitle());
            dst.setContent(src.getContent());
            dst.setStatus(src.getStatus());
            dst.setCreatedBy(src.getCreatedBy());
            dst.setCreatedAt(src.getCreatedAt());
            dst.setUpdatedAt(src.getUpdatedAt());
            return dst;
        }
    }
}
