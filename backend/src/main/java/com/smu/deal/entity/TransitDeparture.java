package com.smu.deal.entity;

import com.baomidou.mybatisplus.annotation.IdType;
import com.baomidou.mybatisplus.annotation.TableId;
import com.baomidou.mybatisplus.annotation.TableName;
import lombok.Data;

import java.time.LocalDateTime;
import java.time.LocalTime;

@Data
@TableName("transit_departure")
public class TransitDeparture {
    @TableId(type = IdType.AUTO)
    private Long id;
    private String lineCode;
    private String lineName;
    private String stationCode;
    private String stationName;
    private String directionCode;
    private String directionName;
    private String scheduleType;
    private String scheduleTypeName;
    private LocalTime departureTime;
    private String serviceType;
    private String serviceLabel;
    private Integer sortOrder;
    private String status;
    private LocalDateTime createdAt;
    private LocalDateTime updatedAt;
}
