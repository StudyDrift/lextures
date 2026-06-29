package com.lextures.android.core.offline

import android.content.Context
import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import java.io.File
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/** Replays queued mutations in order with idempotency keys and backoff. */
class SyncWorker(
    context: Context,
    private val outbox: OutboxStore,
    ownerKey: String,
    private val apiClient: ApiClient = ApiClient(),
) {
    private val metricsFile = File(context.applicationContext.filesDir, "offline/$ownerKey/sync-metrics.json")
    private val json = Json { ignoreUnknownKeys = true }
    private var metrics = loadMetrics()
    private var isRunning = false

    fun currentMetrics(): OfflineSyncMetrics = metrics

    suspend fun sync(accessToken: String): OfflineSyncMetrics {
        if (isRunning) return metrics
        isRunning = true
        try {
            for (var item in outbox.pendingItems()) {
                if (outbox.wasApplied(item.idempotencyKey)) {
                    outbox.update(item.copy(status = OutboxStatus.Synced.name))
                    continue
                }

                outbox.update(item.copy(status = OutboxStatus.Syncing.name))
                try {
                    apiClient.requestRaw(
                        path = item.path,
                        method = item.method,
                        body = item.bodyJson,
                        accessToken = accessToken,
                        idempotencyKey = item.idempotencyKey,
                    )
                    outbox.update(item.copy(status = OutboxStatus.Synced.name, lastError = null))
                    outbox.markApplied(item.idempotencyKey)
                    metrics = metrics.copy(successCount = metrics.successCount + 1)
                } catch (e: ApiError.HttpStatus) {
                    when {
                        e.code == 409 -> {
                            outbox.update(
                                item.copy(
                                    status = OutboxStatus.Conflict.name,
                                    lastError = e.message ?: "Server rejected this change.",
                                ),
                            )
                            metrics = metrics.copy(conflictCount = metrics.conflictCount + 1)
                        }
                        e.code in 500..599 -> {
                            outbox.update(
                                item.copy(
                                    status = OutboxStatus.Failed.name,
                                    lastError = e.message ?: "Server error (HTTP ${e.code}).",
                                ),
                            )
                            metrics = metrics.copy(failureCount = metrics.failureCount + 1)
                        }
                        else -> {
                            outbox.update(
                                item.copy(
                                    status = OutboxStatus.Failed.name,
                                    lastError = e.message ?: "Request failed (HTTP ${e.code}).",
                                ),
                            )
                            metrics = metrics.copy(failureCount = metrics.failureCount + 1)
                        }
                    }
                } catch (_: ApiError.Transport) {
                    outbox.update(item.copy(status = OutboxStatus.Queued.name, lastError = null))
                    persistMetrics()
                    return metrics
                }
            }
            metrics = metrics.copy(lastSyncEpochMs = System.currentTimeMillis())
            persistMetrics()
            return metrics
        } finally {
            isRunning = false
        }
    }

    private fun loadMetrics(): OfflineSyncMetrics {
        if (!metricsFile.exists()) return OfflineSyncMetrics()
        return runCatching { json.decodeFromString<OfflineSyncMetrics>(metricsFile.readText()) }
            .getOrDefault(OfflineSyncMetrics())
    }

    private fun persistMetrics() {
        metricsFile.parentFile?.mkdirs()
        metricsFile.writeText(json.encodeToString(metrics))
    }
}
