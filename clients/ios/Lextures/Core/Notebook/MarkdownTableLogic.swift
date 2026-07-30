import Foundation

/// Column alignment for a GFM pipe table.
enum MarkdownTableAlign: String, Equatable, CaseIterable {
    case `default`
    case left
    case center
    case right
}

/// Pure table detection / blank-line healing / cell split (parity with web `normalize-markdown-tables`).
enum MarkdownTableLogic {
    /// Collapse blank lines inside GitHub-flavored markdown tables (web FR-3 parity).
    static func normalizeMarkdownTables(_ markdown: String) -> String {
        guard markdown.contains("|") else { return markdown }
        let lines = markdown.replacingOccurrences(of: "\r\n", with: "\n").components(separatedBy: "\n")
        var out: [String] = []
        var i = 0
        while i < lines.count {
            let line = lines[i]
            let sepIdx = nextNonEmptyIndex(lines, from: i + 1)
            let looksLikeHeader =
                isTableRowLine(line)
                && sepIdx < lines.count
                && isTableSeparatorLine(lines[sepIdx])
            if !looksLikeHeader {
                out.append(line)
                i += 1
                continue
            }
            var tableLines: [String] = [line]
            i += 1
            while i < lines.count {
                let L = lines[i]
                if L.trimmingCharacters(in: .whitespaces).isEmpty {
                    let next = nextNonEmptyIndex(lines, from: i + 1)
                    if next < lines.count && isTableRelatedLine(lines[next]) {
                        i = next
                        continue
                    }
                    break
                }
                if !isTableRelatedLine(L) { break }
                tableLines.append(L)
                i += 1
            }
            if tableLines.count >= 2 && isTableSeparatorLine(tableLines[1]) {
                out.append(contentsOf: tableLines)
            } else {
                out.append(line)
                if tableLines.count > 1 {
                    out.append(contentsOf: tableLines.dropFirst())
                }
            }
        }
        return out.joined(separator: "\n")
    }

    /// Try to parse a GFM table starting at `start`. Returns nil when not a valid table.
    static func parseTable(
        in lines: [String],
        start: Int
    ) -> (align: [MarkdownTableAlign], header: [String], rows: [[String]], end: Int)? {
        guard start < lines.count, isTableRowLine(lines[start]) else { return nil }
        let sepIdx = nextNonEmptyIndex(lines, from: start + 1)
        guard sepIdx < lines.count, isTableSeparatorLine(lines[sepIdx]) else { return nil }

        let header = splitCells(lines[start])
        let align = parseAlignments(lines[sepIdx], columnCount: header.count)
        var rows: [[String]] = []
        var i = sepIdx + 1
        while i < lines.count {
            let trimmed = lines[i].trimmingCharacters(in: .whitespaces)
            if trimmed.isEmpty { break }
            if !isTableRowLine(lines[i]) { break }
            if isTableSeparatorLine(lines[i]) { break }
            rows.append(padCells(splitCells(lines[i]), to: header.count))
            i += 1
        }
        return (align, header, rows, i)
    }

    static func isTableRowLine(_ line: String) -> Bool {
        let t = line.trimmingCharacters(in: .whitespaces)
        guard t.contains("|") else { return false }
        if t.range(of: #"^\|?[\s:|-]+$"#, options: .regularExpression) != nil && t.contains("-") {
            return false
        }
        if t.count >= 2 {
            let inner = t.dropFirst().dropLast()
            if inner.contains("|") { return true }
        }
        return t.hasPrefix("|") && t.hasSuffix("|")
    }

    static func isTableSeparatorLine(_ line: String) -> Bool {
        let t = line.trimmingCharacters(in: .whitespaces)
        guard t.contains("-") else { return false }
        let a = t.range(
            of: #"^\|?(\s*:?-{3,}:?\s*\|)+\s*:?-{3,}:?\s*\|?$"#,
            options: .regularExpression
        ) != nil
        let b = t.range(
            of: #"^\|?(\s*:?-{3,}:?\s*\|?)+\s*$"#,
            options: .regularExpression
        ) != nil
        return a || b
    }

    static func isTableRelatedLine(_ line: String) -> Bool {
        isTableRowLine(line) || isTableSeparatorLine(line)
    }

    static func splitCells(_ line: String) -> [String] {
        var t = line.trimmingCharacters(in: .whitespaces)
        if t.hasPrefix("|") { t = String(t.dropFirst()) }
        if t.hasSuffix("|") { t = String(t.dropLast()) }
        return t.split(separator: "|", omittingEmptySubsequences: false)
            .map { $0.trimmingCharacters(in: .whitespaces) }
    }

    static func parseAlignments(_ separator: String, columnCount: Int) -> [MarkdownTableAlign] {
        let cells = splitCells(separator)
        let parsed: [MarkdownTableAlign] = cells.map { cell in
            let c = cell.trimmingCharacters(in: .whitespaces)
            let left = c.hasPrefix(":")
            let right = c.hasSuffix(":")
            switch (left, right) {
            case (true, true): return .center
            case (false, true): return .right
            case (true, false): return .left
            default: return .default
            }
        }
        if parsed.count >= columnCount { return Array(parsed.prefix(columnCount)) }
        return parsed + Array(repeating: .default, count: columnCount - parsed.count)
    }

    static func padCells(_ cells: [String], to count: Int) -> [String] {
        if cells.count >= count { return Array(cells.prefix(count)) }
        return cells + Array(repeating: "", count: count - cells.count)
    }

    static func nextNonEmptyIndex(_ lines: [String], from: Int) -> Int {
        var j = from
        while j < lines.count && lines[j].trimmingCharacters(in: .whitespaces).isEmpty {
            j += 1
        }
        return j
    }

    /// Reconstruct pipe-table markdown (used when folding into the editor model).
    static func serialize(align: [MarkdownTableAlign], header: [String], rows: [[String]]) -> String {
        func row(_ cells: [String]) -> String {
            "| " + cells.joined(separator: " | ") + " |"
        }
        let sep = align.map { a -> String in
            switch a {
            case .left: return ":---"
            case .center: return ":---:"
            case .right: return "---:"
            case .default: return "---"
            }
        }
        var lines = [row(header), "| " + sep.joined(separator: " | ") + " |"]
        lines.append(contentsOf: rows.map(row))
        return lines.joined(separator: "\n")
    }
}
