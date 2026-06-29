import Foundation

/// Ordered write queue persisted locally until replay succeeds.
actor OutboxStore {
    private let fileURL: URL
    private var items: [OutboxItem] = []
    private var nextSequence = 0
    private var appliedKeys: Set<String> = []
    private let appliedKeysURL: URL

    init(ownerKey: String) {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Offline", isDirectory: true)
            .appendingPathComponent(ownerKey, isDirectory: true)
        try? FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
        fileURL = base.appendingPathComponent("outbox.json")
        appliedKeysURL = base.appendingPathComponent("applied-idempotency-keys.json")
        load()
    }

    func enqueue(method: String, path: String, bodyJSON: String?, label: String) -> OutboxItem {
        let item = OutboxItem(
            id: UUID().uuidString.lowercased(),
            createdAt: Date(),
            sequence: nextSequence,
            method: method,
            path: path,
            bodyJSON: bodyJSON,
            label: label,
            status: .queued,
            lastError: nil
        )
        nextSequence += 1
        items.append(item)
        persist()
        return item
    }

    func pendingItems() -> [OutboxItem] {
        items.filter { $0.status == .queued || $0.status == .failed }
            .sorted { $0.sequence < $1.sequence }
    }

    func pendingCount() -> Int {
        items.filter { $0.status == .queued || $0.status == .failed || $0.status == .syncing }.count
    }

    func allItems() -> [OutboxItem] {
        items.sorted { $0.sequence < $1.sequence }
    }

    func update(_ item: OutboxItem) {
        guard let index = items.firstIndex(where: { $0.id == item.id }) else { return }
        items[index] = item
        persist()
    }

    func markApplied(idempotencyKey: String) {
        appliedKeys.insert(idempotencyKey)
        persistAppliedKeys()
    }

    func wasApplied(idempotencyKey: String) -> Bool {
        appliedKeys.contains(idempotencyKey)
    }

    func retry(id: String) {
        guard let index = items.firstIndex(where: { $0.id == id }) else { return }
        items[index].status = .queued
        items[index].lastError = nil
        persist()
    }

    func clearAll() {
        items.removeAll()
        nextSequence = 0
        appliedKeys.removeAll()
        persist()
        persistAppliedKeys()
    }

    // MARK: - Private

    private struct PersistedOutbox: Codable {
        var items: [OutboxItem]
        var nextSequence: Int
    }

    private func load() {
        if
            let data = try? Data(contentsOf: fileURL),
            let decoded = try? JSONDecoder().decode(PersistedOutbox.self, from: data) {
            items = decoded.items
            nextSequence = decoded.nextSequence
        }
        if
            let data = try? Data(contentsOf: appliedKeysURL),
            let keys = try? JSONDecoder().decode(Set<String>.self, from: data) {
            appliedKeys = keys
        }
    }

    private func persist() {
        let payload = PersistedOutbox(items: items, nextSequence: nextSequence)
        guard let data = try? JSONEncoder().encode(payload) else { return }
        try? data.write(to: fileURL, options: .atomic)
    }

    private func persistAppliedKeys() {
        guard let data = try? JSONEncoder().encode(appliedKeys) else { return }
        try? data.write(to: appliedKeysURL, options: .atomic)
    }
}
