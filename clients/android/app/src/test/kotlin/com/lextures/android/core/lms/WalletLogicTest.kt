package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class WalletLogicTest {
    @Test
    fun walletEnabled() {
        assertFalse(WalletLogic.walletEnabled(MobilePlatformFeatures()))
        assertTrue(WalletLogic.walletEnabled(MobilePlatformFeatures(ffCompletionCredentials = true)))
        assertTrue(WalletLogic.walletEnabled(MobilePlatformFeatures(ffCoCurricularTranscript = true)))
        assertTrue(WalletLogic.walletEnabled(MobilePlatformFeatures(ffCeuTracking = true)))
        assertTrue(WalletLogic.walletEnabled(MobilePlatformFeatures(ffTranscripts = true)))
    }

    @Test
    fun cacheKeys() {
        assertEquals("wallet:ccr", WalletLogic.cacheKeyCcr())
        assertEquals("wallet:ce-transcript", WalletLogic.cacheKeyCeTranscript())
        assertEquals("wallet:transcript-requests", WalletLogic.cacheKeyTranscriptRequests())
    }

    @Test
    fun officialTranscriptWebUrl() {
        assertTrue(WalletLogic.officialTranscriptWebUrl().contains("/transcripts"))
    }
}
