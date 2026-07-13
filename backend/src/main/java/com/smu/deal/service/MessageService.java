package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.MessageDTO;
import com.smu.deal.entity.Message;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.MessageMapper;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.util.HtmlUtils;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class MessageService {

    private final MessageMapper messageMapper;
    private final UserMapper userMapper;
    private final ProductMapper productMapper;

    public MessageService(MessageMapper messageMapper, UserMapper userMapper, ProductMapper productMapper) {
        this.messageMapper = messageMapper;
        this.userMapper = userMapper;
        this.productMapper = productMapper;
    }

    public MessageDTO.Item send(Long senderId, MessageDTO.SendReq req) {
        if (Objects.equals(senderId, req.getReceiverId())) {
            throw new BusinessException("不能给自己发消息");
        }
        User receiver = userMapper.selectById(req.getReceiverId());
        if (receiver == null) {
            throw new BusinessException("接收人不存在");
        }
        Message m = new Message();
        m.setSenderId(senderId);
        m.setReceiverId(req.getReceiverId());
        m.setProductId(req.getProductId());
        m.setContent(HtmlUtils.htmlEscape(req.getContent()));
        m.setIsRead(false);
        messageMapper.insert(m);
        return toItem(m);
    }

    public List<MessageDTO.Conversation> conversations(Long userId) {
        List<Message> all = messageMapper.selectList(new LambdaQueryWrapper<Message>()
                .and(w -> w.eq(Message::getSenderId, userId).or().eq(Message::getReceiverId, userId))
                .orderByDesc(Message::getCreatedAt));
        Map<Long, MessageDTO.Conversation> map = new LinkedHashMap<>();
        for (Message m : all) {
            Long peer = m.getSenderId().equals(userId) ? m.getReceiverId() : m.getSenderId();
            MessageDTO.Conversation c = map.get(peer);
            if (c == null) {
                c = new MessageDTO.Conversation();
                c.setPeerId(peer);
                c.setLastContent(m.getContent());
                c.setLastTime(m.getCreatedAt());
                c.setProductId(m.getProductId());
                c.setUnreadCount(0);
                map.put(peer, c);
            }
            if (Boolean.FALSE.equals(m.getIsRead()) && m.getReceiverId().equals(userId)) {
                c.setUnreadCount(c.getUnreadCount() + 1);
            }
        }
        if (map.isEmpty()) return new ArrayList<>();
        Set<Long> peerIds = map.keySet();
        Map<Long, User> userMap = userMapper.selectBatchIds(peerIds).stream()
                .collect(Collectors.toMap(User::getId, u -> u));
        Set<Long> productIds = map.values().stream().map(MessageDTO.Conversation::getProductId)
                .filter(Objects::nonNull).collect(Collectors.toSet());
        Map<Long, Product> productMap = productIds.isEmpty() ? Map.of() :
                productMapper.selectBatchIds(productIds).stream().collect(Collectors.toMap(Product::getId, p -> p));
        for (MessageDTO.Conversation c : map.values()) {
            User u = userMap.get(c.getPeerId());
            if (u != null) c.setPeerName(u.getName());
            if (c.getProductId() != null) {
                Product p = productMap.get(c.getProductId());
                if (p != null) c.setProductTitle(p.getTitle());
            }
        }
        return new ArrayList<>(map.values());
    }

    public List<MessageDTO.Item> conversation(Long userId, Long peerId) {
        List<Message> list = messageMapper.selectList(new LambdaQueryWrapper<Message>()
                .and(w -> w
                        .and(x -> x.eq(Message::getSenderId, userId).eq(Message::getReceiverId, peerId))
                        .or(x -> x.eq(Message::getSenderId, peerId).eq(Message::getReceiverId, userId)))
                .orderByAsc(Message::getCreatedAt));
        // 把对方发给我且未读的标记为已读
        for (Message m : list) {
            if (m.getReceiverId().equals(userId) && Boolean.FALSE.equals(m.getIsRead())) {
                m.setIsRead(true);
                messageMapper.updateById(m);
            }
        }
        Set<Long> senderIds = list.stream().map(Message::getSenderId).collect(Collectors.toSet());
        Map<Long, User> userMap = senderIds.isEmpty() ? Map.of() :
                userMapper.selectBatchIds(senderIds).stream().collect(Collectors.toMap(User::getId, u -> u));
        Set<Long> productIds = list.stream().map(Message::getProductId).filter(Objects::nonNull).collect(Collectors.toSet());
        Map<Long, Product> pMap = productIds.isEmpty() ? Map.of() :
                productMapper.selectBatchIds(productIds).stream().collect(Collectors.toMap(Product::getId, p -> p));
        return list.stream().map(m -> {
            MessageDTO.Item it = toItem(m);
            User u = userMap.get(m.getSenderId());
            if (u != null) it.setSenderName(u.getName());
            if (m.getProductId() != null) {
                Product p = pMap.get(m.getProductId());
                if (p != null) it.setProductTitle(p.getTitle());
            }
            return it;
        }).toList();
    }

    public int unreadCount(Long userId) {
        Long c = messageMapper.selectCount(new LambdaQueryWrapper<Message>()
                .eq(Message::getReceiverId, userId)
                .eq(Message::getIsRead, false));
        return c == null ? 0 : c.intValue();
    }

    private MessageDTO.Item toItem(Message m) {
        MessageDTO.Item it = new MessageDTO.Item();
        it.setId(m.getId());
        it.setSenderId(m.getSenderId());
        it.setReceiverId(m.getReceiverId());
        it.setProductId(m.getProductId());
        it.setContent(m.getContent());
        it.setIsRead(m.getIsRead());
        it.setCreatedAt(m.getCreatedAt());
        return it;
    }
}
