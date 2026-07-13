package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.smu.deal.common.BusinessException;
import com.smu.deal.common.PageResult;
import com.smu.deal.dto.ProductDTO;
import com.smu.deal.entity.Category;
import com.smu.deal.entity.Favorite;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.ProductImage;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.CategoryMapper;
import com.smu.deal.mapper.FavoriteMapper;
import com.smu.deal.mapper.ProductImageMapper;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.util.HtmlUtils;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class ProductService {

    private final ProductMapper productMapper;
    private final ProductImageMapper imageMapper;
    private final CategoryMapper categoryMapper;
    private final UserMapper userMapper;
    private final FavoriteMapper favoriteMapper;

    public ProductService(ProductMapper productMapper, ProductImageMapper imageMapper,
                          CategoryMapper categoryMapper, UserMapper userMapper,
                          FavoriteMapper favoriteMapper) {
        this.productMapper = productMapper;
        this.imageMapper = imageMapper;
        this.categoryMapper = categoryMapper;
        this.userMapper = userMapper;
        this.favoriteMapper = favoriteMapper;
    }

    public PageResult<ProductDTO.Item> list(ProductDTO.ListQuery q, Long currentUserId) {
        Page<Product> page = new Page<>(
                q.getPage() == null ? 1 : q.getPage(),
                q.getSize() == null ? 12 : q.getSize());
        LambdaQueryWrapper<Product> w = new LambdaQueryWrapper<>();
        if (q.getKeyword() != null && !q.getKeyword().isBlank()) {
            w.and(x -> x.like(Product::getTitle, q.getKeyword())
                    .or().like(Product::getDescription, q.getKeyword()));
        }
        if (q.getCategoryId() != null) {
            w.eq(Product::getCategoryId, q.getCategoryId());
        }
        if (q.getMinPrice() != null) {
            w.ge(Product::getPrice, q.getMinPrice());
        }
        if (q.getMaxPrice() != null) {
            w.le(Product::getPrice, q.getMaxPrice());
        }
        if (q.getConditionLevel() != null && !q.getConditionLevel().isBlank()) {
            w.eq(Product::getConditionLevel, q.getConditionLevel());
        }
        if (q.getCampus() != null && !q.getCampus().isBlank()) {
            w.like(Product::getTradeLocation, q.getCampus());
        }
        if (q.getSellerId() != null) {
            w.eq(Product::getSellerId, q.getSellerId());
        }
        if (q.getStatus() != null && !q.getStatus().isBlank()) {
            w.eq(Product::getStatus, q.getStatus());
        } else if (q.getSellerId() == null && !Boolean.TRUE.equals(q.getIncludeAllStatus())) {
            w.in(Product::getStatus, List.of("ON_SALE", "RESERVED"));
        }
        switch (q.getSortBy() == null ? "" : q.getSortBy()) {
            case "price_asc" -> w.orderByAsc(Product::getPrice);
            case "price_desc" -> w.orderByDesc(Product::getPrice);
            case "hot" -> w.orderByDesc(Product::getViewCount);
            default -> w.orderByDesc(Product::getCreatedAt);
        }
        Page<Product> result = productMapper.selectPage(page, w);
        List<ProductDTO.Item> items = enrich(result.getRecords(), currentUserId);
        return PageResult.of(result.getTotal(), items);
    }

    public ProductDTO.Item detail(Long id, Long currentUserId) {
        Product p = productMapper.selectById(id);
        if (p == null) {
            throw new BusinessException("商品不存在");
        }
        p.setViewCount((p.getViewCount() == null ? 0 : p.getViewCount()) + 1);
        productMapper.updateById(p);
        List<ProductDTO.Item> items = enrich(List.of(p), currentUserId);
        return items.get(0);
    }

    @Transactional
    public ProductDTO.Item create(Long sellerId, ProductDTO.CreateReq req) {
        Product p = new Product();
        p.setSellerId(sellerId);
        p.setCategoryId(req.getCategoryId());
        p.setTitle(HtmlUtils.htmlEscape(req.getTitle()));
        p.setDescription(req.getDescription() == null ? null : HtmlUtils.htmlEscape(req.getDescription()));
        p.setPrice(req.getPrice());
        p.setOriginalPrice(req.getOriginalPrice());
        p.setConditionLevel(req.getConditionLevel());
        p.setTradeLocation(req.getTradeLocation());
        p.setStatus("ON_SALE");
        p.setViewCount(0);
        productMapper.insert(p);
        saveImages(p.getId(), req.getImages());
        return detail(p.getId(), sellerId);
    }

    @Transactional
    public ProductDTO.Item update(Long productId, Long currentUserId, boolean isAdmin, ProductDTO.UpdateReq req) {
        Product p = productMapper.selectById(productId);
        if (p == null) {
            throw new BusinessException("商品不存在");
        }
        if (!isAdmin && !p.getSellerId().equals(currentUserId)) {
            throw new BusinessException(403, "无权操作");
        }
        if (req.getTitle() != null) p.setTitle(HtmlUtils.htmlEscape(req.getTitle()));
        if (req.getDescription() != null) p.setDescription(HtmlUtils.htmlEscape(req.getDescription()));
        if (req.getCategoryId() != null) p.setCategoryId(req.getCategoryId());
        if (req.getPrice() != null) p.setPrice(req.getPrice());
        if (req.getOriginalPrice() != null) p.setOriginalPrice(req.getOriginalPrice());
        if (req.getConditionLevel() != null) p.setConditionLevel(req.getConditionLevel());
        if (req.getTradeLocation() != null) p.setTradeLocation(req.getTradeLocation());
        if (req.getStatus() != null) p.setStatus(req.getStatus());
        productMapper.updateById(p);
        if (req.getImages() != null) {
            imageMapper.delete(new LambdaQueryWrapper<ProductImage>().eq(ProductImage::getProductId, productId));
            saveImages(productId, req.getImages());
        }
        return detail(productId, currentUserId);
    }

    public void changeStatus(Long productId, Long currentUserId, boolean isAdmin, String status) {
        Product p = productMapper.selectById(productId);
        if (p == null) throw new BusinessException("商品不存在");
        if (!isAdmin && !p.getSellerId().equals(currentUserId)) {
            throw new BusinessException(403, "无权操作");
        }
        p.setStatus(status);
        productMapper.updateById(p);
    }

    private void saveImages(Long productId, List<String> urls) {
        if (urls == null || urls.isEmpty()) return;
        int i = 0;
        for (String url : urls) {
            ProductImage img = new ProductImage();
            img.setProductId(productId);
            img.setImageUrl(url);
            img.setSortOrder(i++);
            imageMapper.insert(img);
        }
    }

    private List<ProductDTO.Item> enrich(List<Product> list, Long currentUserId) {
        if (list.isEmpty()) return new ArrayList<>();
        Set<Long> productIds = list.stream().map(Product::getId).collect(Collectors.toSet());
        Set<Long> sellerIds = list.stream().map(Product::getSellerId).collect(Collectors.toSet());
        Set<Long> categoryIds = list.stream().map(Product::getCategoryId).collect(Collectors.toSet());

        Map<Long, List<String>> imageMap = new HashMap<>();
        List<ProductImage> imgs = imageMapper.selectList(new LambdaQueryWrapper<ProductImage>()
                .in(ProductImage::getProductId, productIds)
                .orderByAsc(ProductImage::getSortOrder));
        for (ProductImage img : imgs) {
            imageMap.computeIfAbsent(img.getProductId(), k -> new ArrayList<>()).add(img.getImageUrl());
        }
        Map<Long, User> userMap = userMapper.selectBatchIds(sellerIds).stream()
                .collect(Collectors.toMap(User::getId, u -> u));
        Map<Long, Category> categoryMap = categoryMapper.selectBatchIds(categoryIds).stream()
                .collect(Collectors.toMap(Category::getId, c -> c));

        Set<Long> favorited = new HashSet<>();
        if (currentUserId != null) {
            List<Favorite> fav = favoriteMapper.selectList(new LambdaQueryWrapper<Favorite>()
                    .eq(Favorite::getUserId, currentUserId)
                    .in(Favorite::getProductId, productIds));
            favorited = fav.stream().map(Favorite::getProductId).collect(Collectors.toSet());
        }

        List<ProductDTO.Item> result = new ArrayList<>();
        for (Product p : list) {
            ProductDTO.Item it = new ProductDTO.Item();
            it.setId(p.getId());
            it.setTitle(p.getTitle());
            it.setDescription(p.getDescription());
            it.setPrice(p.getPrice());
            it.setOriginalPrice(p.getOriginalPrice());
            it.setConditionLevel(p.getConditionLevel());
            it.setTradeLocation(p.getTradeLocation());
            it.setStatus(p.getStatus());
            it.setViewCount(p.getViewCount());
            it.setCreatedAt(p.getCreatedAt());
            it.setCategoryId(p.getCategoryId());
            it.setSellerId(p.getSellerId());
            Category c = categoryMap.get(p.getCategoryId());
            if (c != null) it.setCategoryName(c.getName());
            User u = userMap.get(p.getSellerId());
            if (u != null) {
                it.setSellerName(u.getName());
                it.setSellerCampus(u.getCampus());
            }
            List<String> imgList = imageMap.getOrDefault(p.getId(), Collections.emptyList());
            it.setImages(imgList);
            it.setCover(imgList.isEmpty() ? null : imgList.get(0));
            it.setFavorited(favorited.contains(p.getId()));
            result.add(it);
        }
        return result;
    }
}
