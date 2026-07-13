package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.UploadService;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

import java.util.Map;

@RestController
@RequestMapping("/api/upload")
public class UploadController {

    private final UploadService uploadService;

    public UploadController(UploadService uploadService) {
        this.uploadService = uploadService;
    }

    @PostMapping("/image")
    public R<Map<String, String>> upload(@RequestParam("file") MultipartFile file) {
        CurrentUser.requireId();
        String url = uploadService.upload(file);
        return R.ok(Map.of("url", url));
    }
}
