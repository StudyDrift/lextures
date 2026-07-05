import SwiftUI

struct CodeEditor: View {
    @Environment(\.colorScheme) private var colorScheme

    @Binding var text: String
    let language: String
    let onInsert: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 6) {
                    ForEach(QuizLogic.codeSymbolSnippets, id: \.self) { snippet in
                        Button {
                            onInsert(snippet)
                        } label: {
                            Text(symbolLabel(snippet))
                                .font(.caption.monospaced())
                                .padding(.horizontal, 8)
                                .padding(.vertical, 6)
                                .background(LexturesTheme.sceneBackground(for: colorScheme))
                                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel(L.text("mobile.quiz.code.insertSymbol"))
                    }
                }
            }

            TextEditor(text: Binding(
                get: { text },
                set: { next in text = QuizLogic.applyAutoIndent(to: next) }
            ))
            .font(.system(.body, design: .monospaced))
            .frame(minHeight: 180)
            .padding(8)
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.textSecondary(for: colorScheme).opacity(0.25), lineWidth: 1)
            )
            .accessibilityLabel(L.text("mobile.quiz.code.editorA11y"))
        }
        .accessibilityElement(children: .contain)
    }

    private func symbolLabel(_ snippet: String) -> String {
        snippet
            .replacingOccurrences(of: "\n", with: "↵")
            .replacingOccurrences(of: "    ", with: "⇥")
    }
}
