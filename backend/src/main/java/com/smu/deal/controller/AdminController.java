package com.smu.deal.controller;

import com.smu.deal.common.PageResult;
import com.smu.deal.common.R;
import com.smu.deal.dto.AnnouncementDTO;
import com.smu.deal.dto.AuthDTO;
import com.smu.deal.dto.FeedbackDTO;
import com.smu.deal.dto.ProductDTO;
import com.smu.deal.dto.ReportDTO;
import com.smu.deal.entity.Category;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.AdminUserService;
import com.smu.deal.service.AnnouncementService;
import com.smu.deal.service.CategoryService;
import com.smu.deal.service.FeedbackService;
import com.smu.deal.service.ProductService;
import com.smu.deal.service.ReportService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/admin")
public class AdminController {

    private final AdminUserService adminUserService;
    private final ProductService productService;
    private final CategoryService categoryService;
    private final ReportService reportService;
    private final FeedbackService feedbackService;
    private final AnnouncementService announcementService;

    public AdminController(AdminUserService adminUserService, ProductService productService,
                           CategoryService categoryService, ReportService reportService,
                           FeedbackService feedbackService, AnnouncementService announcementService) {
        this.adminUserService = adminUserService;
        this.productService = productService;
        this.categoryService = categoryService;
        this.reportService = reportService;
        this.feedbackService = feedbackService;
        this.announcementService = announcementService;
    }

    @GetMapping("/users")
    public R<PageResult<AuthDTO.UserInfo>> users(@RequestParam(required = false) String keyword,
                                                  @RequestParam(defaultValue = "1") Integer page,
                                                  @RequestParam(defaultValue = "20") Integer size) {
        return R.ok(adminUserService.list(keyword, page, size));
    }

    @PutMapping("/users/{id}/status")
    public R<Void> userStatus(@PathVariable Long id, @RequestBody Map<String, String> body) {
        adminUserService.changeStatus(id, body.get("status"));
        return R.ok();
    }

    @GetMapping("/products")
    public R<PageResult<ProductDTO.Item>> products(ProductDTO.ListQuery q) {
        q.setIncludeAllStatus(true);
        return R.ok(productService.list(q, CurrentUser.requireId()));
    }

    @PutMapping("/products/{id}/status")
    public R<Void> productStatus(@PathVariable Long id, @RequestBody Map<String, String> body) {
        CurrentUser u = CurrentUser.get();
        productService.changeStatus(id, u.getId(), true, body.get("status"));
        return R.ok();
    }

    @GetMapping("/categories")
    public R<List<Category>> categories() {
        return R.ok(categoryService.list());
    }

    @PostMapping("/categories")
    public R<Category> createCategory(@RequestBody Category c) {
        return R.ok(categoryService.create(c));
    }

    @PutMapping("/categories/{id}")
    public R<Category> updateCategory(@PathVariable Long id, @RequestBody Category c) {
        return R.ok(categoryService.update(id, c));
    }

    @DeleteMapping("/categories/{id}")
    public R<Void> deleteCategory(@PathVariable Long id) {
        categoryService.delete(id);
        return R.ok();
    }

    @GetMapping("/reports")
    public R<List<ReportDTO.Item>> reports(@RequestParam(required = false) String status) {
        return R.ok(reportService.list(status));
    }

    @PutMapping("/reports/{id}")
    public R<Void> handleReport(@PathVariable Long id, @RequestBody ReportDTO.HandleReq req) {
        reportService.handle(id, req);
        return R.ok();
    }

    @GetMapping("/feedback")
    public R<List<FeedbackDTO.Item>> feedback(@RequestParam(required = false) String status) {
        return R.ok(feedbackService.listAll(status));
    }

    @PutMapping("/feedback/{id}")
    public R<Void> replyFeedback(@PathVariable Long id, @RequestBody FeedbackDTO.ReplyReq req) {
        feedbackService.reply(id, req);
        return R.ok();
    }

    @GetMapping("/announcements")
    public R<List<AnnouncementDTO.Item>> announcements() {
        return R.ok(announcementService.list());
    }

    @PostMapping("/announcements")
    public R<AnnouncementDTO.Item> createAnnouncement(@RequestBody @Valid AnnouncementDTO.SaveReq req) {
        return R.ok(announcementService.create(CurrentUser.requireId(), req));
    }

    @PutMapping("/announcements/{id}")
    public R<AnnouncementDTO.Item> updateAnnouncement(@PathVariable Long id,
                                                      @RequestBody @Valid AnnouncementDTO.SaveReq req) {
        return R.ok(announcementService.update(id, req));
    }

    @DeleteMapping("/announcements/{id}")
    public R<Void> deleteAnnouncement(@PathVariable Long id) {
        announcementService.delete(id);
        return R.ok();
    }
}
