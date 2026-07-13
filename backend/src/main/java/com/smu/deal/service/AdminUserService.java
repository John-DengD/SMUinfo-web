package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.baomidou.mybatisplus.extension.plugins.pagination.Page;
import com.smu.deal.common.BusinessException;
import com.smu.deal.common.PageResult;
import com.smu.deal.dto.AuthDTO;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class AdminUserService {

    private final UserMapper userMapper;

    public AdminUserService(UserMapper userMapper) {
        this.userMapper = userMapper;
    }

    public PageResult<AuthDTO.UserInfo> list(String keyword, Integer page, Integer size) {
        Page<User> p = new Page<>(page == null ? 1 : page, size == null ? 20 : size);
        LambdaQueryWrapper<User> w = new LambdaQueryWrapper<>();
        if (keyword != null && !keyword.isBlank()) {
            w.and(x -> x.like(User::getName, keyword)
                    .or().like(User::getStudentNo, keyword)
                    .or().like(User::getPhone, keyword));
        }
        w.orderByDesc(User::getCreatedAt);
        Page<User> r = userMapper.selectPage(p, w);
        List<AuthDTO.UserInfo> records = r.getRecords().stream().map(u -> {
            AuthDTO.UserInfo i = new AuthDTO.UserInfo();
            i.setId(u.getId());
            i.setName(u.getName());
            i.setStudentNo(u.getStudentNo());
            i.setPhone(u.getPhone());
            i.setCollege(u.getCollege());
            i.setCampus(u.getCampus());
            i.setRole(u.getRole());
            i.setAvatar(u.getAvatar());
            i.setStatus(u.getStatus());
            return i;
        }).toList();
        return PageResult.of(r.getTotal(), records);
    }

    public void changeStatus(Long id, String status) {
        User u = userMapper.selectById(id);
        if (u == null) throw new BusinessException("用户不存在");
        u.setStatus(status);
        userMapper.updateById(u);
    }
}
