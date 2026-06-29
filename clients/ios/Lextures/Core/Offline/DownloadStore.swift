import Foundation

private struct DownloadRecord: Codable {
    var savedAt: Date
    var fileName: String
    var mimeType: String?
    var sizeBytes: Int
    var lastAccessAt: Date
}

/// Local storage for downloaded course files and prefetched content pages (shared with M3.2).
actor DownloadStore {
    private let rootURL: URL
    private let indexURL: URL
    private var index: [String: DownloadRecord] = [:]
    private let maxBytes: Int

    init(ownerKey: String, maxBytes: Int = OfflineStorageBudget.defaultMaxBytes) {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
            .appendingPathComponent("Offline", isDirectory: true)
            .appendingPathComponent(ownerKey, isDirectory: true)
            .appendingPathComponent("downloads", isDirectory: true)
        rootURL = base
        indexURL = base.appendingPathComponent("index.json")
        self.maxBytes = maxBytes
        try? FileManager.default.createDirectory(at: rootURL, withIntermediateDirectories: true)
        loadIndex()
    }

    func isDownloaded(key: String) -> Bool {
        index[key] != nil && FileManager.default.fileExists(atPath: fileURL(for: key).path)
    }

    func save(key: String, data: Data, fileName: String, mimeType: String?) throws {
        let url = fileURL(for: key)
        try data.write(to: url, options: .atomic)
        let now = Date()
        index[key] = DownloadRecord(
            savedAt: now,
            fileName: fileName,
            mimeType: mimeType,
            sizeBytes: data.count,
            lastAccessAt: now
        )
        evictIfNeeded()
        persistIndex()
    }

    func loadData(key: String) -> Data? {
        guard var record = index[key] else { return nil }
        record.lastAccessAt = Date()
        index[key] = record
        persistIndex()
        return try? Data(contentsOf: fileURL(for: key))
    }

    func remove(key: String) {
        index.removeValue(forKey: key)
        try? FileManager.default.removeItem(at: fileURL(for: key))
        persistIndex()
    }

    func downloadedKeys() -> [String] {
        Array(index.keys)
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

    private func fileURL(for key: String) -> URL {
        let safe = key.addingPercentEncoding(withAllowedCharacters: .alphanumerics) ?? key
        return rootURL.appendingPathComponent(safe)
    }

    private func loadIndex() {
        guard
            let data = try? Data(contentsOf: indexURL),
            let decoded = try? JSONDecoder().decode([String: DownloadRecord].self, from: data)
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
            remove(key: key)
        }
    }
}
