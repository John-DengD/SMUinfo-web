package com.smu.deal.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;

@Data
@TableName("lost_found_image")
public class LostFoundImage {
    @TableId(type = IdType.AUTO)
    private Long id;
    private Long lostFoundId;
    private String imageUrl;
    private Integer sortOrder;
    private LocalDateTime createdAt;
}
