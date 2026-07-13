package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.FeedbackDTO;
import com.smu.deal.entity.Feedback;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.FeedbackMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.util.HtmlUtils;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class FeedbackService {

    private final FeedbackMapper feedbackMapper;
    private final UserMapper userMapper;

    public FeedbackService(FeedbackMapper feedbackMapper, UserMapper userMapper) {
        this.feedbackMapper = feedbackMapper;
        this.userMapper = userMapper;
    }

    public void create(Long userId, FeedbackDTO.CreateReq req) {
        if (req.getContent() == null || req.getContent().isBlank()) {
            throw new BusinessException("意见内容不能为空");
        }
        Feedback f = new Feedback();
        f.setUserId(userId);
        f.setCategory(req.getCategory() == null || req.getCategory().isBlank() ? "其他" : req.getCategory());
        f.setContent(HtmlUtils.htmlEscape(req.getContent()));
        f.setContact(req.getContact() == null ? null : HtmlUtils.htmlEscape(req.getContact()));
        f.setStatus("PENDING");
        feedbackMapper.insert(f);
    }

    public List<FeedbackDTO.Item> listMine(Long userId) {
        LambdaQueryWrapper<Feedback> w = new LambdaQueryWrapper<>();
        w.eq(Feedback::getUserId, userId).orderByDesc(Feedback::getCreatedAt);
        List<Feedback> list = feedbackMapper.selectList(w);
        return list.stream().map(this::toItem).toList();
    }

    public List<FeedbackDTO.Item> listAll(String status) {
        LambdaQueryWrapper<Feedback> w = new LambdaQueryWrapper<>();
        if (status != null && !status.isBlank()) {
            w.eq(Feedback::getStatus, status);
        }
        w.orderByDesc(Feedback::getCreatedAt);
        List<Feedback> list = feedbackMapper.selectList(w);
        if (list.isEmpty()) return new ArrayList<>();
        Set<Long> userIds = list.stream().map(Feedback::getUserId).filter(Objects::nonNull).collect(Collectors.toSet());
        Map<Long, User> umap = userIds.isEmpty() ? Collections.emptyMap() :
                userMapper.selectBatchIds(userIds).stream().collect(Collectors.toMap(User::getId, u -> u));
        return list.stream().map(f -> {
            FeedbackDTO.Item it = toItem(f);
            User u = umap.get(f.getUserId());
            if (u != null) it.setUserName(u.getName());
            return it;
        }).toList();
    }

    public void reply(Long id, FeedbackDTO.ReplyReq req) {
        Feedback f = feedbackMapper.selectById(id);
        if (f == null) throw new BusinessException("意见不存在");
        if (req.getStatus() != null) f.setStatus(req.getStatus());
        if (req.getAdminReply() != null) f.setAdminReply(HtmlUtils.htmlEscape(req.getAdminReply()));
        feedbackMapper.updateById(f);
    }

    private FeedbackDTO.Item toItem(Feedback f) {
        FeedbackDTO.Item it = new FeedbackDTO.Item();
        it.setId(f.getId());
        it.setUserId(f.getUserId());
        it.setCategory(f.getCategory());
        it.setContent(f.getContent());
        it.setContact(f.getContact());
        it.setStatus(f.getStatus());
        it.setAdminReply(f.getAdminReply());
        it.setCreatedAt(f.getCreatedAt());
        return it;
    }
}
