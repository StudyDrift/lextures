package com.lextures.android.core.offline

import android.content.Context
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.network.ApiClient
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.notebook.NotebookStore
import java.util.UUID
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.KSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

class OfflineService private constructor(context: Context) {
    private val appContext = context.applicationContext
    private val json = Json { ignoreUnknownKeys = true }
    private val apiClient = ApiClient()

    private var ownerKey = "anonymous"
    private lateinit var cacheStore: CacheStore
    private lateinit var outboxStore: OutboxStore
    private lateinit var downloadStore: DownloadStore
    private lateinit var syncWorker: SyncWorker
    val networkMonitor = NetworkMonitor(appContext)

    private val _pendingCount = MutableStateFlow(0)
    val pendingCount: StateFlow<Int> = _pendingCount.asStateFlow()

    private val _storageBytes = MutableStateFlow(0L)
    val storageBytes: StateFlow<Long> = _storageBytes.asStateFlow()

    private val _outboxItems = MutableStateFlow<List<OutboxItem>>(emptyList())
    val outboxItems: StateFlow<List<OutboxItem>> = _outboxItems.asStateFlow()

    init {
        rebuildStores("anonymous")
    }

    fun configure(accessToken: String?) {
        val key = NotebookStore.jwtSubject(accessToken) ?: "anonymous"
        if (key != ownerKey) {
            ownerKey = key
            rebuildStores(key)
        }
        refreshState()
    }

    fun clearAllOnLogout() {
        cacheStore.clearAll()
        downloadStore.clearAll()
        outboxStore.clearAll()
        refreshState()
    }

    suspend fun <T> cachedFetch(
        key: String,
        accessToken: String,
        serializer: KSerializer<T>,
        fetch: suspend () -> T,
    ): Pair<T, Cached<T>?> {
        val online = networkMonitor.isOnline.value
        val existing = cacheStore.get(key, serializer)

        if (online) {
            return try {
                val fresh = fetch()
                cacheStore.put(key, fresh, serializer)
                refreshState()
                fresh to Cached(fresh, java.time.Instant.now())
            } catch (e: Exception) {
                if (existing != null) {
                    existing.value to existing
                } else {
                    throw e
                }
            }
        }

        if (existing != null) {
            return existing.value to existing
        }
        throw ApiError.Transport(java.io.IOException("Device is offline"))
    }

    suspend fun enqueueMutation(
        method: String,
        path: String,
        bodyJson: String?,
        label: String,
        accessToken: String?,
        preferQueue: Boolean = false,
        idempotencyKey: String? = null,
    ): OutboxItem {
        val online = networkMonitor.isOnline.value
        if (online && !preferQueue && !accessToken.isNullOrBlank()) {
            val resolvedKey = idempotencyKey ?: UUID.randomUUID().toString()
            try {
                apiClient.requestRaw(
                    path = path,
                    method = method,
                    body = bodyJson,
                    accessToken = accessToken,
                    idempotencyKey = resolvedKey,
                )
                outboxStore.markApplied(resolvedKey)
                refreshState()
                return OutboxItem(
                    id = resolvedKey,
                    method = method,
                    path = path,
                    bodyJson = bodyJson,
                    label = label,
                    status = OutboxStatus.Synced.name,
                )
            } catch (e: ApiError.Transport) {
                // Fall through to outbox.
            } catch (e: Exception) {
                throw e
            }
        }

        val item = outboxStore.enqueue(method, path, bodyJson, label, idempotencyKey)
        refreshState()
        return item
    }

    suspend fun syncNow(accessToken: String?) {
        if (accessToken.isNullOrBlank() || !networkMonitor.isOnline.value) return
        syncWorker.sync(accessToken)
        refreshState()
    }

    suspend fun retryOutboxItem(id: String, accessToken: String?) {
        outboxStore.retry(id)
        syncNow(accessToken)
    }

    fun refreshState() {
        _pendingCount.value = outboxStore.pendingCount()
        _outboxItems.value = outboxStore.allItems()
        _storageBytes.value = cacheStore.totalSizeBytes() + downloadStore.totalSizeBytes()
    }

    fun clearStorage() {
        cacheStore.clearAll()
        downloadStore.clearAll()
        refreshState()
    }

    fun downloadContent(key: String, data: ByteArray, fileName: String, mimeType: String?) {
        downloadStore.save(key, data, fileName, mimeType)
        refreshState()
    }

    fun downloadedData(key: String): ByteArray? = downloadStore.loadData(key)

    fun isDownloaded(key: String): Boolean = downloadStore.isDownloaded(key)

    private fun rebuildStores(key: String) {
        cacheStore = CacheStore(appContext, key)
        outboxStore = OutboxStore(appContext, key)
        downloadStore = DownloadStore(appContext, key)
        syncWorker = SyncWorker(appContext, outboxStore, key, apiClient)
    }

    companion object {
        @Volatile
        private var instance: OfflineService? = null

        fun get(context: Context): OfflineService =
            instance ?: synchronized(this) {
                instance ?: OfflineService(context.applicationContext).also { instance = it }
            }
    }
}

/** Convenience for course list cached fetch. */
suspend fun OfflineService.fetchCoursesCached(
    accessToken: String,
    fetch: suspend () -> List<CourseSummary>,
): Pair<List<CourseSummary>, Cached<List<CourseSummary>>?> =
    cachedFetch(
        key = OfflineCacheKey.courses(),
        accessToken = accessToken,
        serializer = kotlinx.serialization.builtins.ListSerializer(CourseSummary.serializer()),
        fetch = fetch,
    )
