package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.smu.deal.common.BusinessException;
import com.smu.deal.common.PageResult;
import com.smu.deal.dto.LostFoundDTO;
import com.smu.deal.entity.LostFound;
import com.smu.deal.entity.LostFoundImage;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.LostFoundImageMapper;
import com.smu.deal.mapper.LostFoundMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.util.HtmlUtils;

import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

@Service
public class LostFoundService {

    private static final Set<String> TYPES = Set.of("LOST", "FOUND");

    private final LostFoundMapper lostFoundMapper;
    private final LostFoundImageMapper imageMapper;
    private final UserMapper userMapper;

    public LostFoundService(LostFoundMapper lostFoundMapper, LostFoundImageMapper imageMapper, UserMapper userMapper) {
        this.lostFoundMapper = lostFoundMapper;
        this.imageMapper = imageMapper;
        this.userMapper = userMapper;
    }

    public PageResult<LostFoundDTO.Item> list(LostFoundDTO.ListQuery query) {
        Page<LostFound> page = new Page<>(
                query.getPage() == null ? 1 : query.getPage(),
                query.getSize() == null ? 12 : query.getSize());
        LambdaQueryWrapper<LostFound> wrapper = new LambdaQueryWrapper<>();
        wrapper.eq(LostFound::getStatus, "OPEN");
        if (query.getType() != null && !query.getType().isBlank()) {
            wrapper.eq(LostFound::getType, normalizeType(query.getType()));
        }
        if (query.getKeyword() != null && !query.getKeyword().isBlank()) {
            String keyword = query.getKeyword().trim();
            wrapper.and(w -> w.like(LostFound::getTitle, keyword)
                    .or().like(LostFound::getDescription, keyword)
                    .or().like(LostFound::getLocation, keyword));
        }
        wrapper.orderByDesc(LostFound::getCreatedAt);
        Page<LostFound> result = lostFoundMapper.selectPage(page, wrapper);
        return PageResult.of(result.getTotal(), enrich(result.getRecords()));
    }

    public LostFoundDTO.Item detail(Long id) {
        LostFound row = lostFoundMapper.selectById(id);
        if (row == null || !"OPEN".equals(row.getStatus())) {
            throw new BusinessException("内容不存在");
        }
        row.setViewCount((row.getViewCount() == null ? 0 : row.getViewCount()) + 1);
        lostFoundMapper.updateById(row);
        return enrich(List.of(row)).get(0);
    }

    @Transactional
    public LostFoundDTO.Item create(Long userId, LostFoundDTO.CreateReq req) {
        LostFound row = new LostFound();
        row.setUserId(userId);
        row.setType(normalizeType(req.getType()));
        row.setTitle(HtmlUtils.htmlEscape(req.getTitle().trim()));
        row.setDescription(HtmlUtils.htmlEscape(req.getDescription().trim()));
        row.setLocation(req.getLocation() == null ? null : HtmlUtils.htmlEscape(req.getLocation().trim()));
        row.setContact(req.getContact() == null ? null : HtmlUtils.htmlEscape(req.getContact().trim()));
        row.setEventTime(req.getEventTime());
        row.setStatus("OPEN");
        row.setViewCount(0);
        lostFoundMapper.insert(row);
        saveImages(row.getId(), req.getImages());
        return detail(row.getId());
    }

    public void close(Long id, Long userId, boolean isAdmin) {
        LostFound row = lostFoundMapper.selectById(id);
        if (row == null) throw new BusinessException("内容不存在");
        if (!isAdmin && !row.getUserId().equals(userId)) {
            throw new BusinessException(403, "无权操作");
        }
        row.setStatus("CLOSED");
        lostFoundMapper.updateById(row);
    }

    private String normalizeType(String type) {
        String normalized = type == null ? "" : type.trim().toUpperCase();
        if (!TYPES.contains(normalized)) {
            throw new BusinessException("类型不正确");
        }
        return normalized;
    }

    private void saveImages(Long lostFoundId, List<String> urls) {
        if (urls == null || urls.isEmpty()) return;
        int sort = 0;
        for (String url : urls) {
            if (url == null || url.isBlank()) continue;
            LostFoundImage image = new LostFoundImage();
            image.setLostFoundId(lostFoundId);
            image.setImageUrl(url);
            image.setSortOrder(sort++);
            imageMapper.insert(image);
        }
    }

    private List<LostFoundDTO.Item> enrich(List<LostFound> rows) {
        if (rows.isEmpty()) return new ArrayList<>();
        Set<Long> ids = rows.stream().map(LostFound::getId).collect(Collectors.toSet());
        Set<Long> userIds = rows.stream().map(LostFound::getUserId).collect(Collectors.toSet());

        Map<Long, List<String>> imageMap = new HashMap<>();
        for (LostFoundImage image : imageMapper.selectList(new LambdaQueryWrapper<LostFoundImage>()
                .in(LostFoundImage::getLostFoundId, ids)
                .orderByAsc(LostFoundImage::getSortOrder))) {
            imageMap.computeIfAbsent(image.getLostFoundId(), key -> new ArrayList<>()).add(image.getImageUrl());
        }

        Map<Long, User> userMap = new HashMap<>();
        if (!userIds.isEmpty()) {
            userMap = userMapper.selectBatchIds(new HashSet<>(userIds)).stream()
                    .collect(Collectors.toMap(User::getId, user -> user));
        }

        List<LostFoundDTO.Item> result = new ArrayList<>();
        for (LostFound row : rows) {
            LostFoundDTO.Item item = new LostFoundDTO.Item();
            item.setId(row.getId());
            item.setUserId(row.getUserId());
            item.setType(row.getType());
            item.setTypeText("LOST".equals(row.getType()) ? "寻物" : "招领");
            item.setTitle(row.getTitle());
            item.setDescription(row.getDescription());
            item.setLocation(row.getLocation());
            item.setContact(row.getContact());
            item.setStatus(row.getStatus());
            item.setViewCount(row.getViewCount());
            item.setEventTime(row.getEventTime());
            item.setCreatedAt(row.getCreatedAt());
            User user = userMap.get(row.getUserId());
            if (user != null) {
                item.setUserName(user.getName());
                item.setUserCampus(user.getCampus());
            }
            List<String> images = imageMap.getOrDefault(row.getId(), Collections.emptyList());
            item.setImages(images);
            item.setCover(images.isEmpty() ? null : images.get(0));
            result.add(item);
        }
        return result;
    }
}
