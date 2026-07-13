package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import lombok.Data;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.List;

public class ProductDTO {

    @Data
    public static class CreateReq {
        @NotBlank
        private String title;
        private String description;
        @NotNull
        private Long categoryId;
        @NotNull
        @Positive
        private BigDecimal price;
        private BigDecimal originalPrice;
        private String conditionLevel;
        private String tradeLocation;
        private List<String> images;
    }

    @Data
    public static class UpdateReq {
        private String title;
        private String description;
        private Long categoryId;
        private BigDecimal price;
        private BigDecimal originalPrice;
        private String conditionLevel;
        private String tradeLocation;
        private String status;
        private List<String> images;
    }

    @Data
    public static class ListQuery {
        private String keyword;
        private Long categoryId;
        private BigDecimal minPrice;
        private BigDecimal maxPrice;
        private String conditionLevel;
        private String campus;
        private String sortBy;
        private String status;
        private Long sellerId;
        private Boolean includeAllStatus;
        private Integer page = 1;
        private Integer size = 12;
    }

    @Data
    public static class Item {
        private Long id;
        private String title;
        private String description;
        private BigDecimal price;
        private BigDecimal originalPrice;
        private String conditionLevel;
        private String tradeLocation;
        private String status;
        private Integer viewCount;
        private LocalDateTime createdAt;
        private Long categoryId;
        private String categoryName;
        private Long sellerId;
        private String sellerName;
        private String sellerCampus;
        private List<String> images;
        private String cover;
        private Boolean favorited;
    }
}
