// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct AskQuestionsToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.contentToolsPage) private var page
    @Environment(AuthSession.self) private var session
    let props: ContentToolRendererProps

    @State private var draft = ""
    @State private var busy = false
    @State private var errorText: String?
    @State private var askInstructor = false
    @State private var showClearConfirm = false
    @State private var consent: ContentToolAIConsent?
    @State private var consentFetched = false
    @State private var inFlightTask: Task<Void, Never>?

    private var draftKey: String {
        ContentToolPack2Logic.draftStorageKey(instanceId: props.instanceId)
    }

    private var online: Bool { NetworkMonitor.shared.isOnline }

    private var consentAllowed: Bool {
        ContentToolPack2Logic.composerAIAllowed(
            disclosureMode: consent?.aiDisclosureMode,
            decision: consent?.decision,
            consentFetched: consentFetched
        )
    }

    private var showDisclosure: Bool {
        ContentToolPack2Logic.shouldShowAIDisclosure(
            disclosureMode: consent?.aiDisclosureMode,
            decision: consent?.decision,
            consentFetched: consentFetched
        )
    }

    private var turns: [(id: String, role: String, text: String, citations: [(title: String, url: String?)])] {
        ContentToolPack2Logic.arrayField(props.state, key: "turns").compactMap { raw in
            let o = ContentToolPack2Logic.objectMap(raw)
            guard case .string(let id) = o["id"],
                  case .string(let role) = o["role"],
                  case .string(let text) = o["text"]
            else { return nil }
            let citations: [(String, String?)] = {
                guard case .array(let arr) = o["citations"] else { return [] }
                return arr.compactMap { cite in
                    let c = ContentToolPack2Logic.objectMap(cite)
                    guard case .string(let title) = c["title"] else { return nil }
                    let url: String? = {
                        if case .string(let u) = c["url"] { return u }
                        return nil
                    }()
                    return (title, url)
                }
            }()
            return (id, role, text, citations)
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let intro = ContentToolHostLogic.stringField(props.config, key: "intro"), !intro.isEmpty {
                CourseMarkdownContentView(markdown: intro, compact: true)
            }

            if showDisclosure {
                disclosureBanner
            }

            if turns.isEmpty {
                Text(L.text("mobile.contentTools.tools.ask_questions.empty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 10) {
                        ForEach(turns, id: \.id) { turn in
                            turnRow(turn)
                        }
                    }
                }
                .frame(maxHeight: 280)
                .accessibilityElement(children: .contain)
                .accessibilityLabel(L.text("mobile.contentTools.tools.ask_questions.messagesLabel"))
            }

            if busy {
                HStack(spacing: 8) {
                    ProgressView()
                    Text(L.text("mobile.contentTools.tools.ask_questions.thinking"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .accessibilityLabel(L.text("mobile.contentTools.tools.ask_questions.thinking"))
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
                if askInstructor {
                    Text(L.text("mobile.contentTools.tools.ask_questions.askInstructor"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }

            if !props.readOnly {
                if consentAllowed {
                    ToolComposerView(
                        placeholder: ContentToolHostLogic.stringField(props.config, key: "placeholder")
                            ?? L.text("mobile.contentTools.tools.ask_questions.placeholder"),
                        sendLabel: L.text("mobile.contentTools.tools.ask_questions.ask"),
                        cancelLabel: L.text("mobile.contentTools.runtime.cancel"),
                        text: $draft,
                        draftKey: draftKey,
                        enabled: true,
                        online: online,
                        busy: busy,
                        showCancel: true,
                        onSend: { Task { await ask() } },
                        onCancel: {
                            inFlightTask?.cancel()
                            busy = false
                        }
                    )
                } else if consentFetched {
                    Text(L.text("mobile.contentTools.ai.consentRequired"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                if !turns.isEmpty {
                    Button(L.text("mobile.contentTools.tools.ask_questions.clear"), role: .destructive) {
                        showClearConfirm = true
                    }
                    .disabled(busy)
                    .confirmationDialog(
                        L.text("mobile.contentTools.tools.ask_questions.clearConfirm"),
                        isPresented: $showClearConfirm,
                        titleVisibility: .visible
                    ) {
                        Button(L.text("mobile.contentTools.tools.ask_questions.clear"), role: .destructive) {
                            Task { await clear() }
                        }
                        Button(L.text("mobile.contentTools.runtime.cancel"), role: .cancel) {}
                    }
                }
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
    private func turnRow(_ turn: (id: String, role: String, text: String, citations: [(title: String, url: String?)])) -> some View {
        let roleLabel = turn.role == "user"
            ? L.text("mobile.contentTools.tools.ask_questions.you")
            : L.text("mobile.contentTools.tools.ask_questions.aiBadge")
        VStack(alignment: .leading, spacing: 4) {
            Text(roleLabel)
                .font(.caption2.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            CourseMarkdownContentView(markdown: turn.text, compact: true)
            if ContentToolPack2Logic.boolField(props.config, key: "showCitations") != false,
               !turn.citations.isEmpty {
                Text(L.text("mobile.contentTools.tools.ask_questions.sources"))
                    .font(.caption2.weight(.semibold))
                ForEach(Array(turn.citations.enumerated()), id: \.offset) { idx, cite in
                    Text(L.format("mobile.contentTools.tools.ask_questions.sourceChip", idx + 1, cite.title))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(roleLabel). \(turn.text)")
    }

    private func loadConsent() async {
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

    private func ask() async {
        let question = draft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard ContentToolPack2Logic.canAsk(
            text: question,
            readOnly: props.readOnly,
            online: online,
            consentAllowed: consentAllowed,
            busy: busy
        ) else { return }
        busy = true
        errorText = nil
        askInstructor = false
        let task = Task {
            do {
                let raw = try await props.runAction("ask", .object(["question": .string(question)]))
                let result = ContentToolPack2Logic.objectMap(raw)
                if case .string(let code) = result["error"] {
                    errorText = L.text(ContentToolPack2Logic.plainLanguageMessageKey(for: code))
                    if case .bool(true) = result["askInstructor"] { askInstructor = true }
                    // Retain draft on failure (FR-2 / AC-3).
                    return
                }
                draft = ""
                ToolComposerView.clearDraft(key: draftKey)
                props.save(["draft": .string("")])
                let citeCount: Int = {
                    if case .number(let n) = result["citationCount"] { return Int(n) }
                    return 0
                }()
                props.announce(
                    L.format("mobile.contentTools.tools.ask_questions.answerReceived", citeCount),
                    false
                )
            } catch {
                if !online {
                    errorText = L.text("mobile.contentTools.runtime.offlineComposer")
                } else {
                    errorText = L.text("mobile.contentTools.runtime.retry")
                }
            }
        }
        inFlightTask = task
        await task.value
        busy = false
        inFlightTask = nil
    }

    private func clear() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("clear", .object([:]))
            draft = ""
            ToolComposerView.clearDraft(key: draftKey)
            props.announce(L.text("mobile.contentTools.tools.ask_questions.cleared"), false)
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
        }
    }
}
