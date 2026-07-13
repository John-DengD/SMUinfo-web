package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Data;

import java.time.LocalDateTime;

public class MessageDTO {

    @Data
    public static class SendReq {
        @NotNull
        private Long receiverId;
        private Long productId;
        @NotBlank
        private String content;
    }

    @Data
    public static class Conversation {
        private Long peerId;
        private String peerName;
        private String lastContent;
        private LocalDateTime lastTime;
        private Integer unreadCount;
        private Long productId;
        private String productTitle;
    }

    @Data
    public static class Item {
        private Long id;
        private Long senderId;
        private Long receiverId;
        private String senderName;
        private Long productId;
        private String productTitle;
        private String content;
        private Boolean isRead;
        private LocalDateTime createdAt;
    }
}
