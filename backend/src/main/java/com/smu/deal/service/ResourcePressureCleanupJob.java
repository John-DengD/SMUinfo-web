package com.smu.deal.service;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.concurrent.atomic.AtomicBoolean;

@Component
public class ResourcePressureCleanupJob {

    private static final Logger log = LoggerFactory.getLogger(ResourcePressureCleanupJob.class);

    private final ResourceUsageProbe resourceUsageProbe;
    private final ProductCleanupService productCleanupService;
    private final boolean enabled;
    private final double minDiskFreeRatio;
    private final double maxHeapUsedRatio;
    private final int batchSize;
    private final Duration minAge;
    private final AtomicBoolean running = new AtomicBoolean(false);

    public ResourcePressureCleanupJob(ResourceUsageProbe resourceUsageProbe,
                                      ProductCleanupService productCleanupService,
                                      @Value("${app.cleanup.enabled:true}") boolean enabled,
                                      @Value("${app.cleanup.min-disk-free-ratio:0.15}") double minDiskFreeRatio,
                                      @Value("${app.cleanup.max-heap-used-ratio:0.90}") double maxHeapUsedRatio,
                                      @Value("${app.cleanup.batch-size:100}") int batchSize,
                                      @Value("${app.cleanup.min-age-days:7}") int minAgeDays) {
        this.resourceUsageProbe = resourceUsageProbe;
        this.productCleanupService = productCleanupService;
        this.enabled = enabled;
        this.minDiskFreeRatio = minDiskFreeRatio;
        this.maxHeapUsedRatio = maxHeapUsedRatio;
        this.batchSize = batchSize;
        this.minAge = Duration.ofDays(Math.max(0, minAgeDays));
    }

    @Scheduled(initialDelayString = "${app.cleanup.initial-delay-ms:60000}",
            fixedDelayString = "${app.cleanup.check-interval-ms:300000}")
    public void run() {
        if (!enabled) return;
        if (!running.compareAndSet(false, true)) return;
        try {
            ResourcePressureSnapshot snapshot = resourceUsageProbe.snapshot();
            if (!shouldCleanup(snapshot)) {
                return;
            }
            log.warn("resource pressure detected: diskFreeRatio={}, heapUsedRatio={}",
                    snapshot.diskFreeRatio(), snapshot.heapUsedRatio());
            productCleanupService.cleanupClosedProducts(batchSize, minAge);
        } catch (Exception e) {
            log.error("resource fallback cleanup failed", e);
        } finally {
            running.set(false);
        }
    }

    boolean shouldCleanup(ResourcePressureSnapshot snapshot) {
        return snapshot.diskFreeRatio() < minDiskFreeRatio || snapshot.heapUsedRatio() > maxHeapUsedRatio;
    }
}
