package com.smu.deal.service;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.FileStore;
import java.nio.file.Files;
import java.nio.file.Path;

@Component
public class DefaultResourceUsageProbe implements ResourceUsageProbe {

    private final Path uploadRoot;

    public DefaultResourceUsageProbe(@Value("${app.upload.dir}") String uploadDir) {
        this.uploadRoot = Path.of(uploadDir).toAbsolutePath().normalize();
    }

    @Override
    public ResourcePressureSnapshot snapshot() {
        Runtime runtime = Runtime.getRuntime();
        long maxMemory = runtime.maxMemory();
        long usedMemory = runtime.totalMemory() - runtime.freeMemory();
        double heapUsedRatio = maxMemory <= 0 ? 0 : (double) usedMemory / maxMemory;

        try {
            Path probePath = existingPath(uploadRoot);
            FileStore store = Files.getFileStore(probePath);
            long total = store.getTotalSpace();
            long usable = store.getUsableSpace();
            double diskFreeRatio = total <= 0 ? 1 : (double) usable / total;
            return new ResourcePressureSnapshot(diskFreeRatio, heapUsedRatio);
        } catch (IOException e) {
            return new ResourcePressureSnapshot(1, heapUsedRatio);
        }
    }

    private Path existingPath(Path path) {
        Path current = path;
        while (current != null && !Files.exists(current)) {
            current = current.getParent();
        }
        return current == null ? Path.of(".").toAbsolutePath().normalize() : current;
    }
}
