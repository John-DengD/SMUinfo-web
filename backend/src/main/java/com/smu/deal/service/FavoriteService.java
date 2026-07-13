package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
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

import java.util.*;
import java.util.stream.Collectors;

@Service
public class FavoriteService {

    private final FavoriteMapper favoriteMapper;
    private final ProductMapper productMapper;
    private final ProductImageMapper imageMapper;
    private final CategoryMapper categoryMapper;
    private final UserMapper userMapper;

    public FavoriteService(FavoriteMapper favoriteMapper, ProductMapper productMapper,
                           ProductImageMapper imageMapper, CategoryMapper categoryMapper,
                           UserMapper userMapper) {
        this.favoriteMapper = favoriteMapper;
        this.productMapper = productMapper;
        this.imageMapper = imageMapper;
        this.categoryMapper = categoryMapper;
        this.userMapper = userMapper;
    }

    public void add(Long userId, Long productId) {
        Product p = productMapper.selectById(productId);
        if (p == null) throw new BusinessException("商品不存在");
        Long count = favoriteMapper.selectCount(new LambdaQueryWrapper<Favorite>()
                .eq(Favorite::getUserId, userId)
                .eq(Favorite::getProductId, productId));
        if (count > 0) return;
        Favorite f = new Favorite();
        f.setUserId(userId);
        f.setProductId(productId);
        favoriteMapper.insert(f);
    }

    public void remove(Long userId, Long productId) {
        favoriteMapper.delete(new LambdaQueryWrapper<Favorite>()
                .eq(Favorite::getUserId, userId)
                .eq(Favorite::getProductId, productId));
    }

    public PageResult<ProductDTO.Item> myFavorites(Long userId) {
        List<Favorite> favorites = favoriteMapper.selectList(new LambdaQueryWrapper<Favorite>()
                .eq(Favorite::getUserId, userId)
                .orderByDesc(Favorite::getCreatedAt));
        if (favorites.isEmpty()) {
            return PageResult.of(0, new ArrayList<>());
        }
        List<Long> ids = favorites.stream().map(Favorite::getProductId).toList();
        List<Product> products = productMapper.selectBatchIds(ids);
        Map<Long, Product> pmap = products.stream().collect(Collectors.toMap(Product::getId, p -> p));

        Set<Long> sellerIds = products.stream().map(Product::getSellerId).collect(Collectors.toSet());
        Set<Long> categoryIds = products.stream().map(Product::getCategoryId).collect(Collectors.toSet());

        Map<Long, List<String>> imageMap = new HashMap<>();
        if (!ids.isEmpty()) {
            for (ProductImage img : imageMapper.selectList(new LambdaQueryWrapper<ProductImage>()
                    .in(ProductImage::getProductId, ids)
                    .orderByAsc(ProductImage::getSortOrder))) {
                imageMap.computeIfAbsent(img.getProductId(), k -> new ArrayList<>()).add(img.getImageUrl());
            }
        }
        Map<Long, User> userMap = sellerIds.isEmpty() ? Map.of() :
                userMapper.selectBatchIds(sellerIds).stream().collect(Collectors.toMap(User::getId, u -> u));
        Map<Long, Category> categoryMap = categoryIds.isEmpty() ? Map.of() :
                categoryMapper.selectBatchIds(categoryIds).stream().collect(Collectors.toMap(Category::getId, c -> c));

        List<ProductDTO.Item> items = new ArrayList<>();
        for (Favorite f : favorites) {
            Product p = pmap.get(f.getProductId());
            if (p == null) continue;
            ProductDTO.Item it = new ProductDTO.Item();
            it.setId(p.getId());
            it.setTitle(p.getTitle());
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
            List<String> imgs = imageMap.getOrDefault(p.getId(), Collections.emptyList());
            it.setImages(imgs);
            it.setCover(imgs.isEmpty() ? null : imgs.get(0));
            it.setFavorited(true);
            items.add(it);
        }
        return PageResult.of(items.size(), items);
    }
}
