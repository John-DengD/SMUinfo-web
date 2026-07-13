package com.smu.deal.controller;

import com.smu.deal.common.PageResult;
import com.smu.deal.common.R;
import com.smu.deal.dto.ProductDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.FavoriteService;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/favorites")
public class FavoriteController {

    private final FavoriteService favoriteService;

    public FavoriteController(FavoriteService favoriteService) {
        this.favoriteService = favoriteService;
    }

    @PostMapping("/{productId}")
    public R<Void> add(@PathVariable Long productId) {
        favoriteService.add(CurrentUser.requireId(), productId);
        return R.ok();
    }

    @DeleteMapping("/{productId}")
    public R<Void> remove(@PathVariable Long productId) {
        favoriteService.remove(CurrentUser.requireId(), productId);
        return R.ok();
    }

    @GetMapping
    public R<PageResult<ProductDTO.Item>> list() {
        return R.ok(favoriteService.myFavorites(CurrentUser.requireId()));
    }
}
