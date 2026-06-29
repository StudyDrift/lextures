package com.lextures.android.core.auth

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AuthCallbackParserTest {
    @Test
    fun parsesAuthCallbackQueryParams() {
        val payload = AuthCallbackParser.parse(
            "lextures://auth/callback?access_token=abc123&refresh_token=def456",
        )
        assertEquals("abc123", payload?.accessToken)
        assertEquals("def456", payload?.refreshToken)
    }

    @Test
    fun parsesAuthCallbackMfaParams() {
        val payload = AuthCallbackParser.parse(
            "lextures://auth/callback?mfa_pending_token=pending&requires_mfa=1&mfa_setup_required=1",
        )
        assertEquals("pending", payload?.mfaPendingToken)
        assertTrue(payload?.requiresMfa == true)
        assertTrue(payload?.mfaSetupRequired == true)
    }

    @Test
    fun parsesMagicLinkHttpsUrl() {
        val payload = AuthCallbackParser.parse(
            "https://lextures.com/login/magic-link?token=ml-token-123",
        )
        assertEquals("ml-token-123", payload?.magicLinkToken)
    }

    @Test
    fun ignoresNavigationLinks() {
        assertNull(AuthCallbackParser.parse("/courses/cs101/grades"))
    }
}
