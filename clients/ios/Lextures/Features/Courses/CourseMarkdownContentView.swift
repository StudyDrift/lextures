import SwiftUI
import AVKit

/// Read-only markdown reader for course content pages (NotebookMarkdown blocks + math/video).
/// CT.M2: single renderer for all reading surfaces; `compact` caps spacing; `suppressAffordances`
/// hides copy/expand/link escapes during lockdown quizzes (FR-11 / FR-13).
struct CourseMarkdownContentView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let markdown: String
    var captionsEnabled = false
    var compact = false
    var suppressAffordances = false
    @State private var cachedMarkdown = ""
    @State private var cachedBlocks: [NotebookBlock] = []

    var body: some View {
        VStack(alignment: .leading, spacing: MarkdownRenderStyle.blockSpacing(compact: compact)) {
            ForEach(cachedBlocks) { block in
                blockView(block)
            }
        }
        .onAppear { refreshBlocksIfNeeded() }
        .onChange(of: markdown) { _, _ in refreshBlocksIfNeeded() }
    }

    private func refreshBlocksIfNeeded() {
        guard cachedMarkdown != markdown else { return }
        cachedMarkdown = markdown
        cachedBlocks = NotebookMarkdown.parseBlocks(markdown)
    }

    @ViewBuilder
    private func blockView(_ block: NotebookBlock) -> some View {
        switch block.kind {
        case .heading(let level, let text):
            Text(inline(text))
                .font(LexturesTheme.displayFont(MarkdownRenderStyle.headingPointSize(level: level, compact: compact)))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .padding(.top, MarkdownRenderStyle.headingTopPadding(level: level, compact: compact))
        case .paragraph(let text):
            if let videoURL = ModuleContentMedia.videoURL(in: text) {
                if captionsEnabled {
                    CaptionedPlayerView(url: videoURL)
                } else {
                    ContentVideoPlayer(url: videoURL)
                }
            } else {
                mathAwareText(text)
            }
        case .bulletItem(let text, let depth):
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Circle()
                    .fill(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 5, height: 5)
                    .padding(.top, 6)
                mathAwareText(text)
            }
            .padding(.leading, CGFloat(4 + depth * 16))
        case .orderedItem(let number, let text, let depth):
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("\(number).")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                mathAwareText(text)
            }
            .padding(.leading, CGFloat(4 + depth * 16))
        case .taskItem(let checked, let text, let depth):
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Image(systemName: checked ? "checkmark.square.fill" : "square")
                    .foregroundStyle(checked ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                    .accessibilityHidden(true)
                mathAwareText(text)
                    .strikethrough(checked)
            }
            .padding(.leading, CGFloat(4 + depth * 16))
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(checked ? "Completed" : "Not completed"): \(text)")
        case .quote(let text):
            HStack(alignment: .top, spacing: 10) {
                RoundedRectangle(cornerRadius: 2, style: .continuous)
                    .fill(LexturesTheme.amber)
                    .frame(width: 3, height: 24)
                mathAwareText(text)
                    .italic()
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        case .code(let language, let source):
            MarkdownCodeBlockView(
                language: language,
                source: source,
                suppressCopy: suppressAffordances
            )
        case .math(let latex, let display):
            MathLatexView(latex: latex, displayMode: display)
        case .table(let align, let header, let rows):
            MarkdownTableView(
                align: align,
                header: header,
                rows: rows,
                suppressExpand: suppressAffordances
            )
        case .toolFence(_, let toolId, _):
            Label {
                Text(L.format("mobile.markdown.tool.placeholder", toolId))
                    .font(.subheadline)
            } icon: {
                Image(systemName: "puzzlepiece.extension")
            }
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .padding(compact ? 8 : 12)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            .accessibilityLabel(L.format("mobile.markdown.tool.placeholder", toolId))
        case .divider:
            Divider()
        case .task, .drawing:
            EmptyView()
        case .image(let alt, let url):
            AuthorizedNotebookImage(urlString: url, alt: alt)
        }
    }

    @ViewBuilder
    private func mathAwareText(_ text: String) -> some View {
        let segments = ModuleContentMedia.mathSegments(in: text)
        if segments.count == 1, case .text(let only) = segments[0] {
            Text(inline(only))
                .font(.subheadline)
                .lineSpacing(compact ? 1 : 3)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        } else {
            VStack(alignment: .leading, spacing: compact ? 4 : 6) {
                ForEach(Array(segments.enumerated()), id: \.offset) { _, segment in
                    switch segment {
                    case .text(let value):
                        Text(inline(value))
                            .font(.subheadline)
                            .lineSpacing(compact ? 1 : 3)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    case .math(let latex, let display):
                        MathLatexView(latex: latex, displayMode: display)
                    }
                }
            }
        }
    }

    private func inline(_ text: String) -> AttributedString {
        MarkdownRenderStyle.inlineAttributedString(text, suppressLinks: suppressAffordances)
    }
}

enum ModuleContentMedia {
    enum Segment: Equatable {
        case text(String)
        case math(String, display: Bool)
    }

    static func mathSegments(in text: String) -> [Segment] {
        guard text.contains("$") else { return [.text(text)] }
        var segments: [Segment] = []
        var index = text.startIndex
        while index < text.endIndex {
            if text[index] == "$" {
                let next = text.index(after: index)
                let display = next < text.endIndex && text[next] == "$"
                let openLen = display ? 2 : 1
                let openEnd = text.index(index, offsetBy: openLen, limitedBy: text.endIndex) ?? text.endIndex
                let before = String(text[..<index])
                if !before.isEmpty { segments.append(.text(before)) }
                let closePattern = display ? "$$" : "$"
                if let closeRange = text[openEnd...].range(of: closePattern) {
                    let latex = String(text[openEnd ..< closeRange.lowerBound])
                    segments.append(.math(latex, display: display))
                    index = closeRange.upperBound
                } else {
                    segments.append(.text(String(text[index...])))
                    return segments
                }
                continue
            }
            index = text.index(after: index)
        }
        if segments.isEmpty { segments.append(.text(text)) }
        return segments
    }

    static func videoURL(in text: String) -> URL? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let url = URL(string: trimmed), url.scheme?.hasPrefix("http") == true else { return nil }
        let host = url.host?.lowercased() ?? ""
        if host.contains("youtube.com") || host.contains("youtu.be") || host.contains("vimeo.com") {
            return url
        }
        let path = url.path.lowercased()
        if [".mp4", ".mov", ".m3u8", ".webm"].contains(where: { path.hasSuffix($0) }) {
            return url
        }
        return nil
    }
}

/// Typeset math with a safe monospace fallback (CT.M1 FR-10 / open question #1).
struct MathLatexView: View {
    let latex: String
    let displayMode: Bool

    var body: some View {
        Text(prettyLatex)
            .font(.system(displayMode ? .body : .subheadline, design: .monospaced))
            .foregroundStyle(.primary)
            .padding(.vertical, displayMode ? 4 : 0)
            .frame(maxWidth: displayMode ? .infinity : nil, alignment: displayMode ? .center : .leading)
            .accessibilityLabel(accessibilityText)
    }

    /// Light prettification so common fractions read better without a native math library.
    private var prettyLatex: String {
        let trimmed = latex.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return latex }
        // Keep source visible (never blank) — FR-10 fallback path.
        return trimmed
            .replacingOccurrences(of: "\\frac{", with: "(")
            .replacingOccurrences(of: "}{", with: ")/(")
            .replacingOccurrences(of: "\\cdot", with: "·")
            .replacingOccurrences(of: "\\times", with: "×")
    }

    private var accessibilityText: String {
        if displayMode {
            return L.format("mobile.markdown.math.display", latex)
        }
        return L.format("mobile.markdown.math.inline", latex)
    }
}

struct ContentVideoPlayer: View {
    let url: URL

    var body: some View {
        VideoPlayer(player: AVPlayer(url: url))
            .frame(maxWidth: .infinity)
            .aspectRatio(16 / 9, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .accessibilityLabel("Embedded video")
    }
}
