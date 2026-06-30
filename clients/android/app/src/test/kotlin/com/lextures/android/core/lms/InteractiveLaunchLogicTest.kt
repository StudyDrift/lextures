package com.lextures.android.core.lms

import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class InteractiveLaunchLogicTest {
    @Test
    fun kindForItemKinds() {
        assertTrue(InteractiveLaunchLogic.kindFor("h5p") == InteractiveLaunchKind.H5p)
        assertTrue(InteractiveLaunchLogic.kindFor("scorm") == InteractiveLaunchKind.Scorm)
        assertTrue(InteractiveLaunchLogic.kindFor("lti_link") == InteractiveLaunchKind.LtiLink)
        assertTrue(InteractiveLaunchLogic.kindFor("vibe_activity") == InteractiveLaunchKind.VibeActivity)
        assertNull(InteractiveLaunchLogic.kindFor("quiz"))
    }

    @Test
    fun h5pRenderPathContainsPackageId() {
        val path = InteractiveLaunchLogic.h5pRenderPath("CS101", "pkg-1")
        assertTrue(path.contains("/courses/CS101/h5p/pkg-1/render"))
    }

    @Test
    fun ltiFramePathIncludesTicket() {
        val path = InteractiveLaunchLogic.ltiFramePath("abc123")
        assertTrue(path.contains("ticket=abc123"))
    }

    @Test
    fun scormHasResumeDetectsSuspendData() {
        assertTrue(
            InteractiveLaunchLogic.scormHasResume(mapOf("cmi.core.suspend_data" to "state")),
        )
        assertTrue(
            InteractiveLaunchLogic.scormHasResume(mapOf("cmi.core.entry" to "resume")),
        )
        assertFalse(InteractiveLaunchLogic.scormHasResume(emptyMap()))
    }

    @Test
    fun vibeActivityHtmlFallback() {
        val html = InteractiveLaunchLogic.vibeActivityHtml(null)
        assertTrue(html.contains("Empty activity"))
    }

    @Test
    fun authInjectionScriptIncludesTokenMarker() {
        val script = InteractiveLaunchLogic.authInjectionScript("tok", "http://localhost:8080")
        assertTrue(script.contains("Bearer "))
        assertTrue(script.contains("tok"))
        assertTrue(script.contains("h5p-xapi"))
    }
}
