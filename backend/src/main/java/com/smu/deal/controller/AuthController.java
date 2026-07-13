package com.smu.deal.controller;

import com.smu.deal.common.R;
import com.smu.deal.dto.AuthDTO;
import com.smu.deal.security.CurrentUser;
import com.smu.deal.service.AuthService;
import jakarta.validation.Valid;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api")
public class AuthController {

    private final AuthService authService;

    public AuthController(AuthService authService) {
        this.authService = authService;
    }

    @PostMapping("/auth/register")
    public R<AuthDTO.UserInfo> register(@RequestBody @Valid AuthDTO.RegisterReq req) {
        return R.ok(authService.register(req));
    }

    @PostMapping("/auth/login")
    public R<AuthDTO.LoginResp> login(@RequestBody @Valid AuthDTO.LoginReq req) {
        return R.ok(authService.login(req));
    }

    @GetMapping("/users/me")
    public R<AuthDTO.UserInfo> me() {
        return R.ok(authService.getMe(CurrentUser.requireId()));
    }

    @PutMapping("/users/me")
    public R<AuthDTO.UserInfo> updateMe(@RequestBody AuthDTO.UserInfo req) {
        return R.ok(authService.updateMe(CurrentUser.requireId(), req));
    }
}
