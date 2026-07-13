package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.ProductCommentDTO;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.ProductComment;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.ProductCommentMapper;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.util.HtmlUtils;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class ProductCommentService {

    private final ProductCommentMapper commentMapper;
    private final ProductMapper productMapper;
    private final UserMapper userMapper;

    public ProductCommentService(ProductCommentMapper commentMapper,
                                 ProductMapper productMapper,
                                 UserMapper userMapper) {
        this.commentMapper = commentMapper;
        this.productMapper = productMapper;
        this.userMapper = userMapper;
    }

    public List<ProductCommentDTO.Item> list(Long productId) {
        ensureProduct(productId);
        List<ProductComment> comments = commentMapper.selectList(new LambdaQueryWrapper<ProductComment>()
                .eq(ProductComment::getProductId, productId)
                .orderByAsc(ProductComment::getCreatedAt)
                .orderByAsc(ProductComment::getId));
        return enrich(comments);
    }

    public ProductCommentDTO.Item create(Long productId, Long userId, ProductCommentDTO.CreateReq req) {
        ensureProduct(productId);
        String content = req.getContent() == null ? "" : req.getContent().trim();
        if (content.isBlank()) {
            throw new BusinessException("留言内容不能为空");
        }
        if (content.length() > 300) {
            throw new BusinessException("留言最多 300 字");
        }
        ProductComment comment = new ProductComment();
        comment.setProductId(productId);
        comment.setUserId(userId);
        comment.setContent(HtmlUtils.htmlEscape(content));
        commentMapper.insert(comment);
        ProductComment saved = commentMapper.selectById(comment.getId());
        return enrich(List.of(saved)).get(0);
    }

    private void ensureProduct(Long productId) {
        Product product = productMapper.selectById(productId);
        if (product == null) {
            throw new BusinessException("商品不存在");
        }
    }

    private List<ProductCommentDTO.Item> enrich(List<ProductComment> comments) {
        if (comments.isEmpty()) return new ArrayList<>();
        Set<Long> userIds = comments.stream().map(ProductComment::getUserId).collect(Collectors.toSet());
        Map<Long, User> userMap = userMapper.selectBatchIds(userIds).stream()
                .collect(Collectors.toMap(User::getId, u -> u));
        return comments.stream().map(c -> {
            ProductCommentDTO.Item item = new ProductCommentDTO.Item();
            item.setId(c.getId());
            item.setProductId(c.getProductId());
            item.setUserId(c.getUserId());
            item.setContent(c.getContent());
            item.setCreatedAt(c.getCreatedAt());
            User user = userMap.get(c.getUserId());
            if (user != null) {
                item.setUserName(user.getName());
                item.setStudentNoSuffix(studentNoSuffix(user.getStudentNo()));
            }
            return item;
        }).toList();
    }

    private String studentNoSuffix(String studentNo) {
        if (studentNo == null || studentNo.isBlank()) return "";
        int start = Math.max(0, studentNo.length() - 4);
        return studentNo.substring(start);
    }
}
