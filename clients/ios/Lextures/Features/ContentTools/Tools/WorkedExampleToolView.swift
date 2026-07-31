// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct WorkedExampleToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var draft = ""
    @State private var busy = false
    @State private var prepared = false
    @State private var errorText: String?
    @State private var lastFeedback: String?
    @State private var lastResult: String?
    @State private var hintText: String?
    @State private var revealText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            if !title.isEmpty {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            if !problem.isEmpty {
                CourseMarkdownContentView(markdown: problem, compact: true)
            }

            ForEach(allStepIds, id: \.self) { stepId in
                stepRow(stepId)
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear {
            hydrate()
            Task { await prepareIfNeeded() }
        }
        .onChange(of: props.state) { _, _ in hydrate() }
    }

    private var title: String {
        ContentToolHostLogic.stringField(props.config, key: "title")?
            .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    }

    private var problem: String {
        ContentToolHostLogic.stringField(props.config, key: "problem") ?? ""
    }

    private var allowRevealAll: Bool {
        ContentToolPack4Logic.boolField(props.config, key: "allowRevealAll") == true
    }

    private var allStepIds: [String] {
        ContentToolPack4Logic.parseStepIds(props.config)
    }

    private var blanked: [String] {
        ContentToolPack4Logic.blankedStepIds(config: props.config, state: props.state)
    }

    private var progress: [String: JSONValue] {
        ContentToolPack4Logic.stepProgressMap(props.state)
    }

    private var currentStepId: String {
        ContentToolPack4Logic.resolveCurrentStepId(
            blankedStepIds: blanked,
            currentStepId: ContentToolHostLogic.stringField(props.state, key: "currentStepId"),
            progress: progress
        )
    }

    @ViewBuilder
    private func stepRow(_ stepId: String) -> some View {
        let status = ContentToolPack4Logic.stepStatus(
            stepId: stepId,
            blankedStepIds: blanked,
            currentStepId: currentStepId,
            progress: progress,
            allStepIds: allStepIds
        )
        let step = stepObject(stepId)
        let label = ContentToolHostLogic.stringField(step, key: "label")
            ?? ContentToolHostLogic.stringField(step, key: "text")
            ?? stepId

        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Image(systemName: statusIcon(status))
                    .foregroundStyle(statusColor(status))
                    .accessibilityHidden(true)
                CourseMarkdownContentView(markdown: label, compact: true)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(statusLabel(status)): \(label)")

            if status == .current {
                TextField(
                    L.text("mobile.contentTools.tools.worked_example.answerPlaceholder"),
                    text: $draft,
                    axis: .vertical
                )
                .lineLimit(1 ... 4)
                .disabled(props.readOnly || busy)
                .onChange(of: draft) { _, value in
                    persistDraft(value)
                }

                HStack(spacing: 10) {
                    Button(L.text("mobile.contentTools.tools.worked_example.check")) {
                        Task { await checkStep() }
                    }
                    .disabled(!ContentToolPack4Logic.canCheckStep(
                        draft: draft,
                        readOnly: props.readOnly,
                        busy: busy,
                        stepDone: false
                    ))

                    Button(L.text("mobile.contentTools.tools.worked_example.hint")) {
                        Task { await hint() }
                    }
                    .disabled(props.readOnly || busy)

                    Button(L.text("mobile.contentTools.tools.worked_example.reveal")) {
                        Task { await revealStep() }
                    }
                    .disabled(props.readOnly || busy)
                }

                if let lastResult {
                    Text(resultLabel(lastResult))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(lastResult == "correct" ? LexturesTheme.primary : LexturesTheme.coral)
                }
                if let lastFeedback, !lastFeedback.isEmpty {
                    CourseMarkdownContentView(markdown: lastFeedback, compact: true)
                }
                if let hintText, !hintText.isEmpty {
                    CourseMarkdownContentView(markdown: hintText, compact: true)
                }
                if let revealText, !revealText.isEmpty {
                    CourseMarkdownContentView(markdown: revealText, compact: true)
                }
            } else if status == .revealed {
                Text(L.text("mobile.contentTools.tools.worked_example.statusRevealed"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else if status == .solved {
                Text(L.text("mobile.contentTools.tools.worked_example.statusSolved"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.primary)
            }
        }
        .padding(.vertical, 4)
        .opacity(status == .locked ? 0.55 : 1)

        if status == .current && allowRevealAll {
            Button(L.text("mobile.contentTools.tools.worked_example.revealAll")) {
                Task { await revealAll() }
            }
            .disabled(props.readOnly || busy)
        }
    }

    private func statusIcon(_ status: ContentToolPack4Logic.StepStatus) -> String {
        switch status {
        case .solved: return "checkmark.circle.fill"
        case .revealed: return "eye.fill"
        case .current: return "pencil.circle.fill"
        case .scaffolded: return "text.book.closed"
        case .locked, .allComplete: return "lock.fill"
        }
    }

    private func statusColor(_ status: ContentToolPack4Logic.StepStatus) -> Color {
        switch status {
        case .solved: return LexturesTheme.primary
        case .revealed: return LexturesTheme.textSecondary(for: colorScheme)
        case .current: return LexturesTheme.accent(for: colorScheme)
        default: return LexturesTheme.textSecondary(for: colorScheme)
        }
    }

    private func statusLabel(_ status: ContentToolPack4Logic.StepStatus) -> String {
        switch status {
        case .solved: return L.text("mobile.contentTools.tools.worked_example.statusSolved")
        case .revealed: return L.text("mobile.contentTools.tools.worked_example.statusRevealed")
        case .current: return L.text("mobile.contentTools.tools.worked_example.statusCurrent")
        case .locked: return L.text("mobile.contentTools.tools.worked_example.statusLocked")
        case .scaffolded: return L.text("mobile.contentTools.tools.worked_example.statusScaffolded")
        case .allComplete: return L.text("mobile.contentTools.tools.worked_example.statusComplete")
        }
    }

    private func resultLabel(_ result: String) -> String {
        switch result {
        case "correct": return L.text("mobile.contentTools.tools.worked_example.correct")
        case "needs_review": return L.text("mobile.contentTools.tools.worked_example.needsReview")
        default: return L.text("mobile.contentTools.tools.worked_example.incorrect")
        }
    }

    private func stepObject(_ stepId: String) -> JSONValue? {
        ContentToolPack4Logic.arrayField(props.config, key: "steps").first { raw in
            ContentToolHostLogic.stringField(raw, key: "id") == stepId
        }
    }

    private func hydrate() {
        let sp = progress[currentStepId]
        if let saved = ContentToolHostLogic.stringField(sp, key: "draft") {
            draft = saved
        }
        if ContentToolPack4Logic.arrayField(props.state, key: "blankedStepIds").isEmpty == false {
            prepared = true
        }
    }

    private func persistDraft(_ value: String) {
        guard !props.readOnly else { return }
        let base = ContentToolPack4Logic.objectMap(props.state)
        let patch = ContentToolPack4Logic.mergeStepDraft(state: base, stepId: currentStepId, draft: value)
        props.save(patch)
    }

    private func prepareIfNeeded() async {
        guard !prepared, !props.readOnly else { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("prepare", .object([:]))
            prepared = true
        } catch {
            prepared = true
            errorText = L.text("mobile.contentTools.tools.worked_example.error")
        }
    }

    private func checkStep() async {
        guard ContentToolPack4Logic.canCheckStep(
            draft: draft,
            readOnly: props.readOnly,
            busy: busy,
            stepDone: false
        ) else { return }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let result = try await props.runAction(
                "check_step",
                .object([
                    "stepId": .string(currentStepId),
                    "value": .string(draft),
                ])
            )
            let parsed = ContentToolPack4Logic.parseCheckStepResult(result)
            lastResult = parsed.result
            lastFeedback = parsed.feedback
            if parsed.result == "correct" || parsed.result == "needs_review" {
                props.announce(L.text("mobile.contentTools.tools.worked_example.advanced"), false)
                hintText = nil
                revealText = nil
                draft = ""
            } else {
                props.announce(L.text("mobile.contentTools.tools.worked_example.incorrect"), true)
            }
        } catch {
            errorText = L.text("mobile.contentTools.tools.worked_example.error")
        }
    }

    private func hint() async {
        busy = true
        defer { busy = false }
        do {
            let result = try await props.runAction(
                "hint",
                .object(["stepId": .string(currentStepId)])
            )
            let parsed = ContentToolPack4Logic.parseHintResult(result)
            hintText = parsed.hint
            if let hint = parsed.hint {
                props.announce(hint, false)
            }
        } catch {
            errorText = L.text("mobile.contentTools.tools.worked_example.error")
        }
    }

    private func revealStep() async {
        busy = true
        defer { busy = false }
        do {
            let result = try await props.runAction(
                "reveal_step",
                .object(["stepId": .string(currentStepId)])
            )
            revealText = ContentToolHostLogic.stringField(result, key: "explanation")
                ?? ContentToolHostLogic.stringField(result, key: "expectedDisplay")
            props.announce(L.text("mobile.contentTools.tools.worked_example.statusRevealed"), false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.worked_example.error")
        }
    }

    private func revealAll() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("reveal_all", .object([:]))
            props.announce(L.text("mobile.contentTools.tools.worked_example.revealAllDone"), false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.worked_example.error")
        }
    }
}
