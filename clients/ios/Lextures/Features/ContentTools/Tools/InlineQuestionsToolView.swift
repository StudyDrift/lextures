// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct InlineQuestionsToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var drafts: [String: JSONValue] = [:]
    @State private var busyId: String?
    @State private var lastResults: [String: JSONValue] = [:]
    @State private var pendingSubmit: [String: JSONValue] = [:]
    @State private var gradingPending = false

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(label)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if gradingPending {
                Label(
                    L.text("mobile.contentTools.tools.inline_questions.gradingPending"),
                    systemImage: "icloud.and.arrow.up"
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            ForEach(questions, id: \.id) { question in
                questionBlock(question)
            }
        }
        .onAppear { hydrate() }
        .onChange(of: props.state) { _, _ in hydrate() }
    }

    private var label: String {
        ContentToolHostLogic.stringField(props.config, key: "label")?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty
            ?? L.text("mobile.contentTools.tools.inline_questions.checkLabel")
    }

    private var sequential: Bool {
        ContentToolPack1Logic.boolField(props.config, key: "sequential") == true
    }

    private var shuffleOptions: Bool {
        ContentToolPack1Logic.boolField(props.config, key: "shuffleOptions") == true
    }

    private var maxAttempts: Int? {
        let cfg = ContentToolPack1Logic.objectMap(props.config)
        return ContentToolPack1Logic.parseAttemptsConfig(cfg["attempts"])
    }

    private var answers: [String: JSONValue] {
        ContentToolPack1Logic.objectMap(ContentToolPack1Logic.objectMap(props.state)["answers"])
    }

    private struct Question: Identifiable {
        var id: String
        var type: String
        var prompt: String
        var options: [(id: String, text: String)]
        var unit: String?
    }

    private var questions: [Question] {
        ContentToolPack1Logic.arrayField(props.config, key: "questions").compactMap { raw in
            let obj = ContentToolPack1Logic.objectMap(raw)
            guard case .string(let id) = obj["id"], !id.isEmpty else { return nil }
            let type: String
            if case .string(let t) = obj["type"] { type = t } else { type = "single" }
            let prompt: String
            if case .string(let p) = obj["prompt"] { prompt = p } else { prompt = "" }
            var options: [(id: String, text: String)] = []
            if case .array(let opts) = obj["options"] {
                for opt in opts {
                    let o = ContentToolPack1Logic.objectMap(opt)
                    guard case .string(let oid) = o["id"], case .string(let text) = o["text"] else { continue }
                    options.append((oid, text))
                }
            }
            if shuffleOptions {
                options = ContentToolPack1Logic.shuffleStable(options, seed: "\(props.instanceId):\(id)") { $0.id }
            }
            let unit: String?
            if case .string(let u) = obj["unit"] { unit = u } else { unit = nil }
            return Question(id: id, type: type, prompt: prompt, options: options, unit: unit)
        }
    }

    @ViewBuilder
    private func questionBlock(_ q: Question) -> some View {
        let unlocked = ContentToolPack1Logic.isSequentiallyUnlocked(
            questions: questions.map(\.id),
            answers: answers,
            questionId: q.id,
            sequential: sequential
        )
        let canSubmit = ContentToolPack1Logic.canSubmit(
            answers: answers,
            questionId: q.id,
            maxAttempts: maxAttempts,
            readOnly: props.readOnly
        ) && unlocked
        let used = ContentToolPack1Logic.attemptsUsed(answers: answers, questionId: q.id)

        VStack(alignment: .leading, spacing: 10) {
            CourseMarkdownContentView(markdown: q.prompt, compact: true)
            if !unlocked {
                Text(L.text("mobile.contentTools.tools.inline_questions.sequentialLocked"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                input(for: q, enabled: canSubmit && busyId == nil)
            }
            if let unit = q.unit, !unit.isEmpty {
                Text(unit)
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            HStack {
                Button(L.text("mobile.contentTools.tools.inline_questions.submit")) {
                    Task { await submit(q) }
                }
                .disabled(!canSubmit || busyId != nil || !hasDraft(q.id))
                if let maxAttempts {
                    Text(L.format("mobile.contentTools.tools.inline_questions.attemptsLeft", max(0, maxAttempts - used)))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            if let result = lastResults[q.id] ?? feedbackFromAnswers(q.id) {
                feedbackView(result)
            }
        }
        .padding(.vertical, 4)
        .opacity(unlocked ? 1 : 0.7)
    }

    @ViewBuilder
    private func input(for q: Question, enabled: Bool) -> some View {
        switch q.type {
        case "multi":
            ForEach(q.options, id: \.id) { opt in
                let selected = multiSelected(q.id).contains(opt.id)
                Button {
                    toggleMulti(q.id, optionId: opt.id)
                } label: {
                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: selected ? "checkmark.square.fill" : "square")
                        CourseMarkdownContentView(markdown: opt.text, compact: true)
                    }
                }
                .disabled(!enabled)
                .accessibilityLabel(opt.text)
                .accessibilityAddTraits(selected ? .isSelected : [])
            }
        case "short_text", "numeric":
            TextField(L.text("mobile.contentTools.runtime.yourAnswer"), text: bindingText(q.id), axis: .vertical)
                .lineLimit(1 ... 4)
                .disabled(!enabled)
                .keyboardType(q.type == "numeric" ? .decimalPad : .default)
        default:
            ForEach(q.options, id: \.id) { opt in
                let selected = stringDraft(q.id) == opt.id
                Button {
                    setDraft(q.id, .string(opt.id))
                } label: {
                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: selected ? "largecircle.fill.circle" : "circle")
                        CourseMarkdownContentView(markdown: opt.text, compact: true)
                    }
                }
                .disabled(!enabled)
                .accessibilityLabel(opt.text)
                .accessibilityAddTraits(selected ? .isSelected : [])
            }
        }
    }

    @ViewBuilder
    private func feedbackView(_ result: JSONValue) -> some View {
        let correct = ContentToolPack1Logic.boolField(result, key: "correct") == true
        let icon = correct ? "checkmark.circle.fill" : "xmark.circle.fill"
        let title = correct
            ? L.text("mobile.contentTools.tools.inline_questions.correct")
            : L.text("mobile.contentTools.tools.inline_questions.incorrect")
        VStack(alignment: .leading, spacing: 6) {
            Label(title, systemImage: icon)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(correct ? LexturesTheme.primary : LexturesTheme.coral)
            if let feedback = ContentToolHostLogic.stringField(result, key: "feedback"), !feedback.isEmpty {
                CourseMarkdownContentView(markdown: feedback, compact: true)
            }
            if let explanation = ContentToolHostLogic.stringField(result, key: "explanation"), !explanation.isEmpty {
                CourseMarkdownContentView(markdown: explanation, compact: true)
            }
        }
        .accessibilityElement(children: .combine)
    }

    private func hydrate() {
        let remote = ContentToolPack1Logic.objectMap(ContentToolPack1Logic.objectMap(props.state)["drafts"])
        if drafts.isEmpty || drafts != remote {
            drafts = remote
        }
    }

    private func hasDraft(_ qid: String) -> Bool {
        guard let v = drafts[qid] else { return false }
        switch v {
        case .string(let s): return !s.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        case .array(let a): return !a.isEmpty
        case .number: return true
        default: return false
        }
    }

    private func stringDraft(_ qid: String) -> String? {
        if case .string(let s) = drafts[qid] { return s }
        return nil
    }

    private func multiSelected(_ qid: String) -> Set<String> {
        guard case .array(let arr) = drafts[qid] else { return [] }
        return Set(arr.compactMap {
            if case .string(let s) = $0 { return s }
            return nil
        })
    }

    private func bindingText(_ qid: String) -> Binding<String> {
        Binding(
            get: { stringDraft(qid) ?? "" },
            set: { setDraft(qid, .string($0)) }
        )
    }

    private func setDraft(_ qid: String, _ value: JSONValue) {
        guard !props.readOnly else { return }
        drafts[qid] = value
        persistDrafts()
    }

    private func toggleMulti(_ qid: String, optionId: String) {
        var set = multiSelected(qid)
        if set.contains(optionId) { set.remove(optionId) } else { set.insert(optionId) }
        setDraft(qid, .array(set.sorted().map { .string($0) }))
    }

    private func persistDrafts() {
        let patch = ContentToolPack1Logic.mergePreservingUnknown(
            base: ContentToolPack1Logic.objectMap(props.state),
            patch: [
                "v": .number(1),
                "drafts": .object(drafts),
                "answers": .object(answers),
            ]
        )
        props.save(patch)
    }

    private func feedbackFromAnswers(_ qid: String) -> JSONValue? {
        guard case .object(let q)? = answers[qid],
              case .array(let attempts) = q["attempts"],
              let last = attempts.last,
              case .object(let obj) = last
        else { return nil }
        return .object(obj)
    }

    private func submit(_ q: Question) async {
        guard let value = drafts[q.id], busyId == nil else { return }
        busyId = q.id
        defer { busyId = nil }
        do {
            let raw = try await props.runAction(
                "submit",
                .object(["questionId": .string(q.id), "value": value])
            )
            if let raw {
                lastResults[q.id] = raw
                let correct = ContentToolPack1Logic.boolField(raw, key: "correct") == true
                Haptics.trigger(correct ? .success : .error)
                props.announce(
                    correct
                        ? L.text("mobile.contentTools.tools.inline_questions.correctAnnounce")
                        : L.text("mobile.contentTools.tools.inline_questions.incorrectAnnounce"),
                    false
                )
            }
            gradingPending = false
            pendingSubmit.removeValue(forKey: q.id)
        } catch {
            if ContentToolPack1Logic.canQueueActionOffline(toolId: "inline_questions", action: "submit") {
                pendingSubmit[q.id] = value
                gradingPending = true
                props.announce(L.text("mobile.contentTools.tools.inline_questions.gradingPending"), false)
            } else {
                props.announce(L.text("mobile.contentTools.runtime.needsConnection"), true)
            }
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : self
    }
}
