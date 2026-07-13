package com.smu.deal.security;

import lombok.AllArgsConstructor;
import lombok.Data;
import org.springframework.security.core.Authentication;
import org.springframework.security.core.context.SecurityContextHolder;

@Data
@AllArgsConstructor
public class CurrentUser {
    private Long id;
    private String role;

    public static CurrentUser get() {
        Authentication a = SecurityContextHolder.getContext().getAuthentication();
        if (a == null || !(a.getPrincipal() instanceof CurrentUser u)) {
            return null;
        }
        return u;
    }

    public static Long requireId() {
        CurrentUser u = get();
        if (u == null) {
            throw new com.smu.deal.common.BusinessException(401, "未登录");
        }
        return u.getId();
    }

    public boolean isAdmin() {
        return "ADMIN".equalsIgnoreCase(role);
    }
}
