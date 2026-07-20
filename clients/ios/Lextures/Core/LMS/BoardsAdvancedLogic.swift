import Foundation

/// MOB.8 helpers — templates, export polling, present ordering, governance gating.
enum BoardsAdvancedLogic {
    /// Templates/export/present/analytics require course boards + the mobile advanced gate.
    static func isAdvancedEnabled(
        courseEnabled: Bool,
        features: MobilePlatformFeatures
    ) -> Bool {
        courseEnabled && features.ffMobileBoardsAdvanced
    }

    static func canUseTemplates(
        courseEnabled: Bool,
        features: MobilePlatformFeatures,
        canCreate: Bool
    ) -> Bool {
        isAdvancedEnabled(courseEnabled: courseEnabled, features: features) && canCreate
    }

    static func canExportOrPresent(
        courseEnabled: Bool,
        features: MobilePlatformFeatures,
        canManage: Bool
    ) -> Bool {
        isAdvancedEnabled(courseEnabled: courseEnabled, features: features) && canManage
    }

    static func canViewBoardAnalytics(
        courseEnabled: Bool,
        features: MobilePlatformFeatures,
        canManage: Bool
    ) -> Bool {
        isAdvancedEnabled(courseEnabled: courseEnabled, features: features) && canManage
    }

    static func filterTemplates(
        _ templates: [BoardTemplate],
        scope: BoardTemplateScope?,
        query: String
    ) -> [BoardTemplate] {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return templates.filter { template in
            if let scope, template.scope.lowercased() != scope.rawValue { return false }
            guard !needle.isEmpty else { return true }
            let hay = [
                template.title,
                template.description,
                template.tags.joined(separator: " "),
            ].joined(separator: " ").lowercased()
            return hay.contains(needle)
        }
    }

    /// Exponential backoff for export/copy job polling (seconds): 0.5, 1, 2, 4… capped.
    static func pollDelaySeconds(attempt: Int, cap: TimeInterval = 8) -> TimeInterval {
        let base = 0.5 * pow(2.0, Double(max(0, attempt)))
        return min(cap, base)
    }

    static func isExportTerminal(_ status: String) -> Bool {
        let s = status.lowercased()
        return s == "done" || s == "failed"
    }

    static func isCopyTerminal(_ status: String) -> Bool {
        let s = status.lowercased()
        return s == "completed" || s == "failed"
    }

    static func exportFileExtension(format: BoardExportFormat) -> String {
        switch format {
        case .pdf: "pdf"
        case .csv: "csv"
        case .image: "png"
        }
    }

    static func exportMimeType(format: BoardExportFormat) -> String {
        switch format {
        case .pdf: "application/pdf"
        case .csv: "text/csv"
        case .image: "image/png"
        }
    }

    /// Present-mode ordering: section order, then card sortIndex (web present-mode parity).
    static func orderedPostsForPresent(posts: [BoardPost], sections: [BoardSection]) -> [BoardPost] {
        let secOrder = Dictionary(uniqueKeysWithValues: sections.map { ($0.id, $0.sortIndex) })
        return posts.sorted { a, b in
            let aSec = a.sectionId.flatMap { secOrder[$0] } ?? Double.greatestFiniteMagnitude
            let bSec = b.sectionId.flatMap { secOrder[$0] } ?? Double.greatestFiniteMagnitude
            if aSec != bSec { return aSec < bSec }
            return a.sortIndex < b.sortIndex
        }
    }

    static func postBodyText(_ post: BoardPost) -> String {
        if let text = post.body?.text?.trimmingCharacters(in: .whitespacesAndNewlines), !text.isEmpty {
            return text
        }
        if let html = post.body?.html, !html.isEmpty {
            return stripHTML(html)
        }
        let title = post.title.trimmingCharacters(in: .whitespacesAndNewlines)
        return title
    }

    private static func stripHTML(_ html: String) -> String {
        var result = html
        if let regex = try? NSRegularExpression(pattern: "<[^>]+>", options: .caseInsensitive) {
            let range = NSRange(result.startIndex..., in: result)
            result = regex.stringByReplacingMatches(in: result, options: [], range: range, withTemplate: "")
        }
        return result
            .replacingOccurrences(of: "&nbsp;", with: " ")
            .replacingOccurrences(of: "&amp;", with: "&")
            .replacingOccurrences(of: "&lt;", with: "<")
            .replacingOccurrences(of: "&gt;", with: ">")
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func formatStorageBytes(_ bytes: Int64) -> String {
        BoardsLogic.formatFileSize(bytes)
    }

    static func parseBoardCapDraft(_ raw: String) -> Int? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return nil }
        return Int(trimmed)
    }
}

enum BoardsAdvancedObservability {
    private static var counters: [String: Int] = [:]
    private static let lock = NSLock()

    static func record(_ event: String, attributes: [String: String] = [:]) {
        lock.lock()
        defer { lock.unlock() }
        let key = attributes.isEmpty
            ? event
            : event + "|" + attributes.keys.sorted().map { "\($0)=\(attributes[$0] ?? "")" }.joined(separator: ",")
        counters[key, default: 0] += 1
    }

    static func count(for event: String) -> Int {
        lock.lock()
        defer { lock.unlock() }
        return counters.filter { $0.key == event || $0.key.hasPrefix(event + "|") }.values.reduce(0, +)
    }

    #if DEBUG
    static func resetForTests() {
        lock.lock()
        counters.removeAll()
        lock.unlock()
    }
    #endif
}
