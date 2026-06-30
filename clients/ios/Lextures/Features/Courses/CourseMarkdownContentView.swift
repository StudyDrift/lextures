import SwiftUI
import AVKit

/// Read-only markdown reader for course content pages (NotebookMarkdown blocks + math/video).
struct CourseMarkdownContentView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let markdown: String

    private var blocks: [NotebookBlock] {
        NotebookMarkdown.parseBlocks(markdown)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            ForEach(blocks) { block in
                blockView(block)
            }
        }
    }

    @ViewBuilder
    private func blockView(_ block: NotebookBlock) -> some View {
        switch block.kind {
        case .heading(let level, let text):
            Text(inline(text))
                .font(LexturesTheme.displayFont(level == 1 ? 24 : level == 2 ? 19 : 16))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .padding(.top, level == 1 ? 6 : 2)
        case .paragraph(let text):
            if let videoURL = ModuleContentMedia.videoURL(in: text) {
                ContentVideoPlayer(url: videoURL)
            } else {
                mathAwareText(text)
            }
        case .bulletItem(let text):
            HStack(alignment: .firstTextBaseline, spacing: 10) {
                Circle()
                    .fill(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 5, height: 5)
                    .padding(.top, 6)
                mathAwareText(text)
            }
            .padding(.leading, 4)
        case .orderedItem(let number, let text):
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("\(number).")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                mathAwareText(text)
            }
            .padding(.leading, 4)
        case .quote(let text):
            HStack(alignment: .top, spacing: 10) {
                RoundedRectangle(cornerRadius: 2, style: .continuous)
                    .fill(LexturesTheme.amber)
                    .frame(width: 3, height: 24)
                mathAwareText(text)
                    .italic()
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        case .code(let text):
            Text(text)
                .font(.system(.caption, design: .monospaced))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(LexturesTheme.sceneBackground(for: colorScheme))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
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
                .lineSpacing(3)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        } else {
            VStack(alignment: .leading, spacing: 6) {
                ForEach(Array(segments.enumerated()), id: \.offset) { _, segment in
                    switch segment {
                    case .text(let value):
                        Text(inline(value))
                            .font(.subheadline)
                            .lineSpacing(3)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    case .math(let latex, let display):
                        MathLatexView(latex: latex, displayMode: display)
                    }
                }
            }
        }
    }

    private func inline(_ text: String) -> AttributedString {
        (try? AttributedString(
            markdown: text,
            options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        )) ?? AttributedString(text)
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
        var inDisplay = false
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
                    index = display
                        ? text.index(closeRange.upperBound, offsetBy: 1, limitedBy: text.endIndex) ?? text.endIndex
                        : closeRange.upperBound
                } else {
                    segments.append(.text(String(text[index...])))
                    return segments
                }
                inDisplay = display
                _ = inDisplay
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

struct MathLatexView: View {
    let latex: String
    let displayMode: Bool

    var body: some View {
        Text(latex)
            .font(.system(displayMode ? .body : .subheadline, design: .serif))
            .foregroundStyle(.primary)
            .padding(.vertical, displayMode ? 4 : 0)
            .accessibilityLabel(displayMode ? "Display equation: \(latex)" : "Inline equation: \(latex)")
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
