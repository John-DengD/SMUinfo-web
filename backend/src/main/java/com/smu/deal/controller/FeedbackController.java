package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.FeedbackDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.FeedbackService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/feedback")
public class FeedbackController {

    private final FeedbackService feedbackService;

    public FeedbackController(FeedbackService feedbackService) {
        this.feedbackService = feedbackService;
    }

    @PostMapping
    public R<Void> create(@RequestBody @Valid FeedbackDTO.CreateReq req) {
        feedbackService.create(CurrentUser.requireId(), req);
        return R.ok();
    }

    @GetMapping("/mine")
    public R<List<FeedbackDTO.Item>> mine() {
        return R.ok(feedbackService.listMine(CurrentUser.requireId()));
    }
}
