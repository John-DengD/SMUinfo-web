package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.AnnouncementDTO;
import com.smu.deal.entity.Announcement;
import com.smu.deal.mapper.AnnouncementMapper;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.util.HtmlUtils;

import java.util.List;

@Service
public class AnnouncementService {

    private static final String ACTIVE = "ACTIVE";
    private static final String INACTIVE = "INACTIVE";

    private final AnnouncementMapper announcementMapper;

    public AnnouncementService(AnnouncementMapper announcementMapper) {
        this.announcementMapper = announcementMapper;
    }

    public AnnouncementDTO.Item active() {
        Announcement row = announcementMapper.selectOne(new LambdaQueryWrapper<Announcement>()
                .eq(Announcement::getStatus, ACTIVE)
                .orderByDesc(Announcement::getCreatedAt)
                .last("LIMIT 1"));
        return row == null ? null : toItem(row);
    }

    public List<AnnouncementDTO.Item> list() {
        return announcementMapper.selectList(new LambdaQueryWrapper<Announcement>()
                        .orderByDesc(Announcement::getCreatedAt))
                .stream()
                .map(this::toItem)
                .toList();
    }

    @Transactional
    public AnnouncementDTO.Item create(Long adminId, AnnouncementDTO.SaveReq req) {
        String status = normalizeStatus(req.getStatus());
        Announcement row = new Announcement();
        row.setTitle(clean(req.getTitle()));
        row.setContent(clean(req.getContent()));
        row.setStatus(status);
        row.setCreatedBy(adminId);
        announcementMapper.insert(row);
        if (ACTIVE.equals(status)) {
            disableOtherActive(row.getId());
        }
        return toItem(announcementMapper.selectById(row.getId()));
    }

    @Transactional
    public AnnouncementDTO.Item update(Long id, AnnouncementDTO.SaveReq req) {
        Announcement existing = announcementMapper.selectById(id);
        if (existing == null) {
            throw new BusinessException("公告不存在");
        }
        String status = normalizeStatus(req.getStatus());
        existing.setTitle(clean(req.getTitle()));
        existing.setContent(clean(req.getContent()));
        existing.setStatus(status);
        announcementMapper.updateById(existing);
        if (ACTIVE.equals(status)) {
            disableOtherActive(id);
        }
        return toItem(announcementMapper.selectById(id));
    }

    public void delete(Long id) {
        announcementMapper.deleteById(id);
    }

    private void disableOtherActive(Long activeId) {
        Announcement patch = new Announcement();
        patch.setStatus(INACTIVE);
        announcementMapper.update(patch, new LambdaQueryWrapper<Announcement>()
                .eq(Announcement::getStatus, ACTIVE)
                .ne(Announcement::getId, activeId));
    }

    private String normalizeStatus(String status) {
        if (status == null || status.isBlank()) return ACTIVE;
        String normalized = status.trim().toUpperCase();
        if (!ACTIVE.equals(normalized) && !INACTIVE.equals(normalized)) {
            throw new BusinessException("公告状态不正确");
        }
        return normalized;
    }

    private String clean(String value) {
        return HtmlUtils.htmlEscape(value == null ? "" : value.trim());
    }

    private AnnouncementDTO.Item toItem(Announcement row) {
        AnnouncementDTO.Item item = new AnnouncementDTO.Item();
        item.setId(row.getId());
        item.setTitle(row.getTitle());
        item.setContent(row.getContent());
        item.setStatus(row.getStatus());
        item.setCreatedBy(row.getCreatedBy());
        item.setCreatedAt(row.getCreatedAt());
        item.setUpdatedAt(row.getUpdatedAt());
        return item;
    }
}
