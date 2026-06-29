package com.lextures.android.core.offline

import android.content.Context
import java.io.File
import java.time.Instant
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
private data class CacheEntry(
    val fetchedAtEpochMs: Long,
    val payload: String,
    val sizeBytes: Int,
    val lastAccessEpochMs: Long,
)

class CacheStore(
    context: Context,
    ownerKey: String,
    private val maxBytes: Long = OfflineStorageBudget.DEFAULT_MAX_BYTES,
) {
    private val rootDir = File(context.applicationContext.filesDir, "offline/$ownerKey/cache").apply { mkdirs() }
    private val indexFile = File(rootDir, "index.json")
    private val json = Json { ignoreUnknownKeys = true }
    private var index: MutableMap<String, CacheEntry> = loadIndex()

    inline fun <reified T> get(key: String, serializer: kotlinx.serialization.KSerializer<T>): Cached<T>? {
        val entry = index[key] ?: return null
        val now = Instant.now()
        index[key] = entry.copy(lastAccessEpochMs = now.toEpochMilli())
        persistIndex()
        val value = runCatching { json.decodeFromString(serializer, entry.payload) }.getOrNull() ?: return null
        return Cached(value = value, fetchedAt = Instant.ofEpochMilli(entry.fetchedAtEpochMs))
    }

    inline fun <reified T> put(key: String, value: T, serializer: kotlinx.serialization.KSerializer<T>) {
        val payload = json.encodeToString(serializer, value)
        val now = Instant.now()
        index[key] = CacheEntry(
            fetchedAtEpochMs = now.toEpochMilli(),
            payload = payload,
            sizeBytes = payload.toByteArray().size,
            lastAccessEpochMs = now.toEpochMilli(),
        )
        evictIfNeeded()
        persistIndex()
    }

    fun totalSizeBytes(): Long = index.values.sumOf { it.sizeBytes.toLong() }

    fun clearAll() {
        index.clear()
        rootDir.deleteRecursively()
        rootDir.mkdirs()
        persistIndex()
    }

    private fun loadIndex(): MutableMap<String, CacheEntry> {
        if (!indexFile.exists()) return mutableMapOf()
        return runCatching {
            json.decodeFromString<Map<String, CacheEntry>>(indexFile.readText()).toMutableMap()
        }.getOrDefault(mutableMapOf())
    }

    private fun persistIndex() {
        indexFile.writeText(json.encodeToString(index))
    }

    private fun evictIfNeeded() {
        var total = index.values.sumOf { it.sizeBytes.toLong() }
        if (total <= maxBytes) return
        val sorted = index.entries.sortedBy { it.value.lastAccessEpochMs }
        for ((key, entry) in sorted) {
            if (total <= maxBytes) break
            total -= entry.sizeBytes
            index.remove(key)
        }
    }
}
