package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.entity.Category;
import com.smu.deal.service.CategoryService;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/categories")
public class CategoryController {

    private final CategoryService service;

    public CategoryController(CategoryService service) {
        this.service = service;
    }

    @GetMapping
    public R<List<Category>> list() {
        return R.ok(service.list());
    }
}
