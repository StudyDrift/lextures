package com.lextures.android.core.config

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SchoolCodeLogicTest {
    @Test
    fun normalizeTrimsAndLowercases() {
        assertEquals("example-school", SchoolCodeLogic.normalize("  Example-School "))
    }

    @Test
    fun validSchoolCodes() {
        assertTrue(SchoolCodeLogic.isValid("example"))
        assertTrue(SchoolCodeLogic.isValid("my-school"))
        assertTrue(SchoolCodeLogic.isValid("local"))
    }

    @Test
    fun invalidSchoolCodes() {
        assertFalse(SchoolCodeLogic.isValid(""))
        assertFalse(SchoolCodeLogic.isValid("a"))
        assertFalse(SchoolCodeLogic.isValid("bad_code"))
        assertFalse(SchoolCodeLogic.isValid("-leading"))
        assertFalse(SchoolCodeLogic.isValid("self"))
        assertFalse(SchoolCodeLogic.isValid("www"))
    }

    @Test
    fun mixedCaseIsNormalizedBeforeValidation() {
        assertTrue(SchoolCodeLogic.isValid("Example"))
        assertEquals("example", SchoolCodeLogic.normalize("Example"))
    }

    @Test
    fun apiBaseUrls() {
        assertEquals("https://self.lextures.com", SchoolCodeLogic.SELF_LEARNER_API_BASE)
        assertEquals("https://example.lextures.com", SchoolCodeLogic.apiBaseUrl("example"))
        assertEquals("http://127.0.0.1:8080", SchoolCodeLogic.apiBaseUrl("Local"))
    }

    @Test
    fun previewHost() {
        assertEquals("your-school.lextures.com", SchoolCodeLogic.previewHost(""))
        assertEquals("127.0.0.1:8080", SchoolCodeLogic.previewHost("local"))
        assertEquals("demo-uni.lextures.com", SchoolCodeLogic.previewHost("demo-uni"))
    }

    @Test
    fun errorKeys() {
        assertEquals("auth_getStarted_schoolCodeErrorEmpty", SchoolCodeLogic.errorKey(""))
        assertEquals("auth_getStarted_schoolCodeErrorReserved", SchoolCodeLogic.errorKey("self"))
        assertNull(SchoolCodeLogic.errorKey("local"))
        assertNull(SchoolCodeLogic.errorKey("ok-school"))
    }
}
