package com.smu.deal.dto;

import jakarta.validation.constraints.NotNull;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;

public class OrderDTO {

    @Data
    public static class CreateReq {
        @NotNull
        private Long productId;
        private String meetLocation;
        private String remark;
    }

    @Data
    public static class Item {
        private Long id;
        private Long productId;
        private String productTitle;
        private String productCover;
        private BigDecimal productPrice;
        private Long buyerId;
        private String buyerName;
        private Long sellerId;
        private String sellerName;
        private String status;
        private String meetLocation;
        private String remark;
        private LocalDateTime createdAt;
        private LocalDateTime updatedAt;
        private LocalDateTime completedAt;
    }
}
