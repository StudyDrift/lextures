package com.lextures.android.core.notebook

import com.lextures.android.core.lms.NotebooksApi
import java.time.Instant
import java.time.OffsetDateTime

/**
 * Server sync for student notebooks — last-write-wins by `updatedAt`, parity with web
 * `student-notebook-sync`. Device storage stays the editor's source of truth; sync runs
 * fire-and-forget around it.
 */
object NotebookSync {
    /**
     * Pull all server notebooks and merge into the device store. Server copy wins when newer;
     * local copies that are newer or missing on the server are pushed back.
     * Returns true when any local notebook changed.
     */
    suspend fun pull(store: NotebookStore, accessToken: String?): Boolean {
        if (accessToken.isNullOrBlank()) return false
        val entries = runCatching { NotebooksApi.fetchAll(accessToken) }.getOrNull() ?: return false

        var changed = false
        val serverCodes = mutableSetOf<String>()
        for (entry in entries) {
            val server = entry.data ?: continue
            serverCodes.add(entry.courseCode)
            if (!store.exists(entry.courseCode)) {
                store.saveFromServer(entry.courseCode, server)
                changed = true
                continue
            }
            val local = store.load(entry.courseCode)
            val serverTime = parseIso(entry.updatedAt)
            val localTime = parseIso(local.updatedAt)
            if (serverTime > localTime) {
                store.saveFromServer(entry.courseCode, server)
                changed = true
            } else if (localTime > serverTime) {
                push(store, entry.courseCode, accessToken)
            }
        }
        for (code in store.allCourseCodes()) {
            if (code !in serverCodes) push(store, code, accessToken)
        }
        return changed
    }

    /** Push one notebook to the server, swallowing transport errors (next save retries). */
    suspend fun push(store: NotebookStore, courseCode: String, accessToken: String?) {
        if (accessToken.isNullOrBlank()) return
        val notebook = store.load(courseCode)
        runCatching { NotebooksApi.put(courseCode, notebook, accessToken) }
    }

    private fun parseIso(value: String): Instant =
        runCatching { OffsetDateTime.parse(value).toInstant() }.getOrElse { Instant.EPOCH }
}
