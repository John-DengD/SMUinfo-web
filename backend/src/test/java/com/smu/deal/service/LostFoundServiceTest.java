package com.smu.deal.service;

import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.LostFoundDTO;
import com.smu.deal.entity.LostFound;
import com.smu.deal.entity.LostFoundImage;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.LostFoundImageMapper;
import com.smu.deal.mapper.LostFoundMapper;
import com.smu.deal.mapper.UserMapper;
import org.junit.jupiter.api.Test;

import java.lang.reflect.Proxy;
import java.time.LocalDateTime;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class LostFoundServiceTest {

    @Test
    void createListsAndClosesLostFoundItems() {
        InMemoryLostFound store = new InMemoryLostFound();
        LostFoundService service = new LostFoundService(store.lostFoundMapper(), store.imageMapper(), store.userMapper());

        LostFoundDTO.CreateReq req = new LostFoundDTO.CreateReq();
        req.setType("FOUND");
        req.setTitle("捡到校园卡");
        req.setDescription("在图书馆门口捡到一张校园卡");
        req.setLocation("图书馆门口");
        req.setContact("微信联系");
        req.setImages(List.of("/uploads/2026-06-20/card.jpg"));

        LostFoundDTO.Item created = service.create(1L, req);

        assertThat(created.getType()).isEqualTo("FOUND");
        assertThat(created.getTypeText()).isEqualTo("招领");
        assertThat(created.getCover()).isEqualTo("/uploads/2026-06-20/card.jpg");
        assertThat(service.list(new LostFoundDTO.ListQuery()).getRecords()).hasSize(1);

        service.close(created.getId(), 1L, false);

        assertThat(service.list(new LostFoundDTO.ListQuery()).getRecords()).isEmpty();
        assertThatThrownBy(() -> service.detail(created.getId())).isInstanceOf(BusinessException.class);
    }

    private static class InMemoryLostFound {
        private final Map<Long, LostFound> rows = new LinkedHashMap<>();
        private final Map<Long, LostFoundImage> images = new LinkedHashMap<>();
        private final Map<Long, User> users = new LinkedHashMap<>();
        private long nextId = 1L;
        private long nextImageId = 1L;

        InMemoryLostFound() {
            User user = new User();
            user.setId(1L);
            user.setName("张同学");
            user.setCampus("临港校区");
            users.put(user.getId(), user);
        }

        LostFoundMapper lostFoundMapper() {
            return (LostFoundMapper) Proxy.newProxyInstance(LostFoundMapper.class.getClassLoader(),
                    new Class[]{LostFoundMapper.class},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "insert" -> insert((LostFound) args[0]);
                        case "selectById" -> copy(rows.get(((Number) args[0]).longValue()));
                        case "updateById" -> update((LostFound) args[0]);
                        case "selectPage" -> selectPage((Page<LostFound>) args[0]);
                        default -> throw new UnsupportedOperationException(method.getName());
                    });
        }

        LostFoundImageMapper imageMapper() {
            return (LostFoundImageMapper) Proxy.newProxyInstance(LostFoundImageMapper.class.getClassLoader(),
                    new Class[]{LostFoundImageMapper.class},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "insert" -> insertImage((LostFoundImage) args[0]);
                        case "selectList" -> images.values().stream()
                                .sorted(Comparator.comparing(LostFoundImage::getSortOrder))
                                .map(InMemoryLostFound::copyImage)
                                .toList();
                        default -> throw new UnsupportedOperationException(method.getName());
                    });
        }

        UserMapper userMapper() {
            return (UserMapper) Proxy.newProxyInstance(UserMapper.class.getClassLoader(),
                    new Class[]{UserMapper.class},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "selectBatchIds" -> users.values().stream().toList();
                        default -> throw new UnsupportedOperationException(method.getName());
                    });
        }

        private int insert(LostFound row) {
            row.setId(nextId++);
            row.setCreatedAt(LocalDateTime.of(2026, 6, 20, 12, 0).plusSeconds(row.getId()));
            row.setUpdatedAt(row.getCreatedAt());
            rows.put(row.getId(), copy(row));
            return 1;
        }

        private int update(LostFound patch) {
            LostFound row = rows.get(patch.getId());
            if (row == null) return 0;
            row.setStatus(patch.getStatus());
            row.setViewCount(patch.getViewCount());
            return 1;
        }

        private Page<LostFound> selectPage(Page<LostFound> page) {
            List<LostFound> open = rows.values().stream()
                    .filter(row -> "OPEN".equals(row.getStatus()))
                    .sorted(Comparator.comparing(LostFound::getCreatedAt).reversed())
                    .map(InMemoryLostFound::copy)
                    .toList();
            page.setTotal(open.size());
            page.setRecords(open);
            return page;
        }

        private int insertImage(LostFoundImage image) {
            image.setId(nextImageId++);
            images.put(image.getId(), copyImage(image));
            return 1;
        }

        private static LostFound copy(LostFound src) {
            if (src == null) return null;
            LostFound dst = new LostFound();
            dst.setId(src.getId());
            dst.setUserId(src.getUserId());
            dst.setType(src.getType());
            dst.setTitle(src.getTitle());
            dst.setDescription(src.getDescription());
            dst.setLocation(src.getLocation());
            dst.setContact(src.getContact());
            dst.setStatus(src.getStatus());
            dst.setViewCount(src.getViewCount());
            dst.setCreatedAt(src.getCreatedAt());
            dst.setUpdatedAt(src.getUpdatedAt());
            return dst;
        }

        private static LostFoundImage copyImage(LostFoundImage src) {
            LostFoundImage dst = new LostFoundImage();
            dst.setId(src.getId());
            dst.setLostFoundId(src.getLostFoundId());
            dst.setImageUrl(src.getImageUrl());
            dst.setSortOrder(src.getSortOrder());
            return dst;
        }
    }
}
