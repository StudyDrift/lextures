import Foundation

private struct CacheEntry: Codable {
    var fetchedAt: Date
    var payload: Data
    var sizeBytes: Int
    var lastAccessAt: Date
}

/// Typed on-disk read cache with LRU eviction under a shared storage budget.
actor CacheStore {
    private let rootURL: URL
    private let indexURL: URL
    private var index: [String: CacheEntry] = [:]
    private let maxBytes: Int

    init(ownerKey: String, maxBytes: Int = OfflineStorageBudget.defaultMaxBytes) {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Offline", isDirectory: true)
            .appendingPathComponent(ownerKey, isDirectory: true)
            .appendingPathComponent("cache", isDirectory: true)
        rootURL = base
        indexURL = base.appendingPathComponent("index.json")
        self.maxBytes = maxBytes
        try? FileManager.default.createDirectory(at: rootURL, withIntermediateDirectories: true)
        loadIndex()
    }

    func get<T: Decodable>(_ type: T.Type, key: String) -> Cached<T>? {
        guard let entry = index[key] else { return nil }
        var mutable = entry
        mutable.lastAccessAt = Date()
        index[key] = mutable
        persistIndex()
        guard let value = try? JSONDecoder().decode(T.self, from: entry.payload) else { return nil }
        return Cached(value: value, fetchedAt: entry.fetchedAt)
    }

    func put<T: Encodable>(_ value: T, key: String) throws {
        let data = try JSONEncoder().encode(value)
        let now = Date()
        index[key] = CacheEntry(
            fetchedAt: now,
            payload: data,
            sizeBytes: data.count,
            lastAccessAt: now
        )
        evictIfNeeded()
        persistIndex()
    }

    func remove(key: String) {
        index.removeValue(forKey: key)
        persistIndex()
    }

    func totalSizeBytes() -> Int {
        index.values.reduce(0) { $0 + $1.sizeBytes }
    }

    func clearAll() {
        index.removeAll()
        try? FileManager.default.removeItem(at: rootURL)
        try? FileManager.default.createDirectory(at: rootURL, withIntermediateDirectories: true)
        persistIndex()
    }

    // MARK: - Private

    private func loadIndex() {
        guard
            let data = try? Data(contentsOf: indexURL),
            let decoded = try? JSONDecoder().decode([String: CacheEntry].self, from: data)
        else { return }
        index = decoded
    }

    private func persistIndex() {
        guard let data = try? JSONEncoder().encode(index) else { return }
        try? data.write(to: indexURL, options: .atomic)
    }

    private func evictIfNeeded() {
        var total = index.values.reduce(0) { $0 + $1.sizeBytes }
        guard total > maxBytes else { return }
        let sortedKeys = index.keys.sorted {
            (index[$0]?.lastAccessAt ?? .distantPast) < (index[$1]?.lastAccessAt ?? .distantPast)
        }
        for key in sortedKeys where total > maxBytes {
            total -= index[key]?.sizeBytes ?? 0
            index.removeValue(forKey: key)
        }
    }
}
