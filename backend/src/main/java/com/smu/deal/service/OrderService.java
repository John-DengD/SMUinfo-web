package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.OrderDTO;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.ProductComment;
import com.smu.deal.entity.ProductImage;
import com.smu.deal.entity.TradeOrder;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.ProductCommentMapper;
import com.smu.deal.mapper.ProductImageMapper;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.TradeOrderMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.util.HtmlUtils;

import java.time.LocalDateTime;
import java.util.*;
import java.util.stream.Collectors;

@Service
public class OrderService {

    private final TradeOrderMapper orderMapper;
    private final ProductMapper productMapper;
    private final UserMapper userMapper;
    private final ProductImageMapper imageMapper;
    private final ProductCommentMapper commentMapper;

    public OrderService(TradeOrderMapper orderMapper, ProductMapper productMapper,
                        UserMapper userMapper, ProductImageMapper imageMapper,
                        ProductCommentMapper commentMapper) {
        this.orderMapper = orderMapper;
        this.productMapper = productMapper;
        this.userMapper = userMapper;
        this.imageMapper = imageMapper;
        this.commentMapper = commentMapper;
    }

    @Transactional
    public OrderDTO.Item create(Long buyerId, OrderDTO.CreateReq req) {
        Product p = productMapper.selectById(req.getProductId());
        if (p == null) throw new BusinessException("商品不存在");
        if (p.getSellerId().equals(buyerId)) throw new BusinessException("不能购买自己的商品");
        if (!"ON_SALE".equals(p.getStatus())) throw new BusinessException("商品当前不可购买");
        Long active = orderMapper.selectCount(new LambdaQueryWrapper<TradeOrder>()
                .eq(TradeOrder::getProductId, req.getProductId())
                .eq(TradeOrder::getBuyerId, buyerId)
                .in(TradeOrder::getStatus, List.of("PENDING", "RESERVED")));
        if (active != null && active > 0) {
            throw new BusinessException("你已经提交过预约申请");
        }

        TradeOrder o = new TradeOrder();
        o.setProductId(req.getProductId());
        o.setBuyerId(buyerId);
        o.setSellerId(p.getSellerId());
        o.setStatus("PENDING");
        o.setMeetLocation(req.getMeetLocation());
        o.setRemark(req.getRemark());
        orderMapper.insert(o);
        return enrich(List.of(o)).get(0);
    }

    public List<OrderDTO.Item> myOrders(Long userId, String role) {
        LambdaQueryWrapper<TradeOrder> w = new LambdaQueryWrapper<>();
        if ("seller".equalsIgnoreCase(role)) {
            w.eq(TradeOrder::getSellerId, userId);
        } else if ("buyer".equalsIgnoreCase(role)) {
            w.eq(TradeOrder::getBuyerId, userId);
        } else {
            w.and(x -> x.eq(TradeOrder::getBuyerId, userId).or().eq(TradeOrder::getSellerId, userId));
        }
        w.orderByDesc(TradeOrder::getCreatedAt);
        return enrich(orderMapper.selectList(w));
    }

    @Transactional
    public OrderDTO.Item confirm(Long orderId, Long userId) {
        TradeOrder o = getAndCheck(orderId, userId, true);
        if (!"PENDING".equals(o.getStatus())) {
            throw new BusinessException("当前状态不能确认预约");
        }
        Long reserved = orderMapper.selectCount(new LambdaQueryWrapper<TradeOrder>()
                .eq(TradeOrder::getProductId, o.getProductId())
                .ne(TradeOrder::getId, o.getId())
                .eq(TradeOrder::getStatus, "RESERVED"));
        if (reserved != null && reserved > 0) {
            throw new BusinessException("该商品已有确认的预约");
        }
        o.setStatus("RESERVED");
        orderMapper.updateById(o);

        Product p = productMapper.selectById(o.getProductId());
        if (p != null) {
            p.setStatus("RESERVED");
            productMapper.updateById(p);
        }
        orderMapper.selectList(new LambdaQueryWrapper<TradeOrder>()
                .eq(TradeOrder::getProductId, o.getProductId())
                .ne(TradeOrder::getId, o.getId())
                .eq(TradeOrder::getStatus, "PENDING")
        ).forEach(other -> {
            other.setStatus("CANCELLED");
            orderMapper.updateById(other);
        });
        addReservationComment(o);
        return enrich(List.of(o)).get(0);
    }

    @Transactional
    public OrderDTO.Item finish(Long orderId, Long userId) {
        TradeOrder o = getAndCheck(orderId, userId, false);
        if (!"RESERVED".equals(o.getStatus())) {
            throw new BusinessException("请先由卖家确认预约");
        }
        o.setStatus("COMPLETED");
        o.setCompletedAt(LocalDateTime.now());
        orderMapper.updateById(o);

        Product p = productMapper.selectById(o.getProductId());
        if (p != null) {
            p.setStatus("SOLD");
            productMapper.updateById(p);
        }
        // 把同一商品的其他活动订单标记取消
        orderMapper.selectList(new LambdaQueryWrapper<TradeOrder>()
                .eq(TradeOrder::getProductId, o.getProductId())
                .ne(TradeOrder::getId, o.getId())
                .in(TradeOrder::getStatus, List.of("PENDING", "RESERVED"))
        ).forEach(other -> {
            other.setStatus("CANCELLED");
            orderMapper.updateById(other);
        });
        return enrich(List.of(o)).get(0);
    }

    private void addReservationComment(TradeOrder order) {
        User buyer = userMapper.selectById(order.getBuyerId());
        String buyerName = buyer == null ? "同学" : buyer.getName();
        ProductComment comment = new ProductComment();
        comment.setProductId(order.getProductId());
        comment.setUserId(order.getBuyerId());
        comment.setContent(HtmlUtils.htmlEscape(buyerName + " 已预约成功了这件商品"));
        commentMapper.insert(comment);
    }

    @Transactional
    public OrderDTO.Item cancel(Long orderId, Long userId) {
        TradeOrder o = getAndCheck(orderId, userId, false);
        if ("COMPLETED".equals(o.getStatus())) {
            throw new BusinessException("已完成订单不可取消");
        }
        o.setStatus("CANCELLED");
        orderMapper.updateById(o);

        // 如果该商品没有其他活动订单，把商品状态恢复在售
        Long active = orderMapper.selectCount(new LambdaQueryWrapper<TradeOrder>()
                .eq(TradeOrder::getProductId, o.getProductId())
                .ne(TradeOrder::getId, o.getId())
                .in(TradeOrder::getStatus, List.of("PENDING", "RESERVED")));
        if (active == 0) {
            Product p = productMapper.selectById(o.getProductId());
            if (p != null && "RESERVED".equals(p.getStatus())) {
                p.setStatus("ON_SALE");
                productMapper.updateById(p);
            }
        }
        return enrich(List.of(o)).get(0);
    }

    private TradeOrder getAndCheck(Long orderId, Long userId, boolean sellerOnly) {
        TradeOrder o = orderMapper.selectById(orderId);
        if (o == null) throw new BusinessException("订单不存在");
        if (sellerOnly) {
            if (!o.getSellerId().equals(userId)) throw new BusinessException(403, "无权操作");
        } else {
            if (!o.getSellerId().equals(userId) && !o.getBuyerId().equals(userId)) {
                throw new BusinessException(403, "无权操作");
            }
        }
        return o;
    }

    private List<OrderDTO.Item> enrich(List<TradeOrder> orders) {
        if (orders.isEmpty()) return new ArrayList<>();
        Set<Long> userIds = new HashSet<>();
        Set<Long> productIds = new HashSet<>();
        for (TradeOrder o : orders) {
            userIds.add(o.getBuyerId());
            userIds.add(o.getSellerId());
            productIds.add(o.getProductId());
        }
        Map<Long, User> userMap = userMapper.selectBatchIds(userIds).stream()
                .collect(Collectors.toMap(User::getId, u -> u));
        Map<Long, Product> productMap = productMapper.selectBatchIds(productIds).stream()
                .collect(Collectors.toMap(Product::getId, p -> p));
        Map<Long, String> coverMap = new HashMap<>();
        for (ProductImage img : imageMapper.selectList(new LambdaQueryWrapper<ProductImage>()
                .in(ProductImage::getProductId, productIds)
                .orderByAsc(ProductImage::getSortOrder))) {
            coverMap.putIfAbsent(img.getProductId(), img.getImageUrl());
        }
        List<OrderDTO.Item> res = new ArrayList<>();
        for (TradeOrder o : orders) {
            OrderDTO.Item it = new OrderDTO.Item();
            it.setId(o.getId());
            it.setProductId(o.getProductId());
            it.setBuyerId(o.getBuyerId());
            it.setSellerId(o.getSellerId());
            it.setStatus(o.getStatus());
            it.setMeetLocation(o.getMeetLocation());
            it.setRemark(o.getRemark());
            it.setCreatedAt(o.getCreatedAt());
            it.setUpdatedAt(o.getUpdatedAt());
            it.setCompletedAt(o.getCompletedAt());
            User b = userMap.get(o.getBuyerId());
            if (b != null) it.setBuyerName(b.getName());
            User s = userMap.get(o.getSellerId());
            if (s != null) it.setSellerName(s.getName());
            Product p = productMap.get(o.getProductId());
            if (p != null) {
                it.setProductTitle(p.getTitle());
                it.setProductPrice(p.getPrice());
            }
            it.setProductCover(coverMap.get(o.getProductId()));
            res.add(it);
        }
        return res;
    }
}
