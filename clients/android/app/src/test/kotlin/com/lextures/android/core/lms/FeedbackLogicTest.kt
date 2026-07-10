package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class FeedbackLogicTest {
    @Test
    fun feedbackEnabledDefaultsOn() {
        assertTrue(FeedbackLogic.feedbackEnabled(MobilePlatformFeatures()))
        assertFalse(FeedbackLogic.feedbackEnabled(MobilePlatformFeatures(ffFeedback = false)))
    }

    @Test
    fun messageValid() {
        assertFalse(FeedbackLogic.messageValid(""))
        assertFalse(FeedbackLogic.messageValid("   "))
        assertTrue(FeedbackLogic.messageValid("note"))
    }

    @Test
    fun buildSubmitRequest() {
        val request = FeedbackLogic.buildSubmitRequest(
            message = "  Hello  ",
            category = "idea",
            route = "profile",
            locale = "en",
            viewport = "412x915",
        )
        assertEquals("Hello", request.message)
        assertEquals("android", request.source)
        assertTrue(request.appVersion.isNotBlank())
        assertEquals("profile", request.context.route)
        assertEquals("en", request.context.locale)
        assertEquals("412x915", request.context.viewport)
        assertEquals("idea", request.category)
    }

    @Test
    fun buildSubmitRequestOmitsEmptyCategory() {
        val request = FeedbackLogic.buildSubmitRequest(
            message = "Hi",
            category = "",
            route = "profile",
            locale = null,
            viewport = null,
        )
        assertEquals(null, request.category)
    }

    @Test
    fun mapSubmitError() {
        assertEquals(
            FeedbackLogic.SubmitOutcome.RateLimited,
            FeedbackLogic.mapSubmitError(ApiError.HttpStatus(429, null), isOnline = true),
        )
        assertEquals(
            FeedbackLogic.SubmitOutcome.Offline,
            FeedbackLogic.mapSubmitError(ApiError.Transport(RuntimeException("offline")), isOnline = true),
        )
        assertEquals(
            FeedbackLogic.SubmitOutcome.Error,
            FeedbackLogic.mapSubmitError(ApiError.HttpStatus(500, null), isOnline = true),
        )
        assertEquals(
            FeedbackLogic.SubmitOutcome.Offline,
            FeedbackLogic.mapSubmitError(ApiError.HttpStatus(500, null), isOnline = false),
        )
    }
}
