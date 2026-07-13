package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.AnnouncementDTO;
import com.smu.deal.service.AnnouncementService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/announcements")
public class AnnouncementController {

    private final AnnouncementService announcementService;

    public AnnouncementController(AnnouncementService announcementService) {
        this.announcementService = announcementService;
    }

    @GetMapping("/active")
    public R<AnnouncementDTO.Item> active() {
        return R.ok(announcementService.active());
    }
}
