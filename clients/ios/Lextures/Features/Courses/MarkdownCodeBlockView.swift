import SwiftUI
import UniformTypeIdentifiers

/// Fenced code block: language label, monospace, horizontal scroll, copy (CT.M1 FR-7).
struct MarkdownCodeBlockView: View {
    @Environment(\.colorScheme) private var colorScheme
    let language: String?
    let source: String
    @State private var copied = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text(languageLabel)
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Spacer()
                Button {
                    UIPasteboard.general.string = source
                    copied = true
                    DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) { copied = false }
                } label: {
                    Text(copied ? L.text("mobile.markdown.code.copied") : L.text("mobile.markdown.code.copy"))
                        .font(.caption2.weight(.semibold))
                }
                .buttonStyle(.plain)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .accessibilityLabel(L.text("mobile.markdown.code.copy"))
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)

            ScrollView(.horizontal, showsIndicators: true) {
                Text(source.isEmpty ? " " : source)
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(nil)
                    .fixedSize(horizontal: true, vertical: false)
                    .padding(.horizontal, 12)
                    .padding(.bottom, 12)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.sceneBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .accessibilityElement(children: .combine)
        .accessibilityLabel(codeAccessibilityLabel)
    }

    private var languageLabel: String {
        if let language, !language.isEmpty {
            return L.format("mobile.markdown.code.language", language)
        }
        return L.text("mobile.markdown.code.languageFallback")
    }

    private var codeAccessibilityLabel: String {
        if let language, !language.isEmpty {
            return "\(language) \(L.text("mobile.markdown.code.block"))"
        }
        return L.text("mobile.markdown.code.block")
    }
}
