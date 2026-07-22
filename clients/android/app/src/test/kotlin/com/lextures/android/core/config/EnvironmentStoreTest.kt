package com.lextures.android.core.config

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class EnvironmentStoreTest {
    private lateinit var context: Context

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        context.getSharedPreferences("lextures_environment", Context.MODE_PRIVATE)
            .edit()
            .clear()
            .commit()
        EnvironmentStore.clearInstanceForTests()
    }

    @Test
    fun selectHomeschool_persistsLegacyStorageValue() {
        val store = EnvironmentStore.get(context)
        store.selectHomeschool()
        assertTrue(store.hasSelection)
        assertEquals(EnvironmentStore.Kind.Homeschool, store.kind)
        assertEquals("https://self.lextures.com", store.apiBaseUrl)
        assertNull(store.schoolCode)

        val raw = context.getSharedPreferences("lextures_environment", Context.MODE_PRIVATE)
            .getString("kind", null)
        assertEquals("selfLearner", raw)
    }

    /** Upgrade-in-place: pre-rename installs stored "selfLearner" and must map to Homeschool. */
    @Test
    fun legacySelfLearnerRawValueReadsAsHomeschool() {
        context.getSharedPreferences("lextures_environment", Context.MODE_PRIVATE)
            .edit()
            .putString("kind", "selfLearner")
            .putString("apiBaseURL", "https://self.lextures.com")
            .commit()
        EnvironmentStore.clearInstanceForTests()

        val store = EnvironmentStore.get(context)
        assertTrue(store.hasSelection)
        assertEquals(EnvironmentStore.Kind.Homeschool, store.kind)
        assertEquals("https://self.lextures.com", store.apiBaseUrl)
        assertEquals(SchoolCodeLogic.HOMESCHOOL_API_BASE, store.apiBaseUrl)
    }
}
