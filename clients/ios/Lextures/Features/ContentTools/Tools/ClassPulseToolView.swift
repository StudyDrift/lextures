// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct ClassPulseToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.scenePhase) private var scenePhase
    let props: ContentToolRendererProps

    @State private var selectedOptionId = ""
    @State private var busy = false
    @State private var aggregate: Aggregate?
    @State private var revealText: String?
    @State private var errorText: String?
    @State private var pollTask: Task<Void, Never>?
    @State private var consecutiveFailures = 0

    private struct Aggregate {
        var suppressed: Bool
        var learners: Int
        var options: [(optionId: String, count: Int, percent: Int?)]
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            CourseMarkdownContentView(markdown: question, compact: true)

            if !hasVoted {
                Text(L.text("mobile.contentTools.tools.class_pulse.chooseOption"))
                    .font(.caption.weight(.semibold))
                ForEach(options, id: \.id) { opt in
                    Button {
                        guard !props.readOnly else { return }
                        selectedOptionId = opt.id
                        persistDraft()
                    } label: {
                        HStack(alignment: .top, spacing: 8) {
                            Image(systemName: selectedOptionId == opt.id ? "largecircle.fill.circle" : "circle")
                            CourseMarkdownContentView(markdown: opt.text, compact: true)
                        }
                    }
                    .disabled(props.readOnly || busy)
                    .accessibilityLabel(opt.text)
                    .accessibilityAddTraits(selectedOptionId == opt.id ? .isSelected : [])
                }
                Button(L.text("mobile.contentTools.tools.class_pulse.submitVote")) {
                    Task { await vote() }
                }
                .disabled(props.readOnly || busy || selectedOptionId.isEmpty)
            } else {
                if let aggregate {
                    distribution(aggregate)
                } else {
                    Text(L.text("mobile.contentTools.tools.class_pulse.waitingForMore"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if let revealText {
                    Text(revealText)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear {
            hydrate()
            if hasVoted { startPolling() }
        }
        .onDisappear { stopPolling() }
        .onChange(of: scenePhase) { _, phase in
            if phase == .active, hasVoted {
                startPolling()
            } else if phase != .active {
                stopPolling()
            }
        }
        .onChange(of: props.state) { _, _ in hydrate() }
    }

    private var question: String {
        ContentToolHostLogic.stringField(props.config, key: "question") ?? ""
    }

    private var options: [(id: String, text: String)] {
        ContentToolPack1Logic.arrayField(props.config, key: "options").compactMap { raw in
            let o = ContentToolPack1Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let text) = o["text"] else { return nil }
            return (id, text)
        }
    }

    private var showPercentages: Bool {
        ContentToolPack1Logic.boolField(props.config, key: "showPercentages") != false
    }

    private var votes: [JSONValue] {
        ContentToolPack1Logic.arrayField(props.state, key: "votes")
    }

    private var hasVoted: Bool {
        ContentToolPack1Logic.hasVoted(votes: votes, round: 1)
    }

    @ViewBuilder
    private func distribution(_ agg: Aggregate) -> some View {
        Text(L.text("mobile.contentTools.tools.class_pulse.results"))
            .font(.subheadline.weight(.semibold))
        if agg.suppressed {
            Text(L.text("mobile.contentTools.tools.class_pulse.suppressed"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            Text(L.format("mobile.contentTools.tools.class_pulse.respondents", agg.learners))
                .font(.caption2)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(agg.options, id: \.optionId) { row in
                let label = options.first(where: { $0.id == row.optionId })?.text ?? row.optionId
                let pct = row.percent.map { "\($0)%" } ?? ""
                let yours = votes.contains {
                    guard case .object(let o) = $0, case .string(let id) = o["optionId"] else { return false }
                    return id == row.optionId
                }
                let text: String = {
                    if showPercentages, !pct.isEmpty {
                        return "\(label): \(pct)" + (yours ? " (\(L.text("mobile.contentTools.tools.class_pulse.yourAnswer")))" : "")
                    }
                    return "\(label): \(row.count)" + (yours ? " (\(L.text("mobile.contentTools.tools.class_pulse.yourAnswer")))" : "")
                }()
                Text(text)
                    .font(.caption)
                    .accessibilityLabel(text)
            }
        }
    }

    private func hydrate() {
        let draft = ContentToolPack1Logic.objectMap(ContentToolPack1Logic.objectMap(props.state)["draft"])
        if case .string(let id) = draft["optionId"] { selectedOptionId = id }
        if hasVoted, aggregate == nil {
            Task { await fetchAggregate() }
        }
    }

    private func persistDraft() {
        guard !props.readOnly else { return }
        let patch = ContentToolPack1Logic.mergePreservingUnknown(
            base: ContentToolPack1Logic.objectMap(props.state),
            patch: [
                "v": .number(1),
                "draft": .object(["optionId": .string(selectedOptionId), "round": .number(1)]),
            ]
        )
        props.save(patch)
    }

    private func vote() async {
        if busy || props.readOnly { return }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let raw = try await props.runAction(
                "vote",
                .object(["optionId": .string(selectedOptionId), "round": .number(1)])
            )
            applyAggregate(raw)
            Haptics.trigger(.success)
            props.announce(L.text("mobile.contentTools.tools.class_pulse.resultsAnnounce"), false)
            startPolling()
        } catch {
            errorText = L.text("mobile.contentTools.runtime.needsConnection")
            props.announce(errorText!, true)
        }
    }

    private func fetchAggregate() async {
        do {
            let raw = try await props.runAction("aggregate", .object([:]))
            applyAggregate(raw)
            consecutiveFailures = 0
        } catch {
            consecutiveFailures += 1
        }
    }

    private func applyAggregate(_ raw: JSONValue?) {
        guard case .object(let obj) = raw else { return }
        let source = obj["aggregate"] ?? raw
        guard case .object(let agg) = source else { return }
        let suppressed = {
            if case .bool(let b) = agg["suppressed"] { return b }
            return false
        }()
        let learners = ContentToolPack1Logic.numberField(.object(agg), key: "learners").map { Int($0) } ?? 0
        var rows: [(String, Int, Int?)] = []
        if case .array(let opts) = agg["options"] {
            for opt in opts {
                let o = ContentToolPack1Logic.objectMap(opt)
                guard case .string(let id) = o["optionId"] else { continue }
                let count = ContentToolPack1Logic.numberField(opt, key: "count").map { Int($0) } ?? 0
                let percent = ContentToolPack1Logic.numberField(opt, key: "percent").map { Int($0.rounded()) }
                rows.append((id, count, percent))
            }
        }
        aggregate = Aggregate(suppressed: suppressed, learners: learners, options: rows)
        if case .object(let reveal) = obj["reveal"],
           case .string(let correctId) = reveal["correctOptionId"] {
            let label = options.first(where: { $0.id == correctId })?.text ?? correctId
            revealText = L.format("mobile.contentTools.tools.class_pulse.correctAnswer", label)
        }
    }

    private func startPolling() {
        stopPolling()
        guard ContentToolPack1Logic.shouldPollAggregate(
            visible: scenePhase == .active,
            hasVoted: hasVoted
        ) else { return }
        pollTask = Task {
            while !Task.isCancelled {
                let delay = ContentToolPack1Logic.nextPollDelayMs(consecutiveFailures: consecutiveFailures)
                try? await Task.sleep(nanoseconds: UInt64(delay) * 1_000_000)
                if Task.isCancelled { break }
                if scenePhase != .active { continue }
                await fetchAggregate()
            }
        }
    }

    private func stopPolling() {
        pollTask?.cancel()
        pollTask = nil
    }
}
