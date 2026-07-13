package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.time.LocalDateTime;

public class AnnouncementDTO {

    @Data
    public static class SaveReq {
        @NotBlank
        @Size(max = 80, message = "公告标题不能超过 80 字")
        private String title;

        @NotBlank
        @Size(max = 500, message = "公告内容不能超过 500 字")
        private String content;

        private String status;
    }

    @Data
    public static class Item {
        private Long id;
        private String title;
        private String content;
        private String status;
        private Long createdBy;
        private LocalDateTime createdAt;
        private LocalDateTime updatedAt;
    }
}
