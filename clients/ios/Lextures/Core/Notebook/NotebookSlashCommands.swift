import Foundation

/// Slash-command palette for the notebook block editor (parity with web `markdown-body-slash`).
enum NotebookSlashCommands {
    static let all: [NotebookSlashCommand] = [
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

    static func filter(query: String) -> [NotebookSlashCommand] {
        let normalizedQuery = query.trimmingCharacters(in: .whitespaces).lowercased()
        guard !normalizedQuery.isEmpty else { return all }
        return all.filter { cmd in
            if cmd.id.lowercased() == normalizedQuery
                || cmd.id.lowercased().hasPrefix(normalizedQuery)
                || normalizedQuery.hasPrefix(cmd.id.lowercased()) {
                return true
            }
            if cmd.label.lowercased().contains(normalizedQuery)
                || cmd.detail.lowercased().contains(normalizedQuery) {
                return true
            }
            return cmd.keywords.contains { kw in
                if kw == normalizedQuery { return true }
                guard kw.count >= 2, normalizedQuery.count >= 2 else { return false }
                return kw.hasPrefix(normalizedQuery) || normalizedQuery.hasPrefix(kw)
            }
        }
    }
}
