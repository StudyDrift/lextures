package com.lextures.android.core.offline

import android.content.Context
import java.io.File
import java.time.Instant
import kotlinx.serialization.Serializable
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

@Serializable
private data class DownloadRecord(
    val savedAtEpochMs: Long,
    val fileName: String,
    val mimeType: String? = null,
    val sizeBytes: Int,
    val lastAccessEpochMs: Long,
)

class DownloadStore(
    context: Context,
    ownerKey: String,
    private val maxBytes: Long = OfflineStorageBudget.DEFAULT_MAX_BYTES,
) {
    private val rootDir = File(context.applicationContext.filesDir, "offline/$ownerKey/downloads").apply { mkdirs() }
    private val indexFile = File(rootDir, "index.json")
    private val json = Json { ignoreUnknownKeys = true }
    private var index: MutableMap<String, DownloadRecord> = loadIndex()

    fun isDownloaded(key: String): Boolean = index.containsKey(key) && fileFor(key).exists()

    fun save(key: String, data: ByteArray, fileName: String, mimeType: String?) {
        fileFor(key).writeBytes(data)
        val now = Instant.now()
        index[key] = DownloadRecord(
            savedAtEpochMs = now.toEpochMilli(),
            fileName = fileName,
            mimeType = mimeType,
            sizeBytes = data.size,
            lastAccessEpochMs = now.toEpochMilli(),
        )
        evictIfNeeded()
        persistIndex()
    }

    fun loadData(key: String): ByteArray? {
        val record = index[key] ?: return null
        index[key] = record.copy(lastAccessEpochMs = Instant.now().toEpochMilli())
        persistIndex()
        val file = fileFor(key)
        return if (file.exists()) file.readBytes() else null
    }

    fun totalSizeBytes(): Long = index.values.sumOf { it.sizeBytes.toLong() }

    fun clearAll() {
        index.clear()
        rootDir.deleteRecursively()
        rootDir.mkdirs()
        persistIndex()
    }

    private fun fileFor(key: String): File = File(rootDir, key.replace(Regex("[^a-zA-Z0-9._-]"), "_"))

    private fun loadIndex(): MutableMap<String, DownloadRecord> {
        if (!indexFile.exists()) return mutableMapOf()
        return runCatching {
            json.decodeFromString<Map<String, DownloadRecord>>(indexFile.readText()).toMutableMap()
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
            fileFor(key).delete()
        }
    }
}
