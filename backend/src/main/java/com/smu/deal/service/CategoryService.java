package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.entity.Category;
import com.smu.deal.mapper.CategoryMapper;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class CategoryService {

    private final CategoryMapper categoryMapper;

    public CategoryService(CategoryMapper categoryMapper) {
        this.categoryMapper = categoryMapper;
    }

    public List<Category> list() {
        return categoryMapper.selectList(new LambdaQueryWrapper<Category>()
                .eq(Category::getStatus, "ACTIVE")
                .orderByAsc(Category::getSortOrder));
    }

    public Category create(Category c) {
        c.setId(null);
        if (c.getStatus() == null) c.setStatus("ACTIVE");
        if (c.getSortOrder() == null) c.setSortOrder(0);
        categoryMapper.insert(c);
        return c;
    }

    public Category update(Long id, Category c) {
        c.setId(id);
        categoryMapper.updateById(c);
        return categoryMapper.selectById(id);
    }

    public void delete(Long id) {
        categoryMapper.deleteById(id);
    }
}
