package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.AuthDTO;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.UserMapper;
import com.smu.deal.security.JwtUtil;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

@Service
public class AuthService {

    private static final String STUDENT_NO_PATTERN = "\\d{12}";
    private static final String NAME_PATTERN = "^[\\p{IsHan}A-Za-z·.\\- ]+$";
    private static final String[] OBVIOUS_FAKE_STUDENT_NOS = {
            "000000000000",
            "111111111111",
            "123456789012"
    };

    private final UserMapper userMapper;
    private final PasswordEncoder passwordEncoder;
    private final JwtUtil jwtUtil;

    public AuthService(UserMapper userMapper, PasswordEncoder passwordEncoder, JwtUtil jwtUtil) {
        this.userMapper = userMapper;
        this.passwordEncoder = passwordEncoder;
        this.jwtUtil = jwtUtil;
    }

    public AuthDTO.UserInfo register(AuthDTO.RegisterReq req) {
        String studentNo = validateStudentNo(req.getStudentNo());
        String name = validateName(req.getName());
        Long exists = userMapper.selectCount(new LambdaQueryWrapper<User>()
                .eq(User::getStudentNo, studentNo));
        if (exists > 0) {
            throw new BusinessException("学号已被注册");
        }
        User u = new User();
        u.setName(name);
        u.setStudentNo(studentNo);
        u.setPasswordHash(passwordEncoder.encode(req.getPassword()));
        u.setPhone(req.getPhone());
        u.setCollege(req.getCollege());
        u.setCampus(req.getCampus());
        u.setRole("USER");
        u.setStatus("ACTIVE");
        userMapper.insert(u);
        return toInfo(u);
    }

    public AuthDTO.LoginResp login(AuthDTO.LoginReq req) {
        User u = userMapper.selectOne(new LambdaQueryWrapper<User>()
                .eq(User::getStudentNo, req.getStudentNo()));
        if (u == null) {
            throw new BusinessException("账号或密码错误");
        }
        if (!passwordEncoder.matches(req.getPassword(), u.getPasswordHash())) {
            throw new BusinessException("账号或密码错误");
        }
        if ("DISABLED".equalsIgnoreCase(u.getStatus())) {
            throw new BusinessException("账号已被禁用");
        }
        AuthDTO.LoginResp resp = new AuthDTO.LoginResp();
        resp.setToken(jwtUtil.generate(u.getId(), u.getRole()));
        resp.setUser(toInfo(u));
        return resp;
    }

    public AuthDTO.UserInfo getMe(Long id) {
        User u = userMapper.selectById(id);
        if (u == null) {
            throw new BusinessException(401, "用户不存在");
        }
        return toInfo(u);
    }

    public AuthDTO.UserInfo updateMe(Long id, AuthDTO.UserInfo req) {
        User u = userMapper.selectById(id);
        if (u == null) {
            throw new BusinessException(401, "用户不存在");
        }
        if (req.getName() != null) u.setName(validateName(req.getName()));
        if (req.getPhone() != null) u.setPhone(req.getPhone());
        if (req.getCollege() != null) u.setCollege(req.getCollege());
        if (req.getCampus() != null) u.setCampus(req.getCampus());
        if (req.getAvatar() != null) u.setAvatar(req.getAvatar());
        userMapper.updateById(u);
        return toInfo(u);
    }

    private String validateStudentNo(String value) {
        String studentNo = value == null ? "" : value.trim();
        if (!studentNo.matches(STUDENT_NO_PATTERN)) {
            throw new BusinessException("学号必须是 12 位纯数字");
        }
        for (String fake : OBVIOUS_FAKE_STUDENT_NOS) {
            if (fake.equals(studentNo)) {
                throw new BusinessException("请填写真实学号");
            }
        }
        return studentNo;
    }

    private String validateName(String value) {
        String name = value == null ? "" : value.trim().replaceAll("\\s+", " ");
        if (name.length() < 2 || name.length() > 20) {
            throw new BusinessException("姓名长度需为 2-20 个字符");
        }
        if (!name.matches(NAME_PATTERN)) {
            throw new BusinessException("姓名只能包含中文、英文字母、空格、点号、中点或连字符，不能包含数字");
        }
        return name;
    }

    private AuthDTO.UserInfo toInfo(User u) {
        AuthDTO.UserInfo i = new AuthDTO.UserInfo();
        i.setId(u.getId());
        i.setName(u.getName());
        i.setStudentNo(u.getStudentNo());
        i.setPhone(u.getPhone());
        i.setCollege(u.getCollege());
        i.setCampus(u.getCampus());
        i.setAvatar(u.getAvatar());
        i.setRole(u.getRole());
        i.setStatus(u.getStatus());
        return i;
    }
}
