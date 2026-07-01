import Foundation

enum VibeActivityBlockKind: Equatable {
    case heading(level: Int, text: String)
    case paragraph(String)
    case bulletList([String])
    case orderedList([String])
    case reveal(trigger: String, body: String)
    case checkButton(label: String, feedback: String?)
    case freeResponse(prompt: String, placeholder: String?)
    case unsupported(String)
    case divider
}

struct VibeActivityBlock: Identifiable, Equatable {
    let id: Int
    let kind: VibeActivityBlockKind
}

struct VibeActivityDocument: Equatable {
    var blocks: [VibeActivityBlock]
    var requiresWebFallback: Bool
}

enum VibeActivityLogic {
    static func webPath(courseCode: String, itemId: String) -> String {
        "/courses/\(LMSAPI.encodePath(courseCode))/modules/vibe-activity/\(LMSAPI.encodePath(itemId))"
    }

    static func parse(html: String?) -> VibeActivityDocument {
        let trimmed = html?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return VibeActivityDocument(
                blocks: [VibeActivityBlock(id: 0, kind: .paragraph("Empty activity. The instructor has not added content yet."))],
                requiresWebFallback: false
            )
        }

        let body = extractBody(from: trimmed)
        let stripped = stripNoise(from: body)
        if containsHardUnsupportedTags(stripped) && !hasReadableContent(stripped) {
            return VibeActivityDocument(
                blocks: [VibeActivityBlock(id: 0, kind: .unsupported("This activity uses features that need a larger screen."))],
                requiresWebFallback: true
            )
        }

        var blocks: [VibeActivityBlock] = []
        var cursor = 0
        var working = stripped as NSString

        // Native reveal via <details>.
        while let match = firstMatch(in: working as String, pattern: #"<details[^>]*>([\s\S]*?)</details>"#) {
            let prefix = substring(working, range: NSRange(location: 0, length: match.range.location))
            blocks.append(contentsOf: parseSimpleBlocks(from: prefix, startId: cursor))
            cursor = blocks.count

            let inner = captureGroup(match, in: working as String, index: 1) ?? ""
            let summary = firstMatch(in: inner, pattern: #"<summary[^>]*>([\s\S]*?)</summary>"#)
            let trigger = htmlToPlainText(captureGroup(summary, in: inner, index: 1) ?? "Reveal")
            let bodyHtml = inner.replacingOccurrences(
                of: #"<summary[^>]*>[\s\S]*?</summary>"#,
                with: "",
                options: .regularExpression
            )
            let bodyText = htmlToPlainText(bodyHtml)
            if !trigger.isEmpty || !bodyText.isEmpty {
                blocks.append(VibeActivityBlock(id: cursor, kind: .reveal(trigger: trigger, body: bodyText)))
                cursor += 1
            }

            working = substring(
                working,
                range: NSRange(location: match.range.upperBound, length: working.length - match.range.upperBound)
            ) as NSString
        }

        // Button + hidden target reveal pairs.
        while let buttonMatch = firstMatch(
            in: working as String,
            pattern: #"<button[^>]*onclick\s*=\s*["'][^"']*getElementById\(['"]([^'"]+)['"]\)[^"']*["'][^>]*>([\s\S]*?)</button>"#
        ) {
            let prefix = substring(working, range: NSRange(location: 0, length: buttonMatch.range.location))
            blocks.append(contentsOf: parseSimpleBlocks(from: prefix, startId: cursor))
            cursor = blocks.count

            let targetId = captureGroup(buttonMatch, in: working as String, index: 1) ?? ""
            let label = htmlToPlainText(captureGroup(buttonMatch, in: working as String, index: 2) ?? "Reveal")
            let targetPattern = #"<([a-z]+)[^>]*id=["']\#(NSRegularExpression.escapedPattern(for: targetId))["'][^>]*>([\s\S]*?)</\1>"#
            let targetMatch = firstMatch(in: working as String, pattern: targetPattern)
            let hiddenBody = htmlToPlainText(captureGroup(targetMatch, in: working as String, index: 2) ?? "")
            blocks.append(VibeActivityBlock(id: cursor, kind: .reveal(trigger: label, body: hiddenBody)))
            cursor += 1

            var next = NSMutableString(string: working as String)
            next.deleteCharacters(in: buttonMatch.range)
            if let targetMatch {
                next.deleteCharacters(in: targetMatch.range)
            }
            working = next
        }

        blocks.append(contentsOf: parseSimpleBlocks(from: working as String, startId: cursor))

        let requiresFallback = blocks.contains {
            if case .unsupported = $0.kind { return true }
            return false
        }
        if blocks.isEmpty {
            return VibeActivityDocument(
                blocks: [VibeActivityBlock(id: 0, kind: .unsupported("This activity uses features that need a larger screen."))],
                requiresWebFallback: true
            )
        }
        return VibeActivityDocument(blocks: blocks, requiresWebFallback: requiresFallback)
    }

    // MARK: - Block parsing

    private static func appendParsedTag(
        _ tag: String,
        attrs: String,
        inner: String,
        blocks: inout [VibeActivityBlock],
        nextId: Int,
        flushText: (String) -> Void
    ) -> Int {
        var id = nextId
        switch tag {
        case "h1", "h2", "h3", "h4", "h5", "h6":
            let level = Int(tag.dropFirst()) ?? 2
            blocks.append(VibeActivityBlock(id: id, kind: .heading(level: level, text: htmlToPlainText(inner))))
            id += 1
        case "p":
            blocks.append(VibeActivityBlock(id: id, kind: .paragraph(htmlToPlainText(inner))))
            id += 1
        case "ul":
            let items = listItems(from: inner, ordered: false)
            if items.isEmpty {
                flushText(inner)
            } else {
                blocks.append(VibeActivityBlock(id: id, kind: .bulletList(items)))
                id += 1
            }
        case "ol":
            let items = listItems(from: inner, ordered: true)
            if items.isEmpty {
                flushText(inner)
            } else {
                blocks.append(VibeActivityBlock(id: id, kind: .orderedList(items)))
                id += 1
            }
        case "hr", "":
            blocks.append(VibeActivityBlock(id: id, kind: .divider))
            id += 1
        case "textarea":
            let placeholder = attributeValue("placeholder", in: attrs)
            let prompt = htmlToPlainText(inner)
            blocks.append(VibeActivityBlock(id: id, kind: .freeResponse(prompt: prompt, placeholder: placeholder)))
            id += 1
        case "input":
            let type = (attributeValue("type", in: attrs) ?? "text").lowercased()
            if type == "text" || type.isEmpty {
                let placeholder = attributeValue("placeholder", in: attrs)
                blocks.append(VibeActivityBlock(id: id, kind: .freeResponse(prompt: "", placeholder: placeholder)))
                id += 1
            } else {
                blocks.append(VibeActivityBlock(id: id, kind: .unsupported("This input type works best on the web.")))
                id += 1
            }
        case "button":
            let label = htmlToPlainText(inner)
            if !label.isEmpty {
                blocks.append(VibeActivityBlock(id: id, kind: .checkButton(label: label, feedback: nil)))
                id += 1
            }
        case "iframe", "canvas", "video":
            blocks.append(VibeActivityBlock(id: id, kind: .unsupported("Embedded media works best on the web.")))
            id += 1
        case "div":
            let childBlocks = parseSimpleBlocks(from: inner, startId: id)
            if childBlocks.isEmpty {
                let plain = htmlToPlainText(inner)
                if !plain.isEmpty {
                    blocks.append(VibeActivityBlock(id: id, kind: .paragraph(plain)))
                    id += 1
                }
            } else {
                blocks.append(contentsOf: childBlocks)
                id = (blocks.last?.id ?? id) + 1
            }
        default:
            flushText(inner)
        }
        return id
    }

    private static func parseSimpleBlocks(from html: String, startId: Int) -> [VibeActivityBlock] {
        guard !html.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return [] }
        var blocks: [VibeActivityBlock] = []
        var id = startId
        let pattern = #"(?i)<(h[1-6]|p|ul|ol|hr|textarea|input|button|iframe|canvas|video|div)([^>]*)>([\s\S]*?)</\1>|<hr\s*/?>"#
        let regex = makeRegex(pattern)
        let ns = html as NSString
        let matches = regex.matches(in: html, range: NSRange(location: 0, length: ns.length))
        var index = 0

        func flushText(_ text: String) {
            let plain = htmlToPlainText(text)
            guard !plain.isEmpty else { return }
            blocks.append(VibeActivityBlock(id: id, kind: .paragraph(plain)))
            id += 1
        }

        for match in matches {
            let beforeRange = NSRange(location: index, length: match.range.location - index)
            if beforeRange.length > 0 {
                flushText(ns.substring(with: beforeRange))
            }

            let tag = (captureGroup(match, in: html, index: 1) ?? "").lowercased()
            let attrs = captureGroup(match, in: html, index: 2) ?? ""
            let inner = captureGroup(match, in: html, index: 3) ?? ""
            id = appendParsedTag(
                tag,
                attrs: attrs,
                inner: inner,
                blocks: &blocks,
                nextId: id,
                flushText: flushText
            )

            index = match.range.location + match.range.length
        }

        if index < ns.length {
            flushText(ns.substring(from: index))
        }

        if blocks.isEmpty {
            let plain = htmlToPlainText(html)
            if !plain.isEmpty {
                blocks.append(VibeActivityBlock(id: startId, kind: .paragraph(plain)))
            }
        }
        return blocks
    }

    private static func listItems(from html: String, ordered: Bool) -> [String] {
        let regex = makeRegex(#"<li[^>]*>([\s\S]*?)</li>"#)
        let ns = html as NSString
        return regex.matches(in: html, range: NSRange(location: 0, length: ns.length)).compactMap {
            guard let raw = captureGroup($0, in: html, index: 1) else { return nil }
            let text = htmlToPlainText(raw)
            return text.isEmpty ? nil : text
        }
    }

    // MARK: - HTML helpers

    private static func extractBody(from html: String) -> String {
        if let match = firstMatch(in: html, pattern: #"<body[^>]*>([\s\S]*?)</body>"#) {
            return captureGroup(match, in: html, index: 1) ?? html
        }
        return html
    }

    private static func stripNoise(from html: String) -> String {
        var out = html
        for pattern in [#"<script[\s\S]*?</script>"#, #"<style[\s\S]*?</style>"#, #"<link[^>]*>"#] {
            out = out.replacingOccurrences(of: pattern, with: "", options: [.regularExpression, .caseInsensitive])
        }
        return out
    }

    private static func containsHardUnsupportedTags(_ html: String) -> Bool {
        firstMatch(in: html, pattern: #"(?i)<(iframe|canvas|object|embed|applet)[^>]*>"#) != nil
    }

    private static func hasReadableContent(_ html: String) -> Bool {
        !htmlToPlainText(html).isEmpty
    }

    static func htmlToPlainText(_ html: String) -> String {
        var text = html
        text = text.replacingOccurrences(of: #"<br\s*/?>"#, with: "\n", options: .regularExpression)
        text = text.replacingOccurrences(of: #"<[^>]+>"#, with: " ", options: .regularExpression)
        text = decodeEntities(text)
        text = text.replacingOccurrences(of: #"\s+"#, with: " ", options: .regularExpression)
        return text.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private static func decodeEntities(_ value: String) -> String {
        value
            .replacingOccurrences(of: "&nbsp;", with: " ")
            .replacingOccurrences(of: "&amp;", with: "&")
            .replacingOccurrences(of: "&lt;", with: "<")
            .replacingOccurrences(of: "&gt;", with: ">")
            .replacingOccurrences(of: "&quot;", with: "\"")
            .replacingOccurrences(of: "&#39;", with: "'")
    }

    private static func attributeValue(_ name: String, in attrs: String) -> String? {
        let pattern = #"\#(name)\s*=\s*["']([^"']*)["']"#
        guard let match = firstMatch(in: attrs, pattern: pattern) else { return nil }
        return captureGroup(match, in: attrs, index: 1)
    }

    private static func makeRegex(_ pattern: String) -> NSRegularExpression {
        guard let regex = try? NSRegularExpression(pattern: pattern, options: [.caseInsensitive]) else {
            preconditionFailure("Invalid regex: \(pattern)")
        }
        return regex
    }

    private static func firstMatch(in text: String, pattern: String) -> NSTextCheckingResult? {
        let regex = makeRegex(pattern)
        let range = NSRange(text.startIndex..., in: text)
        return regex.firstMatch(in: text, range: range)
    }

    private static func substring(_ value: NSString, range: NSRange) -> String {
        guard range.location != NSNotFound, range.length > 0, range.location + range.length <= value.length else {
            return ""
        }
        return value.substring(with: range)
    }

    private static func captureGroup(_ match: NSTextCheckingResult?, in html: String, index: Int) -> String? {
        guard let match, match.numberOfRanges > index else { return nil }
        let range = match.range(at: index)
        guard range.location != NSNotFound else { return nil }
        return (html as NSString).substring(with: range)
    }
}