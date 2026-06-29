package com.lextures.android.core.accessibility

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class AccessibilityPreferencesTest {
    private lateinit var context: Context
    private lateinit var preferences: AccessibilityPreferences

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        preferences = AccessibilityPreferences(context)
        preferences.reset()
    }

    @Test
    fun dyslexiaDisplay_persistsAcrossInstances() {
        preferences.dyslexiaDisplayEnabled = true
        val reloaded = AccessibilityPreferences(context)
        assertTrue(reloaded.dyslexiaDisplayEnabled)
    }

    @Test
    fun ttsSpeed_clampsToSupportedRange() {
        preferences.ttsSpeed = 5f
        assertEquals(2f, preferences.ttsSpeed)
        preferences.ttsSpeed = 0.1f
        assertEquals(0.5f, preferences.ttsSpeed)
    }

    @Test
    fun reset_restoresDefaults() {
        preferences.dyslexiaDisplayEnabled = true
        preferences.ttsSpeed = 1.8f
        preferences.reset()
        assertFalse(preferences.dyslexiaDisplayEnabled)
        assertEquals(1f, preferences.ttsSpeed)
    }
}
