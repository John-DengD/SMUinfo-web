package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.ReportDTO;
import com.smu.deal.entity.Product;
import com.smu.deal.entity.Report;
import com.smu.deal.entity.User;
import com.smu.deal.mapper.ProductMapper;
import com.smu.deal.mapper.ReportMapper;
import com.smu.deal.mapper.UserMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.util.HtmlUtils;

import java.util.*;
import java.util.stream.Collectors;

@Service
public class ReportService {

    private final ReportMapper reportMapper;
    private final ProductMapper productMapper;
    private final UserMapper userMapper;

    public ReportService(ReportMapper reportMapper, ProductMapper productMapper, UserMapper userMapper) {
        this.reportMapper = reportMapper;
        this.productMapper = productMapper;
        this.userMapper = userMapper;
    }

    public void create(Long reporterId, ReportDTO.CreateReq req) {
        Product p = productMapper.selectById(req.getProductId());
        if (p == null) throw new BusinessException("商品不存在");
        Report r = new Report();
        r.setReporterId(reporterId);
        r.setProductId(req.getProductId());
        r.setReason(HtmlUtils.htmlEscape(req.getReason()));
        r.setStatus("PENDING");
        reportMapper.insert(r);
    }

    public List<ReportDTO.Item> list(String status) {
        LambdaQueryWrapper<Report> w = new LambdaQueryWrapper<>();
        if (status != null && !status.isBlank()) {
            w.eq(Report::getStatus, status);
        }
        w.orderByDesc(Report::getCreatedAt);
        List<Report> reports = reportMapper.selectList(w);
        if (reports.isEmpty()) return new ArrayList<>();
        Set<Long> userIds = reports.stream().map(Report::getReporterId).collect(Collectors.toSet());
        Set<Long> productIds = reports.stream().map(Report::getProductId).collect(Collectors.toSet());
        Map<Long, User> umap = userMapper.selectBatchIds(userIds).stream()
                .collect(Collectors.toMap(User::getId, u -> u));
        Map<Long, Product> pmap = productMapper.selectBatchIds(productIds).stream()
                .collect(Collectors.toMap(Product::getId, p -> p));
        return reports.stream().map(r -> {
            ReportDTO.Item it = new ReportDTO.Item();
            it.setId(r.getId());
            it.setReporterId(r.getReporterId());
            it.setProductId(r.getProductId());
            it.setReason(r.getReason());
            it.setStatus(r.getStatus());
            it.setAdminRemark(r.getAdminRemark());
            it.setCreatedAt(r.getCreatedAt());
            User u = umap.get(r.getReporterId());
            if (u != null) it.setReporterName(u.getName());
            Product p = pmap.get(r.getProductId());
            if (p != null) it.setProductTitle(p.getTitle());
            return it;
        }).toList();
    }

    public void handle(Long id, ReportDTO.HandleReq req) {
        Report r = reportMapper.selectById(id);
        if (r == null) throw new BusinessException("举报不存在");
        if (req.getStatus() != null) r.setStatus(req.getStatus());
        if (req.getAdminRemark() != null) r.setAdminRemark(req.getAdminRemark());
        reportMapper.updateById(r);
    }
}
