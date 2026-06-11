import Foundation

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

/// Renderable markdown block for the notebook reading view.
enum NotebookBlockKind: Equatable {
    case heading(level: Int, text: String)
    case paragraph(String)
    case bulletItem(String)
    case orderedItem(number: String, text: String)
    case quote(String)
    case code(String)
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

    private static let taskBlockRegex = try! NSRegularExpression(pattern: "```task[ \\t]*\\n([\\s\\S]*?)```")

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

    static func parseBlocks(_ contentMd: String) -> [NotebookBlock] {
        var kinds: [NotebookBlockKind] = []
        var paragraph: [String] = []
        var quote: [String] = []

        func flushParagraph() {
            if !paragraph.isEmpty {
                kinds.append(.paragraph(paragraph.joined(separator: "\n")))
                paragraph = []
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

        let lines = contentMd.replacingOccurrences(of: "\r\n", with: "\n").components(separatedBy: "\n")
        var i = 0
        var drawingIndex = 0
        while i < lines.count {
            let line = lines[i]
            let trimmed = line.trimmingCharacters(in: .whitespaces)

            if trimmed.hasPrefix("```drawing") {
                flushAll()
                var inner: [String] = []
                i += 1
                while i < lines.count, lines[i].trimmingCharacters(in: .whitespaces) != "```" {
                    inner.append(lines[i])
                    i += 1
                }
                kinds.append(.drawing(index: drawingIndex, elementsJson: inner.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)))
                drawingIndex += 1
                i += 1
                continue
            }
            if trimmed == "```task" || trimmed.hasPrefix("```task") {
                flushAll()
                var inner: [String] = []
                i += 1
                while i < lines.count, lines[i].trimmingCharacters(in: .whitespaces) != "```" {
                    inner.append(lines[i])
                    i += 1
                }
                if let task = parseTaskInner(inner.joined(separator: "\n")) {
                    kinds.append(.task(task))
                }
                i += 1
                continue
            }
            if trimmed.hasPrefix("```") {
                flushAll()
                var inner: [String] = []
                i += 1
                while i < lines.count, !lines[i].trimmingCharacters(in: .whitespaces).hasPrefix("```") {
                    inner.append(lines[i])
                    i += 1
                }
                kinds.append(.code(inner.joined(separator: "\n")))
                i += 1
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
            } else if trimmed.hasPrefix("- ") || trimmed.hasPrefix("* ") {
                flushAll()
                kinds.append(.bulletItem(String(trimmed.dropFirst(2))))
            } else if let ordered = parseOrderedItem(trimmed) {
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
            i += 1
        }
        flushAll()
        return kinds.enumerated().map { NotebookBlock(id: $0.offset, kind: $0.element) }
    }

    private static func parseHeading(_ line: String) -> NotebookBlockKind? {
        guard line.hasPrefix("#") else { return nil }
        let hashes = line.prefix(while: { $0 == "#" })
        guard hashes.count <= 6 else { return nil }
        let rest = line.dropFirst(hashes.count)
        guard rest.hasPrefix(" ") else { return nil }
        return .heading(level: hashes.count, text: rest.trimmingCharacters(in: .whitespaces))
    }

    private static let orderedItemRegex = try! NSRegularExpression(pattern: "^(\\d+)[.)] (.*)$")

    private static func parseOrderedItem(_ line: String) -> NotebookBlockKind? {
        let ns = line as NSString
        guard let match = orderedItemRegex.firstMatch(in: line, range: NSRange(location: 0, length: ns.length)) else {
            return nil
        }
        return .orderedItem(number: ns.substring(with: match.range(at: 1)), text: ns.substring(with: match.range(at: 2)))
    }

    private static let imageRegex = try! NSRegularExpression(pattern: "^!\\[([^\\]]*)\\]\\(([^)]+)\\)$")

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
            case .bulletItem(let text):
                out.append(NotebookEditBlock(kind: .bullet, text: text))
            case .orderedItem(_, let text):
                out.append(NotebookEditBlock(kind: .ordered, text: text))
            case .quote(let text):
                for line in text.components(separatedBy: "\n") {
                    out.append(NotebookEditBlock(kind: .quote, text: line))
                }
            case .code(let text):
                out.append(NotebookEditBlock(kind: .code, text: text))
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
        var i = 0
        while i < lines.count {
            let trimmed = lines[i].trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("```drawing") {
                current += 1
                var inner: [String] = []
                i += 1
                while i < lines.count, lines[i].trimmingCharacters(in: .whitespaces) != "```" {
                    inner.append(lines[i])
                    i += 1
                }
                i += 1
                let body = current == index
                    ? elementsJson
                    : inner.joined(separator: "\n").trimmingCharacters(in: .whitespacesAndNewlines)
                out.append("```drawing\n\(body)\n```")
                continue
            }
            out.append(lines[i])
            i += 1
        }
        return out.joined(separator: "\n")
    }

    // MARK: - Slash commands

    static let slashCommands: [NotebookSlashCommand] = [
        NotebookSlashCommand(
            id: "heading1", label: "Heading 1", detail: "Large section heading",
            icon: "textformat.size.larger", keywords: ["h1", "title", "heading"]
        ),
        NotebookSlashCommand(
            id: "heading2", label: "Heading 2", detail: "Medium section heading",
            icon: "textformat.size", keywords: ["h2", "heading"]
        ),
        NotebookSlashCommand(
            id: "heading3", label: "Heading 3", detail: "Small section heading",
            icon: "textformat.size.smaller", keywords: ["h3", "heading"]
        ),
        NotebookSlashCommand(
            id: "task", label: "Task", detail: "Checkbox task with optional due date",
            icon: "checkmark.square", keywords: ["task", "todo", "checkbox", "checklist"]
        ),
        NotebookSlashCommand(
            id: "drawing", label: "Drawing", detail: "Insert a whiteboard to draw on",
            icon: "scribble.variable", keywords: ["drawing", "whiteboard", "sketch", "draw", "canvas"]
        ),
        NotebookSlashCommand(
            id: "bulletList", label: "Bullet list", detail: "Unordered list",
            icon: "list.bullet", keywords: ["ul", "list", "bullets"]
        ),
        NotebookSlashCommand(
            id: "orderedList", label: "Numbered list", detail: "Ordered list",
            icon: "list.number", keywords: ["ol", "list", "numbers"]
        ),
        NotebookSlashCommand(
            id: "blockquote", label: "Quote", detail: "Indented quotation",
            icon: "text.quote", keywords: ["quote", "blockquote"]
        ),
        NotebookSlashCommand(
            id: "codeBlock", label: "Code", detail: "Code block",
            icon: "curlybraces", keywords: ["code", "pre", "snippet"]
        ),
        NotebookSlashCommand(
            id: "horizontalRule", label: "Divider", detail: "Horizontal line",
            icon: "minus", keywords: ["hr", "divider", "line", "rule"]
        ),
    ]

    static func filterCommands(query: String) -> [NotebookSlashCommand] {
        let q = query.trimmingCharacters(in: .whitespaces).lowercased()
        guard !q.isEmpty else { return slashCommands }
        return slashCommands.filter { cmd in
            if cmd.id.lowercased() == q || cmd.id.lowercased().hasPrefix(q) || q.hasPrefix(cmd.id.lowercased()) {
                return true
            }
            if cmd.label.lowercased().contains(q) || cmd.detail.lowercased().contains(q) { return true }
            return cmd.keywords.contains { kw in
                if kw == q { return true }
                guard kw.count >= 2, q.count >= 2 else { return false }
                return kw.hasPrefix(q) || q.hasPrefix(kw)
            }
        }
    }

    // MARK: - Preview text (notebook cards)

    /// Human-readable preview: strips fences and task meta lines so cards never show raw JSON.
    static func previewText(_ contentMd: String) -> String {
        var out: [String] = []
        for block in parseBlocks(contentMd) {
            switch block.kind {
            case .heading(_, let text), .paragraph(let text), .bulletItem(let text), .quote(let text):
                out.append(text)
            case .orderedItem(_, let text):
                out.append(text)
            case .task(let task):
                out.append(task.text)
            case .code(let text):
                out.append(text)
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
