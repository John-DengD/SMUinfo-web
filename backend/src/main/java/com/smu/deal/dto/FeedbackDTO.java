package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.time.LocalDateTime;

public class FeedbackDTO {

    @Data
    public static class CreateReq {
        private String category;
        @NotBlank
        @Size(max = 1000, message = "意见内容不能超过 1000 字")
        private String content;
        @Size(max = 64)
        private String contact;
    }

    @Data
    public static class ReplyReq {
        private String status;
        private String adminReply;
    }

    @Data
    public static class Item {
        private Long id;
        private Long userId;
        private String userName;
        private String category;
        private String content;
        private String contact;
        private String status;
        private String adminReply;
        private LocalDateTime createdAt;
    }
}
