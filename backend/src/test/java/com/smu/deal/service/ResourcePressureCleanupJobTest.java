package com.smu.deal.service;

import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.assertj.core.api.Assertions.assertThat;

class ResourcePressureCleanupJobTest {

    @Test
    void shouldCleanupWhenDiskFreeRatioIsBelowThreshold() {
        ResourcePressureCleanupJob job = new ResourcePressureCleanupJob(
                () -> new ResourcePressureSnapshot(0.14, 0.20),
                null,
                true,
                0.15,
                0.90,
                100,
                7);

        assertThat(job.shouldCleanup(new ResourcePressureSnapshot(0.14, 0.20))).isTrue();
    }

    @Test
    void shouldCleanupWhenHeapUsedRatioIsAboveThreshold() {
        ResourcePressureCleanupJob job = new ResourcePressureCleanupJob(
                () -> new ResourcePressureSnapshot(0.50, 0.91),
                null,
                true,
                0.15,
                0.90,
                100,
                7);

        assertThat(job.shouldCleanup(new ResourcePressureSnapshot(0.50, 0.91))).isTrue();
    }

    @Test
    void shouldNotCleanupWhenResourcesAreHealthy() {
        ResourcePressureCleanupJob job = new ResourcePressureCleanupJob(
                () -> new ResourcePressureSnapshot(0.50, 0.40),
                null,
                true,
                0.15,
                0.90,
                100,
                7);

        assertThat(job.shouldCleanup(new ResourcePressureSnapshot(0.50, 0.40))).isFalse();
    }
}
