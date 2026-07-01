import Foundation
import Observation

/// Shared offline coordinator: cache reads, download storage, and write outbox replay.
@MainActor
@Observable
final class OfflineService {
    static let shared = OfflineService()

    private(set) var pendingCount = 0
    private(set) var storageBytes = 0
    private(set) var outboxItems: [OutboxItem] = []
    private(set) var metrics = OfflineSyncMetrics()

    private var ownerKey = "anonymous"
    private var cacheStore = CacheStore(ownerKey: "anonymous")
    private var outboxStore = OutboxStore(ownerKey: "anonymous")
    private var downloadStore = DownloadStore(ownerKey: "anonymous")
    private var syncEngine: SyncEngine

    private init() {
        let outbox = OutboxStore(ownerKey: "anonymous")
        outboxStore = outbox
        syncEngine = SyncEngine(outbox: outbox, ownerKey: "anonymous")
    }

    func configure(accessToken: String?) {
        let key = NotebookStore.jwtSubject(from: accessToken) ?? "anonymous"
        if key != ownerKey {
            ownerKey = key
            cacheStore = CacheStore(ownerKey: key)
            outboxStore = OutboxStore(ownerKey: key)
            downloadStore = DownloadStore(ownerKey: key)
            syncEngine = SyncEngine(outbox: outboxStore, ownerKey: key)
        }
        Task { await refreshState() }
    }

    func clearAllOnLogout() {
        Task {
            await cacheStore.clearAll()
            await downloadStore.clearAll()
            await outboxStore.clearAll()
            await refreshState()
        }
    }

    /// Cache-first fetch: returns cached data immediately; refreshes when online.
    func cachedFetch<T: Codable>(
        key: String,
        accessToken: String,
        fetch: () async throws -> T
    ) async throws -> (value: T, cached: Cached<T>?) {
        let online = NetworkMonitor.shared.isOnline
        let existing = await cacheStore.get(T.self, key: key)

        if online {
            do {
                let fresh = try await fetch()
                try await cacheStore.put(fresh, key: key)
                await refreshState()
                return (fresh, Cached(value: fresh, fetchedAt: Date()))
            } catch {
                if let existing {
                    return (existing.value, existing)
                }
                throw error
            }
        }

        if let existing {
            return (existing.value, existing)
        }
        throw APIError.transport(URLError(.notConnectedToInternet))
    }

    /// Enqueue a mutation when offline (or when caller prefers queued replay).
    func enqueueMutation(
        method: String,
        path: String,
        body: (any Encodable)?,
        label: String,
        accessToken: String?,
        preferQueue: Bool = false,
        idempotencyKey: String? = nil
    ) async throws -> OutboxItem {
        let bodyJSON: String?
        if let body {
            let data = try JSONEncoder().encode(AnyEncodable(body))
            bodyJSON = String(data: data, encoding: .utf8)
        } else {
            bodyJSON = nil
        }

        let online = NetworkMonitor.shared.isOnline
        if online && !preferQueue, let accessToken {
            let bodyData = bodyJSON.flatMap { $0.data(using: .utf8) }
            let resolvedKey = idempotencyKey ?? UUID().uuidString.lowercased()
            do {
                _ = try await APIClient().requestRaw(
                    path: path,
                    method: method,
                    bodyData: bodyData,
                    authorized: true,
                    accessToken: accessToken,
                    idempotencyKey: resolvedKey
                )
                await outboxStore.markApplied(idempotencyKey: resolvedKey)
                await refreshState()
                return OutboxItem(
                    id: resolvedKey,
                    createdAt: Date(),
                    sequence: -1,
                    method: method,
                    path: path,
                    bodyJSON: bodyJSON,
                    label: label,
                    status: .synced,
                    lastError: nil
                )
            } catch let error as APIError {
                if case .transport = error {
                    // Fall through to outbox enqueue.
                } else {
                    throw error
                }
            }
        }

        let item = await outboxStore.enqueue(
            method: method,
            path: path,
            bodyJSON: bodyJSON,
            label: label,
            id: idempotencyKey
        )
        await refreshState()
        return item
    }

    func downloadContent(key: String, data: Data, fileName: String, mimeType: String?) async throws {
        try await downloadStore.save(key: key, data: data, fileName: fileName, mimeType: mimeType)
        await refreshState()
    }

    func downloadedData(key: String) async -> Data? {
        await downloadStore.loadData(key: key)
    }

    func isDownloaded(key: String) async -> Bool {
        await downloadStore.isDownloaded(key: key)
    }

    func syncNow(accessToken: String?) async {
        guard let accessToken, NetworkMonitor.shared.isOnline else { return }
        metrics = await syncEngine.sync(accessToken: accessToken)
        await refreshState()
    }

    func retryOutboxItem(id: String, accessToken: String?) async {
        await outboxStore.retry(id: id)
        await syncNow(accessToken: accessToken)
    }

    func refreshState() async {
        pendingCount = await outboxStore.pendingCount()
        outboxItems = await outboxStore.allItems()
        let cacheBytes = await cacheStore.totalSizeBytes()
        let downloadBytes = await downloadStore.totalSizeBytes()
        storageBytes = cacheBytes + downloadBytes
        metrics = await syncEngine.currentMetrics()
    }

    func clearStorage() async {
        await cacheStore.clearAll()
        await downloadStore.clearAll()
        await refreshState()
    }
}

private struct AnyEncodable: Encodable {
    private let encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        encode = value.encode
    }

    func encode(to encoder: Encoder) throws {
        try encode(encoder)
    }
}
