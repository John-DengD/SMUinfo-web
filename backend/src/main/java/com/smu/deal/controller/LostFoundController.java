package com.smu.deal.controller;

import com.smu.deal.common.PageResult;
import com.smu.deal.common.R;
import com.smu.deal.dto.LostFoundDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.LostFoundService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/lost-found")
public class LostFoundController {

    private final LostFoundService lostFoundService;

    public LostFoundController(LostFoundService lostFoundService) {
        this.lostFoundService = lostFoundService;
    }

    @GetMapping
    public R<PageResult<LostFoundDTO.Item>> list(LostFoundDTO.ListQuery query) {
        return R.ok(lostFoundService.list(query));
    }

    @GetMapping("/{id}")
    public R<LostFoundDTO.Item> detail(@PathVariable Long id) {
        return R.ok(lostFoundService.detail(id));
    }

    @PostMapping
    public R<LostFoundDTO.Item> create(@RequestBody @Valid LostFoundDTO.CreateReq req) {
        return R.ok(lostFoundService.create(CurrentUser.requireId(), req));
    }

    @DeleteMapping("/{id}")
    public R<Void> close(@PathVariable Long id) {
        CurrentUser user = CurrentUser.get();
        if (user == null) {
            return R.fail(401, "未登录");
        }
        lostFoundService.close(id, user.getId(), user.isAdmin());
        return R.ok();
    }
}
