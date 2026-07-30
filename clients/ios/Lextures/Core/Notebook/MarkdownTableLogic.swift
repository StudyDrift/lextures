import Foundation

/// Column alignment for a GFM pipe table.
enum MarkdownTableAlign: String, Equatable, CaseIterable {
    case `default`
    case left
    case center
    case right
}

/// Result of parsing a GFM pipe table from a line slice.
struct ParsedMarkdownTable: Equatable {
    let align: [MarkdownTableAlign]
    let header: [String]
    let rows: [[String]]
    /// Index of the first line after the table.
    let endIndex: Int
}

/// Pure table detection / blank-line healing / cell split (parity with web `normalize-markdown-tables`).
enum MarkdownTableLogic {
    /// Collapse blank lines inside GitHub-flavored markdown tables (web FR-3 parity).
    static func normalizeMarkdownTables(_ markdown: String) -> String {
        guard markdown.contains("|") else { return markdown }
        let lines = markdown.replacingOccurrences(of: "\r\n", with: "\n").components(separatedBy: "\n")
        var out: [String] = []
        var index = 0
        while index < lines.count {
            let line = lines[index]
            let separatorIndex = nextNonEmptyIndex(lines, from: index + 1)
            let looksLikeHeader =
                isTableRowLine(line)
                && separatorIndex < lines.count
                && isTableSeparatorLine(lines[separatorIndex])
            if !looksLikeHeader {
                out.append(line)
                index += 1
                continue
            }
            var tableLines: [String] = [line]
            index += 1
            while index < lines.count {
                let candidate = lines[index]
                if candidate.trimmingCharacters(in: .whitespaces).isEmpty {
                    let nextIndex = nextNonEmptyIndex(lines, from: index + 1)
                    if nextIndex < lines.count && isTableRelatedLine(lines[nextIndex]) {
                        index = nextIndex
                        continue
                    }
                    break
                }
                if !isTableRelatedLine(candidate) { break }
                tableLines.append(candidate)
                index += 1
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
    static func parseTable(in lines: [String], start: Int) -> ParsedMarkdownTable? {
        guard start < lines.count, isTableRowLine(lines[start]) else { return nil }
        let separatorIndex = nextNonEmptyIndex(lines, from: start + 1)
        guard separatorIndex < lines.count, isTableSeparatorLine(lines[separatorIndex]) else { return nil }

        let header = splitCells(lines[start])
        let align = parseAlignments(lines[separatorIndex], columnCount: header.count)
        var rows: [[String]] = []
        var index = separatorIndex + 1
        while index < lines.count {
            let trimmed = lines[index].trimmingCharacters(in: .whitespaces)
            if trimmed.isEmpty { break }
            if !isTableRowLine(lines[index]) { break }
            if isTableSeparatorLine(lines[index]) { break }
            rows.append(padCells(splitCells(lines[index]), to: header.count))
            index += 1
        }
        return ParsedMarkdownTable(align: align, header: header, rows: rows, endIndex: index)
    }

    static func isTableRowLine(_ line: String) -> Bool {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        guard trimmed.contains("|") else { return false }
        if trimmed.range(of: #"^\|?[\s:|-]+$"#, options: .regularExpression) != nil && trimmed.contains("-") {
            return false
        }
        if trimmed.count >= 2 {
            let inner = trimmed.dropFirst().dropLast()
            if inner.contains("|") { return true }
        }
        return trimmed.hasPrefix("|") && trimmed.hasSuffix("|")
    }

    static func isTableSeparatorLine(_ line: String) -> Bool {
        let trimmed = line.trimmingCharacters(in: .whitespaces)
        guard trimmed.contains("-") else { return false }
        let patternA = trimmed.range(
            of: #"^\|?(\s*:?-{3,}:?\s*\|)+\s*:?-{3,}:?\s*\|?$"#,
            options: .regularExpression
        ) != nil
        let patternB = trimmed.range(
            of: #"^\|?(\s*:?-{3,}:?\s*\|?)+\s*$"#,
            options: .regularExpression
        ) != nil
        return patternA || patternB
    }

    static func isTableRelatedLine(_ line: String) -> Bool {
        isTableRowLine(line) || isTableSeparatorLine(line)
    }

    static func splitCells(_ line: String) -> [String] {
        var trimmed = line.trimmingCharacters(in: .whitespaces)
        if trimmed.hasPrefix("|") { trimmed = String(trimmed.dropFirst()) }
        if trimmed.hasSuffix("|") { trimmed = String(trimmed.dropLast()) }
        return trimmed.split(separator: "|", omittingEmptySubsequences: false)
            .map { $0.trimmingCharacters(in: .whitespaces) }
    }

    static func parseAlignments(_ separator: String, columnCount: Int) -> [MarkdownTableAlign] {
        let cells = splitCells(separator)
        let parsed: [MarkdownTableAlign] = cells.map { cell in
            let trimmed = cell.trimmingCharacters(in: .whitespaces)
            let leftColon = trimmed.hasPrefix(":")
            let rightColon = trimmed.hasSuffix(":")
            switch (leftColon, rightColon) {
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
        var cursor = from
        while cursor < lines.count && lines[cursor].trimmingCharacters(in: .whitespaces).isEmpty {
            cursor += 1
        }
        return cursor
    }

    /// Reconstruct pipe-table markdown (used when folding into the editor model).
    static func serialize(align: [MarkdownTableAlign], header: [String], rows: [[String]]) -> String {
        func row(_ cells: [String]) -> String {
            "| " + cells.joined(separator: " | ") + " |"
        }
        let separators = align.map { alignment -> String in
            switch alignment {
            case .left: return ":---"
            case .center: return ":---:"
            case .right: return "---:"
            case .default: return "---"
            }
        }
        var lines = [row(header), "| " + separators.joined(separator: " | ") + " |"]
        lines.append(contentsOf: rows.map(row))
        return lines.joined(separator: "\n")
    }
}
