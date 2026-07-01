package com.lextures.android.core.lms

import android.content.Context
import java.time.Instant
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.MapSerializer
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json

@Serializable
data class LastVisitedModuleEntry(
    val itemId: String,
    val kind: String,
    val title: String,
    val openedAt: String,
)

/** Per-course last opened module item (parity with web `last-visited-module-item.ts`). */
object ModuleLastVisited {
    private const val STORAGE_KEY = "lextures:last-module-item:v1"
    private val json = Json { ignoreUnknownKeys = true }
    private val storeSerializer = MapSerializer(String.serializer(), LastVisitedModuleEntry.serializer())

    fun record(context: Context, courseCode: String, itemId: String, kind: String, title: String) {
        val code = courseCode.trim()
        val id = itemId.trim()
        if (code.isEmpty() || id.isEmpty()) return

        val store = readStore(context).toMutableMap()
        store[code] = LastVisitedModuleEntry(
            itemId = id,
            kind = kind,
            title = title.trim().ifEmpty { "Untitled" },
            openedAt = Instant.now().toString(),
        )
        writeStore(context, store)
    }

    fun entry(context: Context, courseCode: String): LastVisitedModuleEntry? {
        val code = courseCode.trim()
        if (code.isEmpty()) return null
        return readStore(context)[code]
    }

    private fun readStore(context: Context): Map<String, LastVisitedModuleEntry> {
        val raw = context.getSharedPreferences("module_last_visited", Context.MODE_PRIVATE)
            .getString(STORAGE_KEY, null)
            ?: return emptyMap()
        return runCatching {
            json.decodeFromString(storeSerializer, raw)
        }.getOrDefault(emptyMap())
    }

    private fun writeStore(context: Context, store: Map<String, LastVisitedModuleEntry>) {
        val encoded = json.encodeToString(storeSerializer, store)
        context.getSharedPreferences("module_last_visited", Context.MODE_PRIVATE)
            .edit()
            .putString(STORAGE_KEY, encoded)
            .apply()
    }
}