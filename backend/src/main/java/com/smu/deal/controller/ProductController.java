package com.smu.deal.controller;

import com.smu.deal.common.PageResult;
import com.smu.deal.common.R;
import com.smu.deal.dto.ProductCommentDTO;
import com.smu.deal.dto.ProductDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.ProductCommentService;
import com.smu.deal.service.ProductService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/products")
public class ProductController {

    private final ProductService productService;
    private final ProductCommentService commentService;

    public ProductController(ProductService productService, ProductCommentService commentService) {
        this.productService = productService;
        this.commentService = commentService;
    }

    @GetMapping
    public R<PageResult<ProductDTO.Item>> list(ProductDTO.ListQuery query) {
        CurrentUser u = CurrentUser.get();
        Long uid = u == null ? null : u.getId();
        return R.ok(productService.list(query, uid));
    }

    @GetMapping("/{id}")
    public R<ProductDTO.Item> detail(@PathVariable Long id) {
        CurrentUser u = CurrentUser.get();
        Long uid = u == null ? null : u.getId();
        return R.ok(productService.detail(id, uid));
    }

    @GetMapping("/{id}/comments")
    public R<List<ProductCommentDTO.Item>> comments(@PathVariable Long id) {
        return R.ok(commentService.list(id));
    }

    @PostMapping("/{id}/comments")
    public R<ProductCommentDTO.Item> createComment(@PathVariable Long id,
                                                   @RequestBody @Valid ProductCommentDTO.CreateReq req) {
        return R.ok(commentService.create(id, CurrentUser.requireId(), req));
    }

    @PostMapping
    public R<ProductDTO.Item> create(@RequestBody @Valid ProductDTO.CreateReq req) {
        Long uid = CurrentUser.requireId();
        return R.ok(productService.create(uid, req));
    }

    @PutMapping("/{id}")
    public R<ProductDTO.Item> update(@PathVariable Long id, @RequestBody ProductDTO.UpdateReq req) {
        CurrentUser u = CurrentUser.get();
        if (u == null) {
            return R.fail(401, "未登录");
        }
        return R.ok(productService.update(id, u.getId(), u.isAdmin(), req));
    }

    @DeleteMapping("/{id}")
    public R<Void> delete(@PathVariable Long id) {
        CurrentUser u = CurrentUser.get();
        if (u == null) {
            return R.fail(401, "未登录");
        }
        productService.changeStatus(id, u.getId(), u.isAdmin(), "OFFLINE");
        return R.ok();
    }
}
