package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;
import lombok.Data;

import java.time.LocalDateTime;
import java.util.List;

public class LostFoundDTO {

    @Data
    public static class CreateReq {
        @NotBlank
        private String type;
        @NotBlank
        @Size(max = 80)
        private String title;
        @NotBlank
        @Size(max = 1000)
        private String description;
        @Size(max = 128)
        private String location;
        @Size(max = 128)
        private String contact;
        private LocalDateTime eventTime;
        private List<String> images;
    }

    @Data
    public static class ListQuery {
        private String type;
        private String keyword;
        private Integer page = 1;
        private Integer size = 12;
    }

    @Data
    public static class Item {
        private Long id;
        private Long userId;
        private String userName;
        private String userCampus;
        private String type;
        private String typeText;
        private String title;
        private String description;
        private String location;
        private String contact;
        private String status;
        private Integer viewCount;
        private LocalDateTime eventTime;
        private LocalDateTime createdAt;
        private List<String> images;
        private String cover;
    }
}
