package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.TransitDTO;
import com.smu.deal.service.TransitService;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/transit")
public class TransitController {

    private final TransitService transitService;

    public TransitController(TransitService transitService) {
        this.transitService = transitService;
    }

    @GetMapping("/next")
    public R<TransitDTO.NextResp> next(@RequestParam(required = false) String line,
                                       @RequestParam(required = false) String station,
                                       @RequestParam(required = false) String direction) {
        return R.ok(transitService.next(line, station, direction));
    }
}
