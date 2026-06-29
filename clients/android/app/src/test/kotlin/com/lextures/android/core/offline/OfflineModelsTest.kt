package com.lextures.android.core.offline

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.Instant
import java.time.temporal.ChronoUnit

class OfflineModelsTest {
    @Test
    fun cached_isStaleWhenOffline() {
        val cached = Cached(value = "ok", fetchedAt = Instant.now())
        assertTrue(cached.isStale(isOnline = false))
    }

    @Test
    fun cached_isFreshWhenOnlineAndRecent() {
        val cached = Cached(value = "ok", fetchedAt = Instant.now())
        assertFalse(cached.isStale(isOnline = true))
    }

    @Test
    fun cached_isStaleWhenAged() {
        val cached = Cached(
            value = "ok",
            fetchedAt = Instant.now().minus(10, ChronoUnit.MINUTES),
        )
        assertTrue(cached.isStale(isOnline = true))
    }

    @Test
    fun outboxStatus_userLabelsAreStable() {
        assertEquals("Saved locally — will sync", OutboxStatus.Queued.userLabel)
        assertEquals("Conflict — review required", OutboxStatus.Conflict.userLabel)
    }
}
