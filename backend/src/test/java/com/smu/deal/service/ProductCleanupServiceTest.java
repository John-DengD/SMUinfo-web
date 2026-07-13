package com.smu.deal.service;

import com.smu.deal.entity.Favorite;
import com.smu.deal.entity.Message;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.ProductComment;
import com.smu.deal.entity.ProductImage;
import com.smu.deal.entity.Report;
import com.smu.deal.entity.TradeOrder;
import com.smu.deal.mapper.FavoriteMapper;
import com.smu.deal.mapper.MessageMapper;
import com.smu.deal.mapper.ProductCommentMapper;
import com.smu.deal.mapper.ProductImageMapper;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.ReportMapper;
import com.smu.deal.mapper.TradeOrderMapper;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.lang.reflect.Proxy;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

import static org.assertj.core.api.Assertions.assertThat;

class ProductCleanupServiceTest {

    @TempDir
    Path uploadRoot;

    @Test
    void cleanupClosedProductsDeletesOldSoldAndOfflineProductsWithRelatedRowsAndFiles() throws Exception {
        CleanupStore store = new CleanupStore();
        LocalDateTime now = LocalDateTime.of(2026, 6, 20, 12, 0);
        Clock clock = Clock.fixed(now.atZone(ZoneId.systemDefault()).toInstant(), ZoneId.systemDefault());

        Files.createDirectories(uploadRoot.resolve("2026-06-01"));
        Files.writeString(uploadRoot.resolve("2026-06-01/old-sold.jpg"), "old sold");
        Files.writeString(uploadRoot.resolve("2026-06-01/old-offline.jpg"), "old offline");
        Files.writeString(uploadRoot.resolve("2026-06-01/new-sold.jpg"), "new sold");

        store.products.put(1L, product(1L, "SOLD", now.minusDays(9)));
        store.products.put(2L, product(2L, "OFFLINE", now.minusDays(8)));
        store.products.put(3L, product(3L, "SOLD", now.minusDays(2)));
        store.products.put(4L, product(4L, "ON_SALE", now.minusDays(20)));
        store.images.put(11L, image(11L, 1L, "/uploads/2026-06-01/old-sold.jpg"));
        store.images.put(12L, image(12L, 2L, "/uploads/2026-06-01/old-offline.jpg"));
        store.images.put(13L, image(13L, 3L, "/uploads/2026-06-01/new-sold.jpg"));
        store.favorites.put(21L, favorite(21L, 1L));
        store.favorites.put(22L, favorite(22L, 4L));
        store.comments.put(31L, comment(31L, 2L));
        store.messages.put(41L, message(41L, 1L));
        store.orders.put(51L, order(51L, 2L));
        store.reports.put(61L, report(61L, 1L));

        ProductCleanupService service = new ProductCleanupService(
                store.productMapper(),
                store.imageMapper(),
                store.favoriteMapper(),
                store.commentMapper(),
                store.messageMapper(),
                store.orderMapper(),
                store.reportMapper(),
                uploadRoot,
                clock);

        ProductCleanupService.CleanupResult result = service.cleanupClosedProducts(100, Duration.ofDays(7));

        assertThat(result.deletedProducts()).isEqualTo(2);
        assertThat(result.deletedFiles()).isEqualTo(2);
        assertThat(store.products.keySet()).containsExactlyInAnyOrder(3L, 4L);
        assertThat(store.images.keySet()).containsExactly(13L);
        assertThat(store.favorites.keySet()).containsExactly(22L);
        assertThat(store.comments).isEmpty();
        assertThat(store.messages).isEmpty();
        assertThat(store.orders).isEmpty();
        assertThat(store.reports).isEmpty();
        assertThat(Files.exists(uploadRoot.resolve("2026-06-01/old-sold.jpg"))).isFalse();
        assertThat(Files.exists(uploadRoot.resolve("2026-06-01/old-offline.jpg"))).isFalse();
        assertThat(Files.exists(uploadRoot.resolve("2026-06-01/new-sold.jpg"))).isTrue();
    }

    private static Product product(Long id, String status, LocalDateTime updatedAt) {
        Product product = new Product();
        product.setId(id);
        product.setStatus(status);
        product.setUpdatedAt(updatedAt);
        return product;
    }

    private static ProductImage image(Long id, Long productId, String url) {
        ProductImage image = new ProductImage();
        image.setId(id);
        image.setProductId(productId);
        image.setImageUrl(url);
        return image;
    }

    private static Favorite favorite(Long id, Long productId) {
        Favorite favorite = new Favorite();
        favorite.setId(id);
        favorite.setProductId(productId);
        return favorite;
    }

    private static ProductComment comment(Long id, Long productId) {
        ProductComment comment = new ProductComment();
        comment.setId(id);
        comment.setProductId(productId);
        return comment;
    }

    private static Message message(Long id, Long productId) {
        Message message = new Message();
        message.setId(id);
        message.setProductId(productId);
        return message;
    }

    private static TradeOrder order(Long id, Long productId) {
        TradeOrder order = new TradeOrder();
        order.setId(id);
        order.setProductId(productId);
        return order;
    }

    private static Report report(Long id, Long productId) {
        Report report = new Report();
        report.setId(id);
        report.setProductId(productId);
        return report;
    }

    private static class CleanupStore {
        private final Map<Long, Product> products = new LinkedHashMap<>();
        private final Map<Long, ProductImage> images = new LinkedHashMap<>();
        private final Map<Long, Favorite> favorites = new LinkedHashMap<>();
        private final Map<Long, ProductComment> comments = new LinkedHashMap<>();
        private final Map<Long, Message> messages = new LinkedHashMap<>();
        private final Map<Long, TradeOrder> orders = new LinkedHashMap<>();
        private final Map<Long, Report> reports = new LinkedHashMap<>();
        private Set<Long> selectedProductIds = Set.of();

        ProductMapper productMapper() {
            return (ProductMapper) Proxy.newProxyInstance(ProductMapper.class.getClassLoader(), new Class[]{ProductMapper.class},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "selectList" -> {
                            List<Product> selected = products.values().stream()
                                    .filter(product -> List.of("SOLD", "OFFLINE").contains(product.getStatus()))
                                    .filter(product -> product.getUpdatedAt().isBefore(LocalDateTime.of(2026, 6, 13, 12, 0)))
                                    .sorted(Comparator.comparing(Product::getUpdatedAt))
                                    .toList();
                            selectedProductIds = selected.stream().map(Product::getId).collect(Collectors.toSet());
                            yield selected;
                        }
                        case "deleteBatchIds" -> {
                            ids(args[0]).forEach(products::remove);
                            yield ids(args[0]).size();
                        }
                        default -> throw new UnsupportedOperationException(method.getName());
                    });
        }

        ProductImageMapper imageMapper() {
            return mapper(ProductImageMapper.class, images, ProductImage::getProductId);
        }

        FavoriteMapper favoriteMapper() {
            return mapper(FavoriteMapper.class, favorites, Favorite::getProductId);
        }

        ProductCommentMapper commentMapper() {
            return mapper(ProductCommentMapper.class, comments, ProductComment::getProductId);
        }

        MessageMapper messageMapper() {
            return mapper(MessageMapper.class, messages, Message::getProductId);
        }

        TradeOrderMapper orderMapper() {
            return mapper(TradeOrderMapper.class, orders, TradeOrder::getProductId);
        }

        ReportMapper reportMapper() {
            return mapper(ReportMapper.class, reports, Report::getProductId);
        }

        private <M, T> M mapper(Class<M> mapperType, Map<Long, T> rows, ProductIdGetter<T> productIdGetter) {
            return mapperType.cast(Proxy.newProxyInstance(mapperType.getClassLoader(), new Class[]{mapperType},
                    (proxy, method, args) -> switch (method.getName()) {
                        case "selectList" -> rows.values().stream()
                                .filter(row -> selectedProductIds.contains(productIdGetter.productId(row)))
                                .toList();
                        case "delete" -> {
                            List<Long> removed = new ArrayList<>();
                            rows.forEach((id, row) -> {
                                if (selectedProductIds.contains(productIdGetter.productId(row))) {
                                    removed.add(id);
                                }
                            });
                            removed.forEach(rows::remove);
                            yield removed.size();
                        }
                        default -> throw new UnsupportedOperationException(method.getName());
                    }));
        }

        private static Collection<Long> ids(Object raw) {
            if (raw instanceof Collection<?> collection) {
                return collection.stream().map(id -> ((Number) id).longValue()).toList();
            }
            throw new IllegalArgumentException("expected id collection");
        }
    }

    private interface ProductIdGetter<T> {
        Long productId(T row);
    }
}
