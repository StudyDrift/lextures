package com.lextures.android.features.auth

import com.lextures.android.core.auth.OidcStatusResponse
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.security.MessageDigest

class GoogleSignInLogicTest {
    @Test
    fun sha256Hex_isStable() {
        val raw = "test-nonce-value"
        val h1 = GoogleSignIn.sha256Hex(raw)
        val h2 = GoogleSignIn.sha256Hex(raw)
        assertEquals(h1, h2)
        assertEquals(64, h1.length)
        assertTrue(h1.all { it in "0123456789abcdef" })
        assertNotEquals(h1, GoogleSignIn.sha256Hex("other"))
    }

    @Test
    fun sha256Hex_matchesMessageDigest() {
        val raw = "abc"
        val digest = MessageDigest.getInstance("SHA-256").digest(raw.toByteArray(Charsets.UTF_8))
        val want = digest.joinToString("") { "%02x".format(it) }
        assertEquals(want, GoogleSignIn.sha256Hex(raw))
    }

    @Test
    fun randomNonce_lengthAndEntropy() {
        val a = GoogleSignIn.randomNonce(32)
        val b = GoogleSignIn.randomNonce(32)
        assertEquals(32, a.length)
        assertEquals(32, b.length)
        assertNotEquals(a, b)
    }

    @Test
    fun oidcStatus_showsGoogleNative() {
        assertTrue(OidcStatusResponse(googleNative = true).showsGoogleNative)
        assertFalse(OidcStatusResponse(googleNative = false).showsGoogleNative)
        assertFalse(OidcStatusResponse().showsGoogleNative)
    }
}
