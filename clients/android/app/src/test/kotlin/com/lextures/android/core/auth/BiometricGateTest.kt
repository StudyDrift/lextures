package com.lextures.android.core.auth

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class BiometricGateTest {
    @Test
    fun shouldLockAfterTimeout() {
        assertTrue(BiometricGate.shouldLock(60_000L))
        assertTrue(BiometricGate.shouldLock(120_000L))
    }

    @Test
    fun shouldNotLockBeforeTimeout() {
        assertFalse(BiometricGate.shouldLock(59_000L))
        assertFalse(BiometricGate.shouldLock(0L))
    }
}
