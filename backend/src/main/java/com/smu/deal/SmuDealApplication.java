package com.smu.deal;

import org.mybatis.spring.annotation.MapperScan;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
@MapperScan("com.smu.deal.mapper")
public class SmuDealApplication {
    public static void main(String[] args) {
        SpringApplication.run(SmuDealApplication.class, args);
    }
}
