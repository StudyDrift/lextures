import Foundation

/// Replays queued mutations in order with idempotency keys and backoff.
actor SyncEngine {
    private let outbox: OutboxStore
    private let client: APIClient
    private var isRunning = false
    private var metrics = OfflineSyncMetrics()
    private let metricsURL: URL

    init(outbox: OutboxStore, ownerKey: String, client: APIClient = APIClient()) {
        self.outbox = outbox
        self.client = client
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Offline", isDirectory: true)
            .appendingPathComponent(ownerKey, isDirectory: true)
        metricsURL = base.appendingPathComponent("sync-metrics.json")
        loadMetrics()
    }

    func currentMetrics() -> OfflineSyncMetrics {
        metrics
    }

    func sync(accessToken: String) async -> OfflineSyncMetrics {
        guard !isRunning else { return metrics }
        isRunning = true
        defer { isRunning = false }

        let pending = await outbox.pendingItems()
        for var item in pending {
            if await outbox.wasApplied(idempotencyKey: item.idempotencyKey) {
                item.status = .synced
                await outbox.update(item)
                continue
            }

            item.status = .syncing
            await outbox.update(item)

            do {
                let bodyData = item.bodyJSON.flatMap { $0.data(using: .utf8) }
                _ = try await client.requestRaw(
                    path: item.path,
                    method: item.method,
                    bodyData: bodyData,
                    authorized: true,
                    accessToken: accessToken,
                    idempotencyKey: item.idempotencyKey
                )
                item.status = .synced
                item.lastError = nil
                await outbox.update(item)
                await outbox.markApplied(idempotencyKey: item.idempotencyKey)
                metrics.successCount += 1
            } catch let error as APIError {
                switch error {
                case .httpStatus(let code, let message) where code == 409:
                    item.status = .conflict
                    item.lastError = message ?? "Server rejected this change."
                    metrics.conflictCount += 1
                case .httpStatus(let code, let message) where (500 ... 599).contains(code):
                    item.status = .failed
                    item.lastError = message ?? "Server error (HTTP \(code))."
                    metrics.failureCount += 1
                case .transport:
                    item.status = .queued
                    item.lastError = nil
                    await outbox.update(item)
                    persistMetrics()
                    return metrics
                default:
                    item.status = .failed
                    item.lastError = error.errorDescription
                    metrics.failureCount += 1
                }
                await outbox.update(item)
            } catch {
                item.status = .failed
                item.lastError = error.localizedDescription
                await outbox.update(item)
                metrics.failureCount += 1
            }
        }

        metrics.lastSyncAt = Date()
        persistMetrics()
        return metrics
    }

    // MARK: - Private

    private func loadMetrics() {
        guard
            let data = try? Data(contentsOf: metricsURL),
            let decoded = try? JSONDecoder().decode(OfflineSyncMetrics.self, from: data)
        else { return }
        metrics = decoded
    }

    private func persistMetrics() {
        guard let data = try? JSONEncoder().encode(metrics) else { return }
        try? data.write(to: metricsURL, options: .atomic)
    }
}
