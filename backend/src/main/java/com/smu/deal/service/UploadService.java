package com.smu.deal.service;

import com.smu.deal.common.BusinessException;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.File;
import java.io.IOException;
import java.time.LocalDate;
import java.util.Set;
import java.util.UUID;

@Service
public class UploadService {

    private static final Set<String> ALLOWED = Set.of("jpg", "jpeg", "png", "gif", "webp");
    private static final long MAX_SIZE = 5 * 1024 * 1024;

    @Value("${app.upload.dir}")
    private String uploadDir;

    @Value("${app.upload.url-prefix}")
    private String urlPrefix;

    public String upload(MultipartFile file) {
        if (file == null || file.isEmpty()) {
            throw new BusinessException("文件不能为空");
        }
        if (file.getSize() > MAX_SIZE) {
            throw new BusinessException("文件不能超过 5MB");
        }
        String origin = file.getOriginalFilename();
        if (origin == null || !origin.contains(".")) {
            throw new BusinessException("文件名非法");
        }
        String ext = origin.substring(origin.lastIndexOf('.') + 1).toLowerCase();
        if (!ALLOWED.contains(ext)) {
            throw new BusinessException("仅支持 jpg/jpeg/png/gif/webp");
        }
        String sub = LocalDate.now().toString();
        File dir = new File(uploadDir, sub);
        if (!dir.exists() && !dir.mkdirs()) {
            throw new BusinessException("创建目录失败");
        }
        String filename = UUID.randomUUID().toString().replace("-", "") + "." + ext;
        File dest = new File(dir, filename);
        try {
            file.transferTo(dest);
        } catch (IOException e) {
            throw new BusinessException("保存文件失败");
        }
        return urlPrefix + "/" + sub + "/" + filename;
    }
}
