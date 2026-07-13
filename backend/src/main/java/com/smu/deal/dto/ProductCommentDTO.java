package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.time.LocalDateTime;

public class ProductCommentDTO {

    @Data
    public static class CreateReq {
        @NotBlank
        @Size(max = 300)
        private String content;
    }

    @Data
    public static class Item {
        private Long id;
        private Long productId;
        private Long userId;
        private String userName;
        private String studentNoSuffix;
        private String content;
        private LocalDateTime createdAt;
    }
}
