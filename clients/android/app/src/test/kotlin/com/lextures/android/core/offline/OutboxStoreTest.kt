package com.lextures.android.core.offline

import android.content.Context
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner

@RunWith(RobolectricTestRunner::class)
class OutboxStoreTest {
    private lateinit var context: Context
    private lateinit var store: OutboxStore

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        store = OutboxStore(context, ownerKey = "test-user-${System.nanoTime()}")
    }

    @Test
    fun enqueue_returnsPendingItemInOrder() {
        val first = store.enqueue("POST", "/api/v1/test", """{"a":1}""", "Test write")
        val second = store.enqueue("PATCH", "/api/v1/other", null, "Other write")

        val pending = store.pendingItems()
        assertEquals(2, pending.size)
        assertTrue(pending[0].sequence < pending[1].sequence)
        assertEquals(first.id, pending[0].id)
        assertEquals(second.id, pending[1].id)
    }

    @Test
    fun markApplied_preventsDuplicateApplication() {
        val item = store.enqueue("POST", "/api/v1/test", null, "Test")
        store.markApplied(item.idempotencyKey)
        assertTrue(store.wasApplied(item.idempotencyKey))
    }

    @Test
    fun retry_resetsFailedItemToQueued() {
        val item = store.enqueue("POST", "/api/v1/test", null, "Test")
        store.update(item.copy(status = OutboxStatus.Failed.name, lastError = "boom"))
        store.retry(item.id)
        assertEquals(OutboxStatus.Queued, store.pendingItems().single().outboxStatus())
    }
}
