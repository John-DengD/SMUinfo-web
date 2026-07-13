package com.smu.deal.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import lombok.Data;

public class AuthDTO {

    @Data
    public static class RegisterReq {
        @NotBlank(message = "请输入姓名")
        @Size(min = 2, max = 20, message = "姓名长度需为 2-20 个字符")
        @Pattern(regexp = "^[\\p{IsHan}A-Za-z·.\\- ]+$", message = "姓名只能包含中文、英文字母、空格、点号、中点或连字符，不能包含数字")
        private String name;
        @NotBlank(message = "请输入学号")
        @Pattern(regexp = "^\\d{12}$", message = "学号必须是 12 位纯数字")
        private String studentNo;
        @NotBlank(message = "请输入密码")
        @Size(min = 6, max = 64, message = "密码长度需为 6-64 位")
        private String password;
        private String phone;
        private String college;
        private String campus;
    }

    @Data
    public static class LoginReq {
        @NotBlank
        private String studentNo;
        @NotBlank
        private String password;
    }

    @Data
    public static class LoginResp {
        private String token;
        private UserInfo user;
    }

    @Data
    public static class UserInfo {
        private Long id;
        private String name;
        private String studentNo;
        private String phone;
        private String college;
        private String campus;
        private String avatar;
        private String role;
        private String status;
    }
}
