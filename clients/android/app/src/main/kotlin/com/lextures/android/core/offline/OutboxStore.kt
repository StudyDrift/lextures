package com.lextures.android.core.offline

import android.content.Context
import java.io.File
import java.util.UUID
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
private data class PersistedOutbox(
    val items: List<OutboxItem> = emptyList(),
    val nextSequence: Int = 0,
)

class OutboxStore(context: Context, ownerKey: String) {
    private val rootDir = File(context.applicationContext.filesDir, "offline/$ownerKey").apply { mkdirs() }
    private val outboxFile = File(rootDir, "outbox.json")
    private val appliedKeysFile = File(rootDir, "applied-idempotency-keys.json")
    private val json = Json { ignoreUnknownKeys = true }
    private var state = loadOutbox()
    private var appliedKeys: MutableSet<String> = loadAppliedKeys()

    fun enqueue(method: String, path: String, bodyJson: String?, label: String): OutboxItem {
        val item = OutboxItem(
            id = UUID.randomUUID().toString(),
            sequence = state.nextSequence,
            method = method,
            path = path,
            bodyJson = bodyJson,
            label = label,
        )
        state = state.copy(
            items = state.items + item,
            nextSequence = state.nextSequence + 1,
        )
        persistOutbox()
        return item
    }

    fun pendingItems(): List<OutboxItem> =
        state.items.filter {
            it.outboxStatus() == OutboxStatus.Queued || it.outboxStatus() == OutboxStatus.Failed
        }.sortedBy { it.sequence }

    fun pendingCount(): Int =
        state.items.count {
            val status = it.outboxStatus()
            status == OutboxStatus.Queued || status == OutboxStatus.Failed || status == OutboxStatus.Syncing
        }

    fun allItems(): List<OutboxItem> = state.items.sortedBy { it.sequence }

    fun update(item: OutboxItem) {
        state = state.copy(items = state.items.map { if (it.id == item.id) item else it })
        persistOutbox()
    }

    fun markApplied(idempotencyKey: String) {
        appliedKeys.add(idempotencyKey)
        appliedKeysFile.writeText(json.encodeToString(appliedKeys))
    }

    fun wasApplied(idempotencyKey: String): Boolean = appliedKeys.contains(idempotencyKey)

    fun retry(id: String) {
        state = state.copy(
            items = state.items.map { item ->
                if (item.id == id) item.copy(status = OutboxStatus.Queued.name, lastError = null) else item
            },
        )
        persistOutbox()
    }

    fun clearAll() {
        state = PersistedOutbox()
        appliedKeys.clear()
        persistOutbox()
        appliedKeysFile.delete()
    }

    private fun loadOutbox(): PersistedOutbox {
        if (!outboxFile.exists()) return PersistedOutbox()
        return runCatching { json.decodeFromString<PersistedOutbox>(outboxFile.readText()) }
            .getOrDefault(PersistedOutbox())
    }

    private fun persistOutbox() {
        outboxFile.writeText(json.encodeToString(state))
    }

    private fun loadAppliedKeys(): MutableSet<String> {
        if (!appliedKeysFile.exists()) return mutableSetOf()
        return runCatching { json.decodeFromString<Set<String>>(appliedKeysFile.readText()).toMutableSet() }
            .getOrDefault(mutableSetOf())
    }
}
