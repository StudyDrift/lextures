import Foundation

/// Course accessibility / alt-text review helpers (M13.8).
enum CourseAccessibilityReviewLogic {
    static let decorativeTitleMarker = "lex-decorative"
    static let pageSize = 20

    struct MarkdownImageRef: Identifiable, Equatable, Hashable {
        var globalIndex: Int
        var alt: String
        var src: String
        var title: String?
        var decorative: Bool
        var hasValidAlt: Bool
        var line: Int

        var id: Int { globalIndex }
    }

    struct ImageAltDraft: Equatable, Hashable {
        var alt: String
        var decorative: Bool
    }

    static func cacheKey(courseCode: String) -> String {
        "course:\(courseCode):accessibility"
    }

    static func saveMarkdownIdempotencyKey(courseCode: String, itemId: String, kind: String) -> String {
        "course-accessibility:\(courseCode):\(kind):\(itemId):markdown"
    }

    static func markdownPatchPath(courseCode: String, itemId: String, kind: String) -> String? {
        switch kind {
        case "content_page":
            return "/api/v1/courses/\(courseCode)/content-pages/\(itemId)"
        case "assignment":
            return "/api/v1/courses/\(courseCode)/assignments/\(itemId)"
        default:
            return nil
        }
    }

    static func supportsInlineEdit(kind: String) -> Bool {
        kind == "content_page" || kind == "assignment"
    }

    static func coveragePercent(withAlt: Int, total: Int) -> Int {
        if total <= 0 { return 100 }
        return Int((Double(withAlt) / Double(total) * 100).rounded())
    }

    static func formatCoverageLabel(withAlt: Int, total: Int) -> String {
        let pct = coveragePercent(withAlt: withAlt, total: total)
        return L.format("mobile.courseSettings.accessibility.coverageValue", pct, withAlt, total)
    }

    static func paginatedUncoveredItems(_ items: [UncoveredAccessibilityItem], page: Int) -> [UncoveredAccessibilityItem] {
        let end = min(items.count, max(0, page + 1) * pageSize)
        return Array(items.prefix(end))
    }

    static func hasMorePages(items: [UncoveredAccessibilityItem], page: Int) -> Bool {
        items.count > (page + 1) * pageSize
    }

    static func scanMarkdownImages(_ markdown: String) -> [MarkdownImageRef] {
        let pattern = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/
        var refs: [MarkdownImageRef] = []
        var globalIndex = 0
        let lines = markdown.split(separator: "\n", omittingEmptySubsequences: false)
        for (lineOffset, lineSub) in lines.enumerated() {
            let line = String(lineSub)
            var searchStart = line.startIndex
            while searchStart < line.endIndex {
                guard let match = line[searchStart...].firstMatch(of: pattern) else { break }
                let alt = String(match.output.1)
                let src = String(match.output.2)
                let title = match.output.3.map(String.init)
                let decorative = title == decorativeTitleMarker
                let hasValidAlt = decorative || !alt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                refs.append(MarkdownImageRef(
                    globalIndex: globalIndex,
                    alt: alt,
                    src: src,
                    title: title,
                    decorative: decorative,
                    hasValidAlt: hasValidAlt,
                    line: lineOffset + 1
                ))
                globalIndex += 1
                searchStart = match.range.upperBound
            }
        }
        return refs
    }

    static func missingImages(_ markdown: String) -> [MarkdownImageRef] {
        scanMarkdownImages(markdown).filter { !$0.hasValidAlt }
    }

    static func applyAltTextUpdate(
        in markdown: String,
        imageIndex: Int,
        alt: String,
        decorative: Bool
    ) -> String? {
        applyAltTextUpdates(in: markdown, updates: [(imageIndex: imageIndex, alt: alt, decorative: decorative)])
    }

    static func applyAltTextUpdates(
        in markdown: String,
        updates: [(imageIndex: Int, alt: String, decorative: Bool)]
    ) -> String? {
        guard !updates.isEmpty else { return markdown }
        let pattern = try? NSRegularExpression(pattern: #"!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)"#)
        guard let pattern else { return nil }
        let ns = markdown as NSString
        let matches = pattern.matches(in: markdown, range: NSRange(location: 0, length: ns.length))
        var result = markdown
        for update in updates.sorted(by: { $0.imageIndex > $1.imageIndex }) {
            guard update.imageIndex >= 0, update.imageIndex < matches.count else { return nil }
            let match = matches[update.imageIndex]
            let src = ns.substring(with: match.range(at: 2))
            let replacement: String
            if update.decorative {
                replacement = "![](\(src) \"\(decorativeTitleMarker)\")"
            } else {
                let trimmedAlt = update.alt.trimmingCharacters(in: .whitespacesAndNewlines)
                let escapedAlt = trimmedAlt.replacingOccurrences(of: "]", with: "\\]")
                replacement = "![\(escapedAlt)](\(src))"
            }
            guard let range = Range(match.range, in: result) else { return nil }
            result.replaceSubrange(range, with: replacement)
        }
        return result
    }

    static func kindLabelKey(for kind: String) -> String {
        switch kind {
        case "assignment": return "mobile.courseSettings.accessibility.kind.assignment"
        case "content_page": return "mobile.courseSettings.accessibility.kind.contentPage"
        default: return "mobile.courseSettings.accessibility.kind.other"
        }
    }

    static func itemMissingLabel(missing: Int, total: Int) -> String {
        L.format("mobile.courseSettings.accessibility.itemMissing", missing, total)
    }

    static func drafts(from images: [MarkdownImageRef]) -> [Int: ImageAltDraft] {
        Dictionary(uniqueKeysWithValues: images.map { image in
            (image.globalIndex, ImageAltDraft(alt: image.alt, decorative: image.decorative))
        })
    }

    static func pendingUpdates(
        images: [MarkdownImageRef],
        drafts: [Int: ImageAltDraft]
    ) -> [(imageIndex: Int, alt: String, decorative: Bool)] {
        images.compactMap { image in
            guard let draft = drafts[image.globalIndex] else { return nil }
            let trimmedAlt = draft.alt.trimmingCharacters(in: .whitespacesAndNewlines)
            let resolved = draft.decorative || !trimmedAlt.isEmpty
            guard resolved else { return nil }
            if draft.decorative == image.decorative && trimmedAlt == image.alt.trimmingCharacters(in: .whitespacesAndNewlines) {
                return nil
            }
            return (imageIndex: image.globalIndex, alt: trimmedAlt, decorative: draft.decorative)
        }
    }
}
