import SwiftUI

struct NoopProbeRendererView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps
    @State private var response = ""
    @State private var checking = false
    @State private var resultText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(prompt)
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            TextField(L.text("mobile.contentTools.runtime.yourAnswer"), text: $response, axis: .vertical)
                .lineLimit(3 ... 6)
                .disabled(props.readOnly || checking)
                .onChange(of: response) { _, value in
                    guard !props.readOnly else { return }
                    props.save(["response": .string(value)])
                }
            Button(L.text("mobile.contentTools.runtime.checkAnswer")) {
                Task { await checkAnswer() }
            }
            .disabled(props.readOnly || checking || response.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            if let resultText {
                Text(resultText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
        .onAppear {
            response = ContentToolHostLogic.stringField(props.state, key: "response") ?? ""
        }
        .onChange(of: props.state) { _, newState in
            if let remote = ContentToolHostLogic.stringField(newState, key: "response"), remote != response {
                response = remote
            }
        }
    }

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt")?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty ?? props.toolId
    }

    private func checkAnswer() async {
        checking = true
        resultText = nil
        defer { checking = false }
        do {
            let raw = try await props.runAction("grade", .object(["response": .string(response)]))
            resultText = ContentToolHostLogic.stringField(raw, key: "reason")
                ?? ContentToolHostLogic.stringField(raw, key: "correct")
            if case .object(let obj) = raw, case .bool(true) = obj["correct"] {
                props.announce(L.text("mobile.contentTools.runtime.score"), false)
            }
        } catch {
            resultText = L.text("mobile.contentTools.runtime.needsConnection")
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
