package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CoursePlagiarismLogicTest {
    @Test
    fun draftUsesDefaultsWhenSettingsMissing() {
        val draft = CoursePlagiarismLogic.draft(null)
        assertTrue(draft.checksEnabled)
        assertEquals("", draft.provider)
        assertEquals("40", draft.thresholdPct)
    }

    @Test
    fun draftMapsSettings() {
        val draft = CoursePlagiarismLogic.draft(
            CoursePlagiarismSettings(
                plagiarismChecksEnabled = false,
                plagiarismProvider = "turnitin",
                plagiarismAlertThresholdPct = 55.0,
            ),
        )
        assertFalse(draft.checksEnabled)
        assertEquals("turnitin", draft.provider)
        assertEquals("55", draft.thresholdPct)
    }

    @Test
    fun isDirtyDetectsProviderChange() {
        val baseline = CoursePlagiarismLogic.draft(null)
        val current = baseline.copy(provider = "copyleaks")
        assertFalse(CoursePlagiarismLogic.isDirty(baseline, baseline))
        assertTrue(CoursePlagiarismLogic.isDirty(current, baseline))
    }

    @Test
    fun validateDraftRejectsInvalidThreshold() {
        assertEquals(
            CoursePlagiarismLogic.ValidationError.ThresholdInvalid,
            CoursePlagiarismLogic.validateDraft(CoursePlagiarismLogic.FormDraft(thresholdPct = "150")),
        )
        assertNull(CoursePlagiarismLogic.validateDraft(CoursePlagiarismLogic.FormDraft(thresholdPct = "40")))
    }

    @Test
    fun buildPatchBodyUsesNullProviderForDefault() {
        val body = CoursePlagiarismLogic.buildPatchBody(
            CoursePlagiarismLogic.FormDraft(provider = "", thresholdPct = "25"),
        )
        assertNull(body.plagiarismProvider)
        assertEquals(25.0, body.plagiarismAlertThresholdPct, 0.001)
    }

    @Test
    fun normalizedProviderRejectsUnknown() {
        assertEquals("", CoursePlagiarismLogic.normalizedProvider("proprietary"))
        assertEquals("turnitin", CoursePlagiarismLogic.normalizedProvider("Turnitin"))
    }
}
