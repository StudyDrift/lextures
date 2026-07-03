package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CredentialsLogicTest {
    @Test
    fun credentialsEnabled() {
        assertFalse(CredentialsLogic.credentialsEnabled(MobilePlatformFeatures()))
        assertTrue(CredentialsLogic.credentialsEnabled(MobilePlatformFeatures(ffCompletionCredentials = true)))
    }

    @Test
    fun sourceTypeLabel() {
        assertEquals("Course", CredentialsLogic.sourceTypeLabel("course"))
        assertEquals("Learning path", CredentialsLogic.sourceTypeLabel("path"))
    }

    @Test
    fun cacheKeys() {
        assertEquals("credentials:list", CredentialsLogic.cacheKey())
        assertEquals("credentials:abc", CredentialsLogic.credentialDetailCacheKey("abc"))
    }
}