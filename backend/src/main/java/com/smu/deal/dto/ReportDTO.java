package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.Data;

import java.time.LocalDateTime;

public class ReportDTO {

    @Data
    public static class CreateReq {
        @NotNull
        private Long productId;
        @NotBlank
        private String reason;
    }

    @Data
    public static class HandleReq {
        private String status;
        private String adminRemark;
    }

    @Data
    public static class Item {
        private Long id;
        private Long reporterId;
        private String reporterName;
        private Long productId;
        private String productTitle;
        private String reason;
        private String status;
        private String adminRemark;
        private LocalDateTime createdAt;
    }
}
