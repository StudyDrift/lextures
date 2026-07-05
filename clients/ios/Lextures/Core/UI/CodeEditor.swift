import SwiftUI
import UIKit

enum CodeSyntaxHighlighter {
    private static let keywordSets: [String: Set<String>] = [
        "python": pythonKeywords,
        "python3": pythonKeywords,
        "javascript": javascriptKeywords,
        "node": javascriptKeywords,
    ]

    private static let pythonKeywords: Set<String> = [
        "def", "return", "if", "elif", "else", "for", "while", "import", "from",
        "class", "pass", "break", "continue", "True", "False", "None", "in", "not", "and", "or",
    ]

    private static let javascriptKeywords: Set<String> = [
        "function", "return", "if", "else", "for", "while", "const", "let", "var", "class",
        "import", "export", "true", "false", "null", "undefined", "new",
    ]

    static func highlighted(_ text: String, language: String) -> NSAttributedString {
        let attributed = NSMutableAttributedString(
            string: text,
            attributes: [
                .font: UIFont.monospacedSystemFont(ofSize: 14, weight: .regular),
                .foregroundColor: UIColor.label,
            ]
        )

        let normalized = language.lowercased()
        let keywords = keywordSets[normalized] ?? keywordSets["python3"] ?? Set<String>()
        guard !keywords.isEmpty else { return attributed }
        let pattern = "\\b(" + keywords.sorted(by: { $0.count > $1.count }).joined(separator: "|") + ")\\b"
        guard let regex = try? NSRegularExpression(pattern: pattern) else { return attributed }
        let nsText = text as NSString
        let matches = regex.matches(in: text, range: NSRange(location: 0, length: nsText.length))
        for match in matches {
            attributed.addAttributes(
                [
                    .foregroundColor: UIColor.systemBlue,
                    .font: UIFont.monospacedSystemFont(ofSize: 14, weight: .semibold),
                ],
                range: match.range
            )
        }
        return attributed
    }
}

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
                            Text(snippet.replacingOccurrences(of: "\n", with: "↵").replacingOccurrences(of: "    ", with: "⇥"))
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

            CodeEditorRepresentable(text: $text, language: language)
                .frame(minHeight: 180)
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.textSecondary(for: colorScheme).opacity(0.25), lineWidth: 1)
                )
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.quiz.code.editorA11y"))
    }
}

private struct CodeEditorRepresentable: UIViewRepresentable {
    @Binding var text: String
    let language: String

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    func makeUIView(context: Context) -> UITextView {
        let view = UITextView()
        view.font = UIFont.monospacedSystemFont(ofSize: 14, weight: .regular)
        view.autocorrectionType = .no
        view.autocapitalizationType = .none
        view.smartDashesType = .no
        view.smartQuotesType = .no
        view.smartInsertDeleteType = .no
        view.backgroundColor = .secondarySystemBackground
        view.layer.cornerRadius = 12
        view.textContainerInset = UIEdgeInsets(top: 12, left: 10, bottom: 12, right: 10)
        view.delegate = context.coordinator
        view.accessibilityLabel = "Code editor"
        return view
    }

    func updateUIView(_ uiView: UITextView, context: Context) {
        context.coordinator.parent = self
        if uiView.text != text {
            let selected = uiView.selectedRange
            uiView.attributedText = CodeSyntaxHighlighter.highlighted(text, language: language)
            uiView.selectedRange = selected
        } else {
            uiView.attributedText = CodeSyntaxHighlighter.highlighted(text, language: language)
        }
    }

    final class Coordinator: NSObject, UITextViewDelegate {
        var parent: CodeEditorRepresentable

        init(parent: CodeEditorRepresentable) {
            self.parent = parent
        }

        func textViewDidChange(_ textView: UITextView) {
            let plain = textView.text ?? ""
            let next = QuizLogic.applyAutoIndent(to: plain)
            parent.text = next
            textView.attributedText = CodeSyntaxHighlighter.highlighted(next, language: parent.language)
        }
    }
}
