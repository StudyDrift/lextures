package com.lextures.android.core.offline

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import com.lextures.android.core.lms.CourseSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class CacheStoreTest {
    private lateinit var context: Context
    private lateinit var store: CacheStore

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        store = CacheStore(context, ownerKey = "test-user-${System.nanoTime()}")
    }

    @Test
    fun putAndGet_roundTripsTypedValue() {
        val courses = listOf(
            CourseSummary(id = "1", courseCode = "CS101", title = "Intro"),
        )
        val listSerializer = kotlinx.serialization.builtins.ListSerializer(CourseSummary.serializer())
        store.put(OfflineCacheKey.courses(), courses, listSerializer)
        val cached = store.get(OfflineCacheKey.courses(), listSerializer)
        assertNotNull(cached)
        assertEquals(courses, cached?.value)
    }
}
