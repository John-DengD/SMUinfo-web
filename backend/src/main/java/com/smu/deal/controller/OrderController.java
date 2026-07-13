package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.OrderDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.OrderService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/orders")
public class OrderController {

    private final OrderService orderService;

    public OrderController(OrderService orderService) {
        this.orderService = orderService;
    }

    @PostMapping
    public R<OrderDTO.Item> create(@RequestBody @Valid OrderDTO.CreateReq req) {
        return R.ok(orderService.create(CurrentUser.requireId(), req));
    }

    @GetMapping
    public R<List<OrderDTO.Item>> myOrders(@RequestParam(required = false) String role) {
        return R.ok(orderService.myOrders(CurrentUser.requireId(), role));
    }

    @PutMapping("/{id}/confirm")
    public R<OrderDTO.Item> confirm(@PathVariable Long id) {
        return R.ok(orderService.confirm(id, CurrentUser.requireId()));
    }

    @PutMapping("/{id}/finish")
    public R<OrderDTO.Item> finish(@PathVariable Long id) {
        return R.ok(orderService.finish(id, CurrentUser.requireId()));
    }

    @PutMapping("/{id}/cancel")
    public R<OrderDTO.Item> cancel(@PathVariable Long id) {
        return R.ok(orderService.cancel(id, CurrentUser.requireId()));
    }
}
