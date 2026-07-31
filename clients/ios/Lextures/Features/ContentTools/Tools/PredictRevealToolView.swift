import SwiftUI

struct PredictRevealToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var outcomeId = ""
    @State private var text = ""
    @State private var confidence: Double?
    @State private var reflection = ""
    @State private var busy = false
    @State private var revealMarkdown: String?
    @State private var peerRows: [(label: String, count: Int)] = []
    @State private var peersSuppressed = false
    @State private var errorText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            CourseMarkdownContentView(markdown: question, compact: true)

            if canEdit {
                predictionInput
                if !confidenceOptions.isEmpty {
                    Text(L.text("mobile.contentTools.tools.predict_reveal.howSure"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    confidencePicker
                }
                Button(L.text("mobile.contentTools.tools.predict_reveal.commit")) {
                    Task { await commit() }
                }
                .disabled(busy || !canCommit)
                Text(L.text("mobile.contentTools.tools.predict_reveal.commitHelper"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if committed {
                lockedPrediction
                if ContentToolPack1Logic.canShowReveal(committed: true, hasRevealPayload: revealMarkdown != nil),
                   let revealMarkdown {
                    Text(L.text("mobile.contentTools.tools.predict_reveal.whatHappens"))
                        .font(.subheadline.weight(.semibold))
                    CourseMarkdownContentView(markdown: revealMarkdown, compact: true)
                } else if committed && revealMarkdown == nil {
                    Button(L.text("mobile.contentTools.tools.predict_reveal.commit")) {
                        Task { await commit(reloading: true) }
                    }
                    .disabled(busy)
                }
                if peersSuppressed {
                    Text(L.text("mobile.contentTools.tools.predict_reveal.peersSuppressed"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else if !peerRows.isEmpty {
                    Text(L.text("mobile.contentTools.tools.predict_reveal.peerResults"))
                        .font(.caption.weight(.semibold))
                    ForEach(peerRows, id: \.label) { row in
                        Text("\(row.label): \(row.count)")
                            .font(.caption)
                    }
                }
                if !reflectionPrompt.isEmpty {
                    Text(reflectionPrompt)
                        .font(.caption)
                    TextField(
                        L.text("mobile.contentTools.tools.predict_reveal.reflectionPlaceholder"),
                        text: $reflection,
                        axis: .vertical
                    )
                    .lineLimit(2 ... 6)
                    .disabled(props.readOnly || busy)
                    Button(L.text("mobile.contentTools.tools.predict_reveal.saveReflection")) {
                        Task { await reflect() }
                    }
                    .disabled(props.readOnly || busy || reflection.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear { hydrate(); Task { if committed { await commit(reloading: true) } } }
        .onChange(of: props.state) { _, _ in hydrate() }
    }

    private var question: String {
        ContentToolHostLogic.stringField(props.config, key: "question") ?? ""
    }

    private var mode: String {
        ContentToolHostLogic.stringField(props.config, key: "mode") == "open" ? "open" : "choice"
    }

    private var confidenceScale: String {
        ContentToolHostLogic.stringField(props.config, key: "confidenceScale") ?? "three"
    }

    private var confidenceRequired: Bool {
        ContentToolPack1Logic.boolField(props.config, key: "confidenceRequired") != false
    }

    private var reflectionPrompt: String {
        ContentToolHostLogic.stringField(props.config, key: "reflectionPrompt") ?? ""
    }

    private var outcomes: [(id: String, text: String)] {
        ContentToolPack1Logic.arrayField(props.config, key: "outcomes").compactMap { raw in
            let o = ContentToolPack1Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let text) = o["text"] else { return nil }
            return (id, text)
        }
    }

    private var committed: Bool { ContentToolPack1Logic.isCommitted(props.state) }
    private var canEdit: Bool { ContentToolPack1Logic.canEditPrediction(committed: committed, readOnly: props.readOnly) }

    private var canCommit: Bool {
        if mode == "open" {
            if text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return false }
        } else if outcomeId.isEmpty {
            return false
        }
        if confidenceRequired && confidenceScale != "none" && confidence == nil { return false }
        return true
    }

    private var confidenceOptions: [(value: Double, label: String)] {
        switch confidenceScale {
        case "none": return []
        case "five":
            return (1 ... 5).map { (Double($0), String($0)) }
        case "percent":
            return [0, 25, 50, 75, 100].map { (Double($0), "\($0)%") }
        default:
            return [
                (1, L.text("mobile.contentTools.tools.predict_reveal.confidence.guessing")),
                (2, L.text("mobile.contentTools.tools.predict_reveal.confidence.fairlySure")),
                (3, L.text("mobile.contentTools.tools.predict_reveal.confidence.certain")),
            ]
        }
    }

    @ViewBuilder
    private var predictionInput: some View {
        Text(L.text("mobile.contentTools.tools.predict_reveal.yourPrediction"))
            .font(.caption.weight(.semibold))
        if mode == "open" {
            TextField(
                ContentToolHostLogic.stringField(props.config, key: "openPlaceholder")
                    ?? L.text("mobile.contentTools.tools.predict_reveal.openPlaceholder"),
                text: $text,
                axis: .vertical
            )
            .lineLimit(2 ... 6)
            .onChange(of: text) { _, _ in persistDraft() }
        } else {
            ForEach(outcomes, id: \.id) { outcome in
                Button {
                    outcomeId = outcome.id
                    persistDraft()
                } label: {
                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: outcomeId == outcome.id ? "largecircle.fill.circle" : "circle")
                        CourseMarkdownContentView(markdown: outcome.text, compact: true)
                    }
                }
                .accessibilityLabel(outcome.text)
                .accessibilityAddTraits(outcomeId == outcome.id ? .isSelected : [])
            }
        }
    }

    private var confidencePicker: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(confidenceOptions, id: \.value) { opt in
                Button {
                    confidence = opt.value
                    persistDraft()
                } label: {
                    HStack {
                        Image(systemName: confidence == opt.value ? "largecircle.fill.circle" : "circle")
                        Text(opt.label)
                    }
                }
                .accessibilityLabel(opt.label)
            }
        }
    }

    @ViewBuilder
    private var lockedPrediction: some View {
        Text(L.text("mobile.contentTools.tools.predict_reveal.yourPrediction"))
            .font(.caption.weight(.semibold))
        if mode == "open" {
            Text(text.isEmpty ? (ContentToolHostLogic.stringField(props.state, key: "prediction").map { _ in "" } ?? text) : text)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        } else {
            let label = outcomes.first(where: { $0.id == outcomeId })?.text ?? outcomeId
            CourseMarkdownContentView(markdown: label, compact: true)
        }
    }

    private func hydrate() {
        let draft = ContentToolPack1Logic.objectMap(ContentToolPack1Logic.objectMap(props.state)["draft"])
        let prediction = ContentToolPack1Logic.objectMap(ContentToolPack1Logic.objectMap(props.state)["prediction"])
        if case .string(let o) = draft["outcomeId"] ?? prediction["outcomeId"] { outcomeId = o }
        if case .string(let t) = draft["text"] ?? prediction["text"] { text = t }
        if case .number(let c) = draft["confidence"] { confidence = c }
        if let saved = ContentToolHostLogic.stringField(props.state, key: "reflection") { reflection = saved }
    }

    private func persistDraft() {
        guard canEdit else { return }
        var draft: [String: JSONValue] = [:]
        if mode == "open" {
            draft["text"] = .string(text)
        } else {
            draft["outcomeId"] = .string(outcomeId)
        }
        if let confidence { draft["confidence"] = .number(confidence) }
        let patch = ContentToolPack1Logic.mergePreservingUnknown(
            base: ContentToolPack1Logic.objectMap(props.state),
            patch: ["v": .number(1), "draft": .object(draft)]
        )
        props.save(patch)
    }

    private func commit(reloading: Bool = false) async {
        if busy { return }
        busy = true
        errorText = nil
        defer { busy = false }
        var input: [String: JSONValue] = [:]
        if !reloading {
            if mode == "open" {
                input["prediction"] = .object(["text": .string(text)])
            } else {
                input["prediction"] = .object(["outcomeId": .string(outcomeId)])
            }
            if let confidence { input["confidence"] = .number(confidence) }
        }
        do {
            let raw = try await props.runAction("commit", .object(input))
            applyCommitResult(raw)
            Haptics.trigger(.success)
            if revealMarkdown != nil {
                props.announce(L.text("mobile.contentTools.tools.predict_reveal.revealedAnnounce"), false)
            }
        } catch {
            errorText = L.text("mobile.contentTools.runtime.needsConnection")
            props.announce(errorText!, true)
        }
    }

    private func reflect() async {
        if busy || props.readOnly { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("reflect", .object(["text": .string(reflection)]))
            Haptics.trigger(.success)
        } catch {
            errorText = L.text("mobile.contentTools.runtime.needsConnection")
            props.announce(errorText!, true)
        }
    }

    private func applyCommitResult(_ raw: JSONValue?) {
        guard case .object(let obj) = raw else { return }
        if case .object(let reveal) = obj["reveal"], case .string(let md) = reveal["markdown"] {
            revealMarkdown = md
        }
        if case .object(let peers) = obj["peerResults"] {
            if case .bool(true) = peers["suppressed"] {
                peersSuppressed = true
                peerRows = []
            } else if case .array(let outcomes) = peers["outcomes"] {
                peersSuppressed = false
                peerRows = outcomes.compactMap { row in
                    let o = ContentToolPack1Logic.objectMap(row)
                    guard case .string(let id) = o["outcomeId"] else { return nil }
                    let count = ContentToolPack1Logic.numberField(row, key: "count").map { Int($0) } ?? 0
                    let label = self.outcomes.first(where: { $0.id == id })?.text ?? id
                    return (label, count)
                }
            }
        }
        if case .object(let pred) = ContentToolPack1Logic.objectMap(props.state)["prediction"] {
            if case .string(let o) = pred["outcomeId"] { outcomeId = o }
            if case .string(let t) = pred["text"] { text = t }
        }
    }
}
