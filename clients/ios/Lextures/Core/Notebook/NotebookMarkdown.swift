import Foundation

// swiftlint:disable type_body_length

/// A task parsed from a ```task fenced block (parity with web `notebook-task-markdown`).
struct ParsedNotebookTask: Identifiable, Equatable {
    let id: String
    var text: String
    var checked: Bool
    var dueAt: String?
}

/// One slash-command / toolbar insert action (parity with web `markdown-body-slash`).
struct NotebookSlashCommand: Identifiable, Equatable {
    let id: String
    let label: String
    let detail: String
    let icon: String
    let keywords: [String]
}

/// Renderable markdown block for the notebook / course reading view (CT.M1).
enum NotebookBlockKind: Equatable {
    case heading(level: Int, text: String)
    case paragraph(String)
    case bulletItem(text: String, depth: Int)
    case orderedItem(number: String, text: String, depth: Int)
    /// GFM task-list item (`- [ ]` / `- [x]`), read-only in reader contexts.
    case taskItem(checked: Bool, text: String, depth: Int)
    case quote(String)
    case code(language: String?, source: String)
    case math(latex: String, display: Bool)
    case table(align: [MarkdownTableAlign], header: [String], rows: [[String]])
    /// ` ```lex-tool ` pointer; raw JSON must never be shown to learners (CT.M3 hosts it).
    case toolFence(instanceId: String, toolId: String, version: Int)
    case divider
    case task(ParsedNotebookTask)
    case image(alt: String, url: String)
    /// `index` is the drawing's ordinal among all drawings on the page (for write-back).
    case drawing(index: Int, elementsJson: String)
}

struct NotebookBlock: Identifiable, Equatable {
    let id: Int
    let kind: NotebookBlockKind
}

/// One editable block in the WYSIWYG notebook editor (parity with the web block editor:
/// blocks stay rendered while editing; markdown is only the storage format).
struct NotebookEditBlock: Identifiable, Equatable {
    enum Kind: Equatable {
        case paragraph
        case heading(Int)
        case bullet
        case ordered
        case quote
        case code
        case divider
        case task(taskId: String, checked: Bool, dueAt: String?)
        case image(alt: String, url: String)
        case drawing(elementsJson: String)

        var isOrdered: Bool {
            if case .ordered = self { return true }
            return false
        }

        /// Consecutive items of the same list/quote kind join with one newline, not a blank line.
        func sameListRun(as other: Kind?) -> Bool {
            switch (self, other) {
            case (.bullet, .bullet), (.ordered, .ordered), (.quote, .quote): return true
            default: return false
            }
        }
    }

    let id: UUID
    var kind: Kind
    var text: String

    init(kind: Kind, text: String = "") {
        id = UUID()
        self.kind = kind
        self.text = text
    }

    /// Whether the block carries user-editable text (false for divider / image / drawing).
    var isTextual: Bool {
        switch kind {
        case .divider, .image, .drawing: return false
        default: return true
        }
    }
}

enum NotebookMarkdown {
    // MARK: - Task blocks (```task + JSON meta line)

    private static let taskBlockRegex = makeRegex("```task[ \\t]*\\n([\\s\\S]*?)```")

    private static func makeRegex(_ pattern: String) -> NSRegularExpression {
        guard let regex = try? NSRegularExpression(pattern: pattern) else {
            preconditionFailure("Invalid regex: \(pattern)")
        }
        return regex
    }

    static func newTaskId() -> String {
        UUID().uuidString.lowercased()
    }

    static func taskMetaLine(id: String, checked: Bool, dueAt: String?) -> String {
        let due = dueAt.map { "\"\(jsonEscape($0))\"" } ?? "null"
        return "{\"id\":\"\(jsonEscape(id))\",\"checked\":\(checked),\"dueAt\":\(due)}"
    }

    private static func jsonEscape(_ value: String) -> String {
        value
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
            .replacingOccurrences(of: "\n", with: "\\n")
    }

    private static func parseTaskMeta(line: String) -> (id: String, checked: Bool, dueAt: String?)? {
        guard
            let data = line.data(using: .utf8),
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
            let id = json["id"] as? String, !id.isEmpty
        else { return nil }
        return (id, json["checked"] as? Bool == true, json["dueAt"] as? String)
    }

    private static func parseTaskInner(_ inner: String) -> ParsedNotebookTask? {
        var lines = inner.components(separatedBy: "\n")
        guard let meta = parseTaskMeta(line: lines.first ?? "") else { return nil }
        lines.removeFirst()
        let text = lines.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
        return ParsedNotebookTask(id: meta.id, text: text, checked: meta.checked, dueAt: meta.dueAt)
    }

    static func parseTasks(in contentMd: String) -> [ParsedNotebookTask] {
        let ns = contentMd as NSString
        return taskBlockRegex.matches(in: contentMd, range: NSRange(location: 0, length: ns.length))
            .compactMap { match in
                parseTaskInner(ns.substring(with: match.range(at: 1)))
            }
    }

    /// Rewrite the matching task block, transforming its meta (`checked` / `dueAt`); body text unchanged.
    private static func rewriteTask(
        in contentMd: String,
        taskId: String,
        transform: (ParsedNotebookTask) -> (checked: Bool, dueAt: String?)
    ) -> String {
        let ns = contentMd as NSString
        var result = ""
        var cursor = 0
        for match in taskBlockRegex.matches(in: contentMd, range: NSRange(location: 0, length: ns.length)) {
            result += ns.substring(with: NSRange(location: cursor, length: match.range.location - cursor))
            let inner = ns.substring(with: match.range(at: 1))
            if let task = parseTaskInner(inner), task.id == taskId {
                let next = transform(task)
                var bodyLines = inner.components(separatedBy: "\n")
                bodyLines.removeFirst()
                let body = bodyLines.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
                let meta = taskMetaLine(id: task.id, checked: next.checked, dueAt: next.dueAt)
                result += "```task\n\(meta)\n\(body)\n```"
            } else {
                result += ns.substring(with: match.range)
            }
            cursor = match.range.location + match.range.length
        }
        result += ns.substring(from: cursor)
        return result
    }

    static func setTaskChecked(in contentMd: String, taskId: String, checked: Bool) -> String {
        rewriteTask(in: contentMd, taskId: taskId) { (checked, $0.dueAt) }
    }

    static func setTaskDueAt(in contentMd: String, taskId: String, dueAt: String?) -> String {
        rewriteTask(in: contentMd, taskId: taskId) { ($0.checked, dueAt) }
    }

    // MARK: - Block parsing (reading view)

    /// Kind labels used by the shared golden-fixture corpus (`clients/mobile/fixtures/markdown`).
    static func fixtureKindName(_ kind: NotebookBlockKind) -> String {
        switch kind {
        case .heading: return "heading"
        case .paragraph: return "paragraph"
        case .bulletItem: return "bullet"
        case .orderedItem: return "ordered"
        case .taskItem: return "taskItem"
        case .quote: return "quote"
        case .code: return "code"
        case .math: return "math"
        case .table: return "table"
        case .toolFence: return "toolFence"
        case .divider: return "divider"
        case .task: return "task"
        case .image: return "image"
        case .drawing: return "drawing"
        }
    }

    static func parseBlocks(_ contentMd: String) -> [NotebookBlock] {
        var kinds: [NotebookBlockKind] = []
        var paragraph: [String] = []
        var quote: [String] = []

        func flushParagraph() {
            guard !paragraph.isEmpty else { return }
            let text = paragraph.joined(separator: "\n")
            paragraph = []
            if let math = parseStandaloneMath(text) {
                kinds.append(math)
            } else {
                kinds.append(.paragraph(text))
            }
        }
        func flushQuote() {
            if !quote.isEmpty {
                kinds.append(.quote(quote.joined(separator: "\n")))
                quote = []
            }
        }
        func flushAll() {
            flushParagraph()
            flushQuote()
        }

        let healed = MarkdownTableLogic.normalizeMarkdownTables(
            contentMd.replacingOccurrences(of: "\r\n", with: "\n")
        )
        let lines = healed.components(separatedBy: "\n")
        var lineIndex = 0
        var drawingIndex = 0
        while lineIndex < lines.count {
            let line = lines[lineIndex]
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            let indent = listDepth(line)

            if let table = MarkdownTableLogic.parseTable(in: lines, start: lineIndex) {
                flushAll()
                kinds.append(.table(align: table.align, header: table.header, rows: table.rows))
                lineIndex = table.endIndex
                continue
            }
            if let fenceResult = consumeFenceBlock(
                lines: lines,
                lineIndex: &lineIndex,
                trimmed: trimmed,
                drawingIndex: &drawingIndex
            ) {
                flushAll()
                if let fenceKind = fenceResult {
                    kinds.append(fenceKind)
                }
                continue
            }
            if let heading = parseHeading(trimmed) {
                flushAll()
                kinds.append(heading)
            } else if trimmed == "---" || trimmed == "***" || trimmed == "___" {
                flushAll()
                kinds.append(.divider)
            } else if let image = parseImage(trimmed) {
                flushAll()
                kinds.append(image)
            } else if let taskItem = parseTaskListItem(trimmed) {
                flushAll()
                kinds.append(.taskItem(checked: taskItem.checked, text: taskItem.text, depth: min(indent, 3)))
            } else if trimmed.hasPrefix("- ") || trimmed.hasPrefix("* ") {
                flushAll()
                kinds.append(.bulletItem(text: String(trimmed.dropFirst(2)), depth: min(indent, 3)))
            } else if let ordered = parseOrderedItem(trimmed, depth: min(indent, 3)) {
                flushAll()
                kinds.append(ordered)
            } else if trimmed.hasPrefix(">") {
                flushParagraph()
                quote.append(trimmed.dropFirst().trimmingCharacters(in: .whitespaces))
            } else if trimmed.isEmpty {
                flushAll()
            } else {
                flushQuote()
                paragraph.append(trimmed)
            }
            lineIndex += 1
        }
        flushAll()
        return kinds.enumerated().map { NotebookBlock(id: $0.offset, kind: $0.element) }
    }

    /// Consume a fenced block (drawing / task / code / lex-tool). Advances `lineIndex` past the closer.
    /// Returns `nil` when the line is not a fence; `.some(nil)` when the fence was skipped (e.g. bad task).
    private static func consumeFenceBlock(
        lines: [String],
        lineIndex: inout Int,
        trimmed: String,
        drawingIndex: inout Int
    ) -> NotebookBlockKind?? {
        guard trimmed.hasPrefix("```") else { return nil }
        if trimmed.hasPrefix("```drawing") {
            let source = readFenceInner(lines: lines, lineIndex: &lineIndex)
            let kind: NotebookBlockKind = .drawing(
                index: drawingIndex,
                elementsJson: source.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            drawingIndex += 1
            return .some(kind)
        }
        if trimmed == "```task" || trimmed.hasPrefix("```task") {
            let source = readFenceInner(lines: lines, lineIndex: &lineIndex)
            return .some(parseTaskInner(source).map { .task($0) })
        }
        let language = fenceLanguage(trimmed)
        let source = readFenceInner(lines: lines, lineIndex: &lineIndex)
        if language == "lex-tool", let tool = parseLexToolFence(source) {
            return .some(.toolFence(instanceId: tool.instanceId, toolId: tool.toolId, version: tool.version))
        }
        return .some(.code(language: language, source: source))
    }

    private static func readFenceInner(lines: [String], lineIndex: inout Int) -> String {
        var inner: [String] = []
        lineIndex += 1
        while lineIndex < lines.count, !lines[lineIndex].trimmingCharacters(in: .whitespaces).hasPrefix("```") {
            inner.append(lines[lineIndex])
            lineIndex += 1
        }
        lineIndex += 1
        return inner.joined(separator: "\n")
    }

    private static func fenceLanguage(_ opener: String) -> String? {
        let rest = opener.dropFirst(3).trimmingCharacters(in: .whitespaces)
        if rest.isEmpty { return nil }
        let language = rest.split(whereSeparator: { $0.isWhitespace }).first.map(String.init)
        return language?.isEmpty == true ? nil : language
    }

    private struct LexToolFencePayload {
        let instanceId: String
        let toolId: String
        let version: Int
    }

    private static func parseLexToolFence(_ source: String) -> LexToolFencePayload? {
        guard
            let data = source.trimmingCharacters(in: .whitespacesAndNewlines).data(using: .utf8),
            let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
        else { return nil }
        let instanceId = (json["instanceId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let toolId = (json["toolId"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let version: Int? = {
            if let intVersion = json["v"] as? Int { return intVersion }
            if let stringVersion = json["v"] as? String { return Int(stringVersion) }
            return nil
        }()
        guard !instanceId.isEmpty, !toolId.isEmpty, version == 1 else { return nil }
        return LexToolFencePayload(instanceId: instanceId, toolId: toolId, version: 1)
    }

    private static func parseStandaloneMath(_ text: String) -> NotebookBlockKind? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.hasPrefix("$$"), trimmed.hasSuffix("$$"), trimmed.count >= 4 else { return nil }
        let inner = String(trimmed.dropFirst(2).dropLast(2)).trimmingCharacters(in: .whitespacesAndNewlines)
        guard !inner.isEmpty, !inner.contains("$$") else { return nil }
        return .math(latex: inner, display: true)
    }

    private static func listDepth(_ line: String) -> Int {
        var spaces = 0
        for ch in line {
            if ch == " " { spaces += 1 }
            else if ch == "\t" { spaces += 2 }
            else { break }
        }
        return min(spaces / 2, 3)
    }

    private static let taskListRegex = makeRegex("^[-*]\\s+\\[([ xX])\\]\\s+(.*)$")

    private static func parseTaskListItem(_ line: String) -> (checked: Bool, text: String)? {
        let ns = line as NSString
        guard let match = taskListRegex.firstMatch(in: line, range: NSRange(location: 0, length: ns.length)) else {
            return nil
        }
        let mark = ns.substring(with: match.range(at: 1)).lowercased()
        return (mark == "x", ns.substring(with: match.range(at: 2)))
    }

    private static func parseHeading(_ line: String) -> NotebookBlockKind? {
        guard line.hasPrefix("#") else { return nil }
        let hashes = line.prefix(while: { $0 == "#" })
        guard hashes.count <= 6 else { return nil }
        let rest = line.dropFirst(hashes.count)
        guard rest.hasPrefix(" ") else { return nil }
        return .heading(level: hashes.count, text: rest.trimmingCharacters(in: .whitespaces))
    }

    private static let orderedItemRegex = makeRegex("^(\\d+)[.)] (.*)$")

    private static func parseOrderedItem(_ line: String, depth: Int = 0) -> NotebookBlockKind? {
        let ns = line as NSString
        guard let match = orderedItemRegex.firstMatch(in: line, range: NSRange(location: 0, length: ns.length)) else {
            return nil
        }
        return .orderedItem(
            number: ns.substring(with: match.range(at: 1)),
            text: ns.substring(with: match.range(at: 2)),
            depth: depth
        )
    }

    private static let imageRegex = makeRegex("^!\\[([^\\]]*)\\]\\(([^)]+)\\)$")

    private static func parseImage(_ line: String) -> NotebookBlockKind? {
        let ns = line as NSString
        guard let match = imageRegex.firstMatch(in: line, range: NSRange(location: 0, length: ns.length)) else {
            return nil
        }
        return .image(alt: ns.substring(with: match.range(at: 1)), url: ns.substring(with: match.range(at: 2)))
    }

    // MARK: - Edit blocks (WYSIWYG editor, parity with web block editor)

    static func editBlocks(from contentMd: String) -> [NotebookEditBlock] {
        var out: [NotebookEditBlock] = []
        for block in parseBlocks(contentMd) {
            switch block.kind {
            case .heading(let level, let text):
                out.append(NotebookEditBlock(kind: .heading(level), text: text))
            case .paragraph(let text):
                for line in text.components(separatedBy: "\n") {
                    out.append(NotebookEditBlock(kind: .paragraph, text: line))
                }
            case .bulletItem(let text, _):
                out.append(NotebookEditBlock(kind: .bullet, text: text))
            case .orderedItem(_, let text, _):
                out.append(NotebookEditBlock(kind: .ordered, text: text))
            case .taskItem(let checked, let text, _):
                let mark = checked ? "x" : " "
                out.append(NotebookEditBlock(kind: .bullet, text: "[\(mark)] \(text)"))
            case .quote(let text):
                for line in text.components(separatedBy: "\n") {
                    out.append(NotebookEditBlock(kind: .quote, text: line))
                }
            case .code(_, let source):
                out.append(NotebookEditBlock(kind: .code, text: source))
            case .math(let latex, let display):
                out.append(NotebookEditBlock(
                    kind: .paragraph,
                    text: display ? "$$\(latex)$$" : "$\(latex)$"
                ))
            case .table(let align, let header, let rows):
                for line in MarkdownTableLogic.serialize(align: align, header: header, rows: rows)
                    .components(separatedBy: "\n")
                {
                    out.append(NotebookEditBlock(kind: .paragraph, text: line))
                }
            case .toolFence(let instanceId, let toolId, let version):
                // Preserve pointer as a code fence body so editors do not drop the instance.
                out.append(NotebookEditBlock(
                    kind: .code,
                    text: "{\"instanceId\":\"\(instanceId)\",\"toolId\":\"\(toolId)\",\"v\":\(version)}"
                ))
            case .divider:
                out.append(NotebookEditBlock(kind: .divider))
            case .task(let task):
                out.append(NotebookEditBlock(
                    kind: .task(taskId: task.id, checked: task.checked, dueAt: task.dueAt),
                    text: task.text
                ))
            case .image(let alt, let url):
                out.append(NotebookEditBlock(kind: .image(alt: alt, url: url)))
            case .drawing(_, let elementsJson):
                out.append(NotebookEditBlock(kind: .drawing(elementsJson: elementsJson)))
            }
        }
        if out.isEmpty {
            out.append(NotebookEditBlock(kind: .paragraph))
        }
        return out
    }

    static func markdown(from blocks: [NotebookEditBlock]) -> String {
        var out = ""
        var previous: NotebookEditBlock.Kind?
        var orderedRun = 0

        for block in blocks {
            let chunk: String
            switch block.kind {
            case .paragraph:
                if block.text.trimmingCharacters(in: .whitespaces).isEmpty { continue }
                chunk = block.text
            case .heading(let level):
                chunk = String(repeating: "#", count: max(1, min(level, 6))) + " " + block.text
            case .bullet:
                chunk = "- \(block.text)"
            case .ordered:
                orderedRun = previous?.isOrdered == true ? orderedRun + 1 : 1
                chunk = "\(orderedRun). \(block.text)"
            case .quote:
                chunk = "> \(block.text)"
            case .code:
                chunk = "```\n\(block.text)\n```"
            case .divider:
                chunk = "---"
            case .task(let taskId, let checked, let dueAt):
                chunk = "```task\n\(taskMetaLine(id: taskId, checked: checked, dueAt: dueAt))\n\(block.text)\n```"
            case .image(let alt, let url):
                chunk = "![\(alt)](\(url))"
            case .drawing(let elementsJson):
                chunk = "```drawing\n\(elementsJson)\n```"
            }
            if out.isEmpty {
                out = chunk
            } else if block.kind.sameListRun(as: previous) {
                out += "\n" + chunk
            } else {
                out += "\n\n" + chunk
            }
            previous = block.kind
        }
        return out
    }

    /// Replace the elements JSON of the page's Nth drawing fence (0-based, document order).
    static func replaceDrawing(in contentMd: String, index: Int, elementsJson: String) -> String {
        var out: [String] = []
        var current = -1
        let lines = contentMd.replacingOccurrences(of: "\r\n", with: "\n").components(separatedBy: "\n")
        var lineIndex = 0
        while lineIndex < lines.count {
            let trimmed = lines[lineIndex].trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("```drawing") {
                current += 1
                var inner: [String] = []
                lineIndex += 1
                while lineIndex < lines.count, lines[lineIndex].trimmingCharacters(in: .whitespaces) != "```" {
                    inner.append(lines[lineIndex])
                    lineIndex += 1
                }
                lineIndex += 1
                let body = current == index
                    ? elementsJson
                    : inner.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
                out.append("```drawing\n\(body)\n```")
                continue
            }
            out.append(lines[lineIndex])
            lineIndex += 1
        }
        return out.joined(separator: "\n")
    }

    // MARK: - Preview text (notebook cards)

    /// Human-readable preview: strips fences and task meta lines so cards never show raw JSON.
    static func previewText(_ contentMd: String) -> String {
        var out: [String] = []
        for block in parseBlocks(contentMd) {
            switch block.kind {
            case .heading(_, let text), .paragraph(let text), .quote(let text):
                out.append(text)
            case .bulletItem(let text, _), .taskItem(_, let text, _), .orderedItem(_, let text, _):
                out.append(text)
            case .task(let task):
                out.append(task.text)
            case .code(_, let source):
                out.append(source)
            case .math(let latex, _):
                out.append(latex)
            case .table(_, let header, let rows):
                out.append(header.joined(separator: " "))
                for row in rows.prefix(3) {
                    out.append(row.joined(separator: " "))
                }
            case .toolFence(_, let toolId, _):
                out.append(toolId)
            case .image(let alt, _):
                out.append(alt)
            case .drawing:
                out.append("Drawing")
            case .divider:
                continue
            }
        }
        return out.joined(separator: " · ").trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
