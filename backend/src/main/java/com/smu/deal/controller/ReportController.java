package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.ReportDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.ReportService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/reports")
public class ReportController {

    private final ReportService reportService;

    public ReportController(ReportService reportService) {
        this.reportService = reportService;
    }

    @PostMapping
    public R<Void> create(@RequestBody @Valid ReportDTO.CreateReq req) {
        reportService.create(CurrentUser.requireId(), req);
        return R.ok();
    }
}
