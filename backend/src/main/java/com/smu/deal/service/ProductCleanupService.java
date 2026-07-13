package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
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
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Clock;
import java.time.Duration;
import java.time.LocalDateTime;
import java.util.List;
import java.util.Objects;

@Service
public class ProductCleanupService {

    private static final Logger log = LoggerFactory.getLogger(ProductCleanupService.class);
    private static final List<String> CLEANABLE_STATUS = List.of("SOLD", "OFFLINE");

    private final ProductMapper productMapper;
    private final ProductImageMapper imageMapper;
    private final FavoriteMapper favoriteMapper;
    private final ProductCommentMapper commentMapper;
    private final MessageMapper messageMapper;
    private final TradeOrderMapper orderMapper;
    private final ReportMapper reportMapper;
    private final Path uploadRoot;
    private final Clock clock;

    @Autowired
    public ProductCleanupService(ProductMapper productMapper,
                                 ProductImageMapper imageMapper,
                                 FavoriteMapper favoriteMapper,
                                 ProductCommentMapper commentMapper,
                                 MessageMapper messageMapper,
                                 TradeOrderMapper orderMapper,
                                 ReportMapper reportMapper,
                                 @Value("${app.upload.dir}") String uploadDir) {
        this(productMapper, imageMapper, favoriteMapper, commentMapper, messageMapper,
                orderMapper, reportMapper, Path.of(uploadDir), Clock.systemDefaultZone());
    }

    ProductCleanupService(ProductMapper productMapper,
                          ProductImageMapper imageMapper,
                          FavoriteMapper favoriteMapper,
                          ProductCommentMapper commentMapper,
                          MessageMapper messageMapper,
                          TradeOrderMapper orderMapper,
                          ReportMapper reportMapper,
                          Path uploadRoot,
                          Clock clock) {
        this.productMapper = productMapper;
        this.imageMapper = imageMapper;
        this.favoriteMapper = favoriteMapper;
        this.commentMapper = commentMapper;
        this.messageMapper = messageMapper;
        this.orderMapper = orderMapper;
        this.reportMapper = reportMapper;
        this.uploadRoot = uploadRoot.toAbsolutePath().normalize();
        this.clock = clock;
    }

    @Transactional
    public CleanupResult cleanupClosedProducts(int batchSize, Duration minAge) {
        int safeBatchSize = Math.max(1, Math.min(batchSize, 500));
        LocalDateTime before = LocalDateTime.now(clock).minus(minAge);
        List<Product> products = productMapper.selectList(new LambdaQueryWrapper<Product>()
                .in(Product::getStatus, CLEANABLE_STATUS)
                .lt(Product::getUpdatedAt, before)
                .orderByAsc(Product::getUpdatedAt)
                .last("LIMIT " + safeBatchSize));
        if (products.isEmpty()) {
            return new CleanupResult(0, 0);
        }

        List<Long> productIds = products.stream().map(Product::getId).filter(Objects::nonNull).toList();
        List<ProductImage> images = imageMapper.selectList(new LambdaQueryWrapper<ProductImage>()
                .in(ProductImage::getProductId, productIds));

        favoriteMapper.delete(new LambdaQueryWrapper<Favorite>().in(Favorite::getProductId, productIds));
        commentMapper.delete(new LambdaQueryWrapper<ProductComment>().in(ProductComment::getProductId, productIds));
        messageMapper.delete(new LambdaQueryWrapper<Message>().in(Message::getProductId, productIds));
        orderMapper.delete(new LambdaQueryWrapper<TradeOrder>().in(TradeOrder::getProductId, productIds));
        reportMapper.delete(new LambdaQueryWrapper<Report>().in(Report::getProductId, productIds));
        imageMapper.delete(new LambdaQueryWrapper<ProductImage>().in(ProductImage::getProductId, productIds));
        productMapper.deleteBatchIds(productIds);

        int deletedFiles = 0;
        for (ProductImage image : images) {
            if (deleteUploadFile(image.getImageUrl())) {
                deletedFiles++;
            }
        }
        log.warn("resource fallback cleanup deleted products={}, files={}", productIds.size(), deletedFiles);
        return new CleanupResult(productIds.size(), deletedFiles);
    }

    private boolean deleteUploadFile(String imageUrl) {
        if (imageUrl == null || imageUrl.isBlank()) return false;
        String path = imageUrl;
        int marker = path.indexOf("/uploads/");
        if (marker >= 0) {
            path = path.substring(marker + "/uploads/".length());
        } else if (path.startsWith("/")) {
            path = path.substring(1);
        }
        Path file = uploadRoot.resolve(path).normalize();
        if (!file.startsWith(uploadRoot)) {
            log.warn("skip suspicious upload path: {}", imageUrl);
            return false;
        }
        try {
            boolean deleted = Files.deleteIfExists(file);
            deleteEmptyParent(file.getParent());
            return deleted;
        } catch (IOException e) {
            log.warn("failed to delete upload file: {}", file, e);
            return false;
        }
    }

    private void deleteEmptyParent(Path dir) {
        if (dir == null || dir.equals(uploadRoot)) return;
        try {
            Files.deleteIfExists(dir);
        } catch (IOException ignored) {
            // Directory is not empty or cannot be removed; leaving it is harmless.
        }
    }

    public record CleanupResult(int deletedProducts, int deletedFiles) {
    }
}
