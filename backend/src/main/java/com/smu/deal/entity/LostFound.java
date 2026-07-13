package com.smu.deal.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("lost_found")
public class LostFound {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long userId;
    private String type;
    private String title;
    private String description;
    private String location;
    private String contact;
    private String status;
    private Integer viewCount;
    private LocalDateTime eventTime;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
