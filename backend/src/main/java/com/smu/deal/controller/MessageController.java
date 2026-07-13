package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.MessageDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.MessageService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/messages")
public class MessageController {

    private final MessageService messageService;

    public MessageController(MessageService messageService) {
        this.messageService = messageService;
    }

    @PostMapping
    public R<MessageDTO.Item> send(@RequestBody @Valid MessageDTO.SendReq req) {
        return R.ok(messageService.send(CurrentUser.requireId(), req));
    }

    @GetMapping
    public R<List<MessageDTO.Conversation>> conversations() {
        return R.ok(messageService.conversations(CurrentUser.requireId()));
    }

    @GetMapping("/conversation/{userId}")
    public R<List<MessageDTO.Item>> conversation(@PathVariable Long userId) {
        return R.ok(messageService.conversation(CurrentUser.requireId(), userId));
    }

    @GetMapping("/unread-count")
    public R<Map<String, Integer>> unreadCount() {
        int count = messageService.unreadCount(CurrentUser.requireId());
        return R.ok(Map.of("count", count));
    }
}
