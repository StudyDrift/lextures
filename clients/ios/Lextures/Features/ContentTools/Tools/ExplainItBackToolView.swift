// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct ExplainItBackToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.contentToolsPage) private var page
    @Environment(AuthSession.self) private var session
    let props: ContentToolRendererProps

    @State private var draft = ""
    @State private var busy = false
    @State private var errorText: String?
    @State private var revising = false
    @State private var consent: ContentToolAIConsent?
    @State private var consentFetched = false
    @State private var labels: [String: String] = [:]

    private var draftKey: String {
        ContentToolPack2Logic.draftStorageKey(instanceId: props.instanceId)
    }

    private var online: Bool { NetworkMonitor.shared.isOnline }

    private var consentAllowed: Bool {
        // When AI feedback is disabled, consent is not required for submit.
        if ContentToolPack2Logic.boolField(props.config, key: "aiFeedback") == false {
            return true
        }
        return ContentToolPack2Logic.composerAIAllowed(
            disclosureMode: consent?.aiDisclosureMode,
            decision: consent?.decision,
            consentFetched: consentFetched
        )
    }

    /// CT.M9: disclosure lives in ToolFrame chrome so sandboxed tools cannot cover it.
    private var showDisclosure: Bool { false }

    private var minWords: Int {
        Int(ContentToolPack2Logic.numberField(props.config, key: "minWords") ?? 25)
    }

    private var maxWords: Int {
        Int(ContentToolPack2Logic.numberField(props.config, key: "maxWords") ?? 150)
    }

    private var maxAttempts: Int {
        Int(ContentToolPack2Logic.numberField(props.config, key: "attempts") ?? 3)
    }

    private var attempts: [(at: String, text: String, feedback: Feedback?)] {
        ContentToolPack2Logic.arrayField(props.state, key: "attempts").compactMap { raw in
            let o = ContentToolPack2Logic.objectMap(raw)
            guard case .string(let text) = o["text"] else { return nil }
            let at: String = {
                if case .string(let v) = o["at"] { return v }
                return ""
            }()
            return (at, text, parseFeedback(o["feedback"]))
        }
    }

    private struct Feedback {
        var covered: [String]
        var missing: [String]
        var strength: String
        var suggestion: String
        var probe: String?
        var mode: String
    }

    private var latest: (at: String, text: String, feedback: Feedback?)? { attempts.last }
    private var wordCount: Int { ContentToolPack2Logic.wordCount(draft) }
    private var attemptsLeft: Int { max(0, maxAttempts - attempts.count) }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            let prompt = ContentToolHostLogic.stringField(props.config, key: "prompt")
                ?? L.text("mobile.contentTools.tools.explain_it_back.defaultPrompt")
            CourseMarkdownContentView(markdown: prompt, compact: true)

            Text(L.format("mobile.contentTools.tools.explain_it_back.lengthGuide", minWords, maxWords))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            Text(L.text("mobile.contentTools.tools.explain_it_back.practiceNote"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if showDisclosure {
                disclosureBanner
            }

            if let note = instructorNote {
                VStack(alignment: .leading, spacing: 4) {
                    Text(L.text("mobile.contentTools.tools.explain_it_back.instructorNote"))
                        .font(.caption.weight(.semibold))
                    CourseMarkdownContentView(markdown: note, compact: true)
                }
            }

            if let latest, let feedback = latest.feedback, !revising {
                feedbackCard(feedback)
                if attemptsLeft > 0, !props.readOnly {
                    Button(L.format("mobile.contentTools.tools.explain_it_back.revise", attemptsLeft)) {
                        revising = true
                        draft = latest.text
                    }
                }
            } else if !props.readOnly {
                if consentAllowed || ContentToolPack2Logic.boolField(props.config, key: "aiFeedback") == false {
                    ToolComposerView(
                        placeholder: L.text("mobile.contentTools.tools.explain_it_back.inputLabel"),
                        sendLabel: ContentToolPack2Logic.boolField(props.config, key: "aiFeedback") == false
                            ? L.text("mobile.contentTools.tools.explain_it_back.submitReview")
                            : L.text("mobile.contentTools.tools.explain_it_back.submitFeedback"),
                        cancelLabel: nil,
                        text: $draft,
                        draftKey: draftKey,
                        enabled: true,
                        online: online,
                        busy: busy,
                        onSend: { Task { await submit() } }
                    )
                    Text(L.format(
                        "mobile.contentTools.tools.explain_it_back.wordCount",
                        wordCount,
                        minWords,
                        maxWords
                    ))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if attemptsLeft < maxAttempts {
                        Text(L.format("mobile.contentTools.tools.explain_it_back.attemptsLeft", attemptsLeft))
                            .font(.caption2)
                    }
                } else if consentFetched {
                    Text(L.text("mobile.contentTools.ai.consentRequired"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            } else if let latest {
                CourseMarkdownContentView(markdown: latest.text, compact: true)
                if let feedback = latest.feedback {
                    feedbackCard(feedback)
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .task { await loadConsent() }
        .onAppear {
            if draft.isEmpty {
                draft = ContentToolHostLogic.stringField(props.state, key: "draft")
                    ?? ContentToolDraftStore.load(key: draftKey)
            }
        }
    }

    private var instructorNote: String? {
        let note = ContentToolPack2Logic.objectMap(
            ContentToolPack2Logic.objectMap(props.state)["instructorNote"]
        )
        if case .string(let text) = note["text"], !text.isEmpty { return text }
        return nil
    }

    @ViewBuilder
    private var disclosureBanner: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.contentTools.ai.disclosureTitle"))
                .font(.caption.weight(.semibold))
            Text(L.text("mobile.contentTools.ai.disclosureBody"))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            HStack {
                Button(L.text("mobile.contentTools.ai.acknowledge")) {
                    Task { await setConsent("acknowledged") }
                }
                Button(L.text("mobile.contentTools.ai.optOut")) {
                    Task { await setConsent("opted_out") }
                }
            }
        }
        .padding(10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    @ViewBuilder
    private func feedbackCard(_ feedback: Feedback) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(
                feedback.mode == "review"
                    ? L.text("mobile.contentTools.tools.explain_it_back.reviewTitle")
                    : L.text("mobile.contentTools.tools.explain_it_back.feedbackTitle")
            )
            .font(.subheadline.weight(.semibold))
            if !feedback.covered.isEmpty {
                Text(L.text("mobile.contentTools.tools.explain_it_back.whatYouGot"))
                    .font(.caption.weight(.semibold))
                ForEach(feedback.covered, id: \.self) { id in
                    Text("• \(labels[id] ?? id)")
                        .font(.caption)
                }
            }
            if !feedback.missing.isEmpty {
                Text(L.text("mobile.contentTools.tools.explain_it_back.whatsMissing"))
                    .font(.caption.weight(.semibold))
                ForEach(feedback.missing, id: \.self) { id in
                    Text("• \(labels[id] ?? id)")
                        .font(.caption)
                }
            }
            if !feedback.strength.isEmpty {
                Text(L.text("mobile.contentTools.tools.explain_it_back.strength"))
                    .font(.caption.weight(.semibold))
                CourseMarkdownContentView(markdown: feedback.strength, compact: true)
            }
            if !feedback.suggestion.isEmpty {
                Text(L.text("mobile.contentTools.tools.explain_it_back.suggestion"))
                    .font(.caption.weight(.semibold))
                CourseMarkdownContentView(markdown: feedback.suggestion, compact: true)
            }
            if let probe = feedback.probe, !probe.isEmpty {
                Text(L.text("mobile.contentTools.tools.explain_it_back.probe"))
                    .font(.caption.weight(.semibold))
                CourseMarkdownContentView(markdown: probe, compact: true)
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(L.text("mobile.contentTools.tools.explain_it_back.feedbackReceived"))
    }

    private func parseFeedback(_ value: JSONValue?) -> Feedback? {
        guard let value else { return nil }
        let o = ContentToolPack2Logic.objectMap(value)
        let covered = ContentToolPack2Logic.arrayField(value, key: "covered").compactMap { item -> String? in
            if case .string(let s) = item { return s }
            return nil
        }
        let missing = ContentToolPack2Logic.arrayField(value, key: "missing").compactMap { item -> String? in
            if case .string(let s) = item { return s }
            return nil
        }
        let strength: String = { if case .string(let s) = o["strength"] { return s }; return "" }()
        let suggestion: String = { if case .string(let s) = o["suggestion"] { return s }; return "" }()
        let probe: String? = { if case .string(let s) = o["probe"] { return s }; return nil }()
        let mode: String = { if case .string(let s) = o["mode"] { return s }; return "ai" }()
        return Feedback(
            covered: covered,
            missing: missing,
            strength: strength,
            suggestion: suggestion,
            probe: probe,
            mode: mode
        )
    }

    private func loadConsent() async {
        guard ContentToolPack2Logic.boolField(props.config, key: "aiFeedback") != false else {
            consentFetched = true
            return
        }
        guard let page, let token = session.accessToken else {
            consentFetched = false
            return
        }
        do {
            consent = try await LMSAPI.fetchContentToolAIConsent(
                courseCode: page.courseCode,
                toolId: props.toolId,
                accessToken: token
            )
            consentFetched = true
        } catch {
            consentFetched = false
        }
    }

    private func setConsent(_ decision: String) async {
        guard let page, let token = session.accessToken else { return }
        do {
            consent = try await LMSAPI.postContentToolAIConsent(
                courseCode: page.courseCode,
                toolId: props.toolId,
                decision: decision,
                accessToken: token
            )
            consentFetched = true
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
        }
    }

    private func submit() async {
        let text = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard ContentToolPack2Logic.canSubmitExplanation(
            text: text,
            minWords: minWords,
            maxWords: maxWords,
            attemptsUsed: attempts.count,
            maxAttempts: maxAttempts,
            readOnly: props.readOnly,
            online: online,
            consentAllowed: consentAllowed
        ) else {
            if !ContentToolPack2Logic.lengthGuidanceOK(text: text, minWords: minWords, maxWords: maxWords) {
                errorText = wordCount < minWords
                    ? L.text("mobile.contentTools.tools.explain_it_back.error.tooShort")
                    : L.text("mobile.contentTools.tools.explain_it_back.error.tooLong")
            }
            return
        }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let raw = try await props.runAction("submit", .object(["text": .string(text)]))
            let result = ContentToolPack2Logic.objectMap(raw)
            if case .string(let code) = result["error"] {
                errorText = L.dynamicText(ContentToolPack2Logic.plainLanguageMessageKey(for: code))
                return
            }
            if case .object(let keyLabels) = result["keyPointLabels"] {
                for (k, v) in keyLabels {
                    if case .string(let label) = v { labels[k] = label }
                }
            }
            draft = ""
            revising = false
            ToolComposerView.clearDraft(key: draftKey)
            props.save(["draft": .string("")])
            props.announce(L.text("mobile.contentTools.tools.explain_it_back.feedbackReceived"), false)
        } catch {
            errorText = online
                ? L.text("mobile.contentTools.runtime.retry")
                : L.text("mobile.contentTools.runtime.offlineComposer")
        }
    }
}
