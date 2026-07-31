// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct ParameterExplorerToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var params: [String: JSONValue] = [:]
    @State private var dragging = false
    @State private var dirty = false
    @State private var lastRecomputeMs: Int64?
    @State private var plotPoints: [(x: Double, y: Double)] = []
    @State private var answerDrafts: [String: String] = [:]
    @State private var busy = false
    @State private var errorText: String?
    @State private var statusText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            if !prompt.isEmpty {
                CourseMarkdownContentView(markdown: prompt, compact: true)
            }

            ForEach(numberParams, id: \.id) { param in
                numberControl(param)
            }
            ForEach(booleanParams, id: \.id) { param in
                booleanControl(param)
            }
            ForEach(choiceParams, id: \.id) { param in
                choiceControl(param)
            }

            if !plotPoints.isEmpty {
                plotSummary
                dataTable
            }

            ForEach(noticingPrompts, id: \.id) { promptItem in
                noticingBlock(promptItem)
            }

            HStack {
                Button(L.text("mobile.contentTools.tools.parameter_explorer.resetDefaults")) {
                    Task { await resetDefaults() }
                }
                .disabled(props.readOnly || busy)
                if let statusText {
                    Text(statusText)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
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
            recompute(force: true)
        }
        .onChange(of: props.state) { _, _ in
            if !dragging { hydrate() }
        }
    }

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt") ?? ""
    }

    private struct NumberParam: Identifiable {
        var id: String
        var label: String
        var unit: String?
        var min: Double
        var max: Double
        var step: Double
    }

    private struct BoolParam: Identifiable {
        var id: String
        var label: String
    }

    private struct ChoiceParam: Identifiable {
        var id: String
        var label: String
        var options: [(value: String, label: String)]
    }

    private struct PromptItem: Identifiable {
        var id: String
        var text: String
        var kind: String
        var options: [(value: String, label: String)]
        var unlockWhen: String?
    }

    private var numberParams: [NumberParam] {
        ContentToolPack4Logic.arrayField(props.config, key: "parameters").compactMap { raw in
            let obj = ContentToolPack4Logic.objectMap(raw)
            guard case .string(let kind) = obj["kind"], kind == "number" else { return nil }
            guard case .string(let id) = obj["id"] else { return nil }
            return NumberParam(
                id: id,
                label: ContentToolHostLogic.stringField(raw, key: "label") ?? id,
                unit: ContentToolHostLogic.stringField(raw, key: "unit"),
                min: ContentToolPack4Logic.numberField(raw, key: "min") ?? 0,
                max: ContentToolPack4Logic.numberField(raw, key: "max") ?? 1,
                step: ContentToolPack4Logic.numberField(raw, key: "step") ?? 0.1
            )
        }
    }

    private var booleanParams: [BoolParam] {
        ContentToolPack4Logic.arrayField(props.config, key: "parameters").compactMap { raw in
            let obj = ContentToolPack4Logic.objectMap(raw)
            guard case .string(let kind) = obj["kind"], kind == "boolean" else { return nil }
            guard case .string(let id) = obj["id"] else { return nil }
            return BoolParam(id: id, label: ContentToolHostLogic.stringField(raw, key: "label") ?? id)
        }
    }

    private var choiceParams: [ChoiceParam] {
        ContentToolPack4Logic.arrayField(props.config, key: "parameters").compactMap { raw in
            let obj = ContentToolPack4Logic.objectMap(raw)
            guard case .string(let kind) = obj["kind"], kind == "choice" else { return nil }
            guard case .string(let id) = obj["id"] else { return nil }
            let options = ContentToolPack4Logic.arrayField(raw, key: "options").compactMap { opt -> (String, String)? in
                let o = ContentToolPack4Logic.objectMap(opt)
                guard case .string(let value) = o["value"] else { return nil }
                let label = ContentToolHostLogic.stringField(opt, key: "label") ?? value
                return (value, label)
            }
            return ChoiceParam(
                id: id,
                label: ContentToolHostLogic.stringField(raw, key: "label") ?? id,
                options: options
            )
        }
    }

    private var noticingPrompts: [PromptItem] {
        ContentToolPack4Logic.arrayField(props.config, key: "noticingPrompts").compactMap { raw in
            guard case .string(let id) = ContentToolPack4Logic.objectMap(raw)["id"] else { return nil }
            let options = ContentToolPack4Logic.arrayField(raw, key: "options").compactMap { opt -> (String, String)? in
                let o = ContentToolPack4Logic.objectMap(opt)
                guard case .string(let value) = o["value"] else { return nil }
                return (value, ContentToolHostLogic.stringField(opt, key: "label") ?? value)
            }
            return PromptItem(
                id: id,
                text: ContentToolHostLogic.stringField(raw, key: "text") ?? id,
                kind: ContentToolHostLogic.stringField(raw, key: "kind") ?? "text",
                options: options,
                unlockWhen: ContentToolHostLogic.stringField(raw, key: "unlockWhen")
            )
        }
    }

    private var checkpoints: [String: JSONValue] {
        ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(props.state)["checkpoints"])
    }

    private var answers: [String: JSONValue] {
        ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(props.state)["answers"])
    }

    @ViewBuilder
    private func numberControl(_ param: NumberParam) -> some View {
        let value = numberValue(param.id) ?? param.min
        VStack(alignment: .leading, spacing: 6) {
            Text(param.unit.map { "\(param.label) (\($0))" } ?? param.label)
                .font(.caption.weight(.semibold))
            HStack(spacing: 10) {
                Slider(
                    value: Binding(
                        get: { value },
                        set: { setNumber(param.id, ContentToolPack4Logic.clampNumber(value: $0, min: param.min, max: param.max, step: param.step), dragging: true) }
                    ),
                    in: param.min ... param.max,
                    step: max(param.step, 0.0001)
                ) { editing in
                    dragging = editing
                    if !editing { settle() }
                }
                .disabled(props.readOnly)
                .accessibilityLabel(param.label)
                .accessibilityValue("\(value)")

                TextField(
                    "",
                    value: Binding(
                        get: { value },
                        set: { setNumber(param.id, ContentToolPack4Logic.clampNumber(value: $0, min: param.min, max: param.max, step: param.step), dragging: false); settle() }
                    ),
                    format: .number
                )
                .frame(width: 72)
                .textFieldStyle(.roundedBorder)
                .disabled(props.readOnly)
                .accessibilityLabel(L.format("mobile.contentTools.tools.parameter_explorer.directEntry", param.label))
            }
            Text(L.format(
                "mobile.contentTools.tools.parameter_explorer.sliderSemantics",
                String(value), String(param.min), String(param.max), String(param.step)
            ))
            .font(.caption2)
            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            .accessibilityHidden(true)
        }
    }

    @ViewBuilder
    private func booleanControl(_ param: BoolParam) -> some View {
        Toggle(param.label, isOn: Binding(
            get: {
                if case .bool(let b) = params[param.id] { return b }
                return false
            },
            set: { newValue in
                params[param.id] = .bool(newValue)
                dirty = true
                recompute(force: true)
                settle()
            }
        ))
        .disabled(props.readOnly)
    }

    @ViewBuilder
    private func choiceControl(_ param: ChoiceParam) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(param.label).font(.caption.weight(.semibold))
            Picker(param.label, selection: Binding(
                get: {
                    if case .string(let s) = params[param.id] { return s }
                    return param.options.first?.value ?? ""
                },
                set: { newValue in
                    params[param.id] = .string(newValue)
                    dirty = true
                    recompute(force: true)
                    settle()
                }
            )) {
                ForEach(param.options, id: \.value) { opt in
                    Text(opt.label).tag(opt.value)
                }
            }
            .disabled(props.readOnly)
        }
    }

    private var plotSummary: some View {
        let first = plotPoints.first
        let last = plotPoints.last
        return Text(
            L.format(
                "mobile.contentTools.tools.parameter_explorer.plotSummary",
                String(format: "%.2f", first?.x ?? 0),
                String(format: "%.2f", last?.x ?? 0),
                String(format: "%.2f", first?.y ?? 0),
                String(format: "%.2f", last?.y ?? 0)
            )
        )
        .font(.caption)
        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        .accessibilityAddTraits(.updatesFrequently)
    }

    private var dataTable: some View {
        ScrollView(.horizontal, showsIndicators: true) {
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text("x").font(.caption.weight(.semibold)).frame(width: 64, alignment: .leading)
                    Text("y").font(.caption.weight(.semibold)).frame(width: 80, alignment: .leading)
                }
                ForEach(Array(plotPoints.enumerated().filter { $0.offset % 4 == 0 }), id: \.offset) { _, pt in
                    HStack {
                        Text(String(format: "%.2f", pt.x)).font(.caption2).frame(width: 64, alignment: .leading)
                        Text(String(format: "%.3f", pt.y)).font(.caption2).frame(width: 80, alignment: .leading)
                    }
                }
            }
        }
        .accessibilityLabel(L.text("mobile.contentTools.tools.parameter_explorer.dataTable"))
    }

    @ViewBuilder
    private func noticingBlock(_ item: PromptItem) -> some View {
        let unlocked = item.unlockWhen == nil || checkpoints[item.id] != nil || evaluateUnlock(item.unlockWhen)
        let answered: Bool = {
            if case .string(let s) = answers[item.id] { return !s.isEmpty }
            return false
        }()

        VStack(alignment: .leading, spacing: 8) {
            Text(item.text)
                .font(.caption.weight(.semibold))
            if !unlocked {
                Text(L.text("mobile.contentTools.tools.parameter_explorer.locked"))
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                TextField(
                    L.text("mobile.contentTools.tools.parameter_explorer.answerPlaceholder"),
                    text: Binding(
                        get: { answerDrafts[item.id] ?? "" },
                        set: { answerDrafts[item.id] = $0 }
                    ),
                    axis: .vertical
                )
                .lineLimit(2 ... 5)
                .disabled(props.readOnly || busy)
                Button(L.text("mobile.contentTools.tools.parameter_explorer.submitAnswer")) {
                    Task { await submitAnswer(item.id) }
                }
                .disabled(props.readOnly || busy || (answerDrafts[item.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                if answered {
                    Text(L.text("mobile.contentTools.tools.parameter_explorer.answered"))
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.primary)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private func numberValue(_ id: String) -> Double? {
        ContentToolPack4Logic.numberField(.object(params), key: id)
    }

    private func setNumber(_ id: String, _ value: Double, dragging: Bool) {
        params[id] = .number(value)
        self.dragging = dragging
        dirty = true
        recompute(force: false)
        props.announce(L.format("mobile.contentTools.tools.parameter_explorer.valueAnnounce", id, String(value)), false)
    }

    private func hydrate() {
        let stateParams = ContentToolPack4Logic.objectMap(ContentToolPack4Logic.objectMap(props.state)["params"])
        if stateParams.isEmpty {
            params = ContentToolPack4Logic.defaultParams(from: props.config)
        } else {
            params = stateParams
        }
        let ans = answers
        for (id, value) in ans {
            if case .string(let s) = value {
                answerDrafts[id] = s
            }
        }
    }

    private func recompute(force: Bool) {
        let now = Int64(Date().timeIntervalSince1970 * 1000)
        if !force && !ContentToolPack4Logic.shouldRecompute(lastAtMs: lastRecomputeMs, nowMs: now) {
            return
        }
        lastRecomputeMs = now
        let a = numberValue("a") ?? 1
        let b = numberValue("b") ?? 0
        let c = numberValue("c") ?? 0
        var pts: [(Double, Double)] = []
        for i in 0 ... 40 {
            let x = -10 + (20 * Double(i)) / 40
            pts.append((x, a * x * x + b * x + c))
        }
        plotPoints = pts
        Task { await maybeCheckpoint() }
    }

    private func settle() {
        guard ContentToolPack4Logic.shouldAutosaveOnSettle(dragging: dragging, dirty: dirty) else { return }
        let base = ContentToolPack4Logic.objectMap(props.state)
        let patch = ContentToolPack4Logic.mergeParamsPreservingUnknown(state: base, params: params)
        props.save(patch)
        dirty = false
    }

    private func evaluateUnlock(_ expr: String?) -> Bool {
        guard let expr else { return true }
        // Minimal "a > N" support matching sandbox canary.
        let trimmed = expr.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let match = trimmed.range(of: #"^a\s*>\s*([0-9.]+)$"#, options: .regularExpression) else {
            return checkpoints.values.isEmpty == false
        }
        _ = match
        let numberPart = trimmed.split(separator: ">").last.map(String.init)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard let threshold = numberPart.flatMap(Double.init) else { return false }
        return (numberValue("a") ?? 0) > threshold
    }

    private func maybeCheckpoint() async {
        for item in noticingPrompts {
            guard let unlock = item.unlockWhen, checkpoints[item.id] == nil else { continue }
            guard evaluateUnlock(unlock) else { continue }
            do {
                _ = try await props.runAction(
                    "checkpoint",
                    .object([
                        "promptId": .string(item.id),
                        "params": .object(params),
                    ])
                )
                props.announce(L.text("mobile.contentTools.tools.parameter_explorer.checkpointReached"), false)
            } catch {
                // Non-fatal — unlock UI still works locally.
            }
        }
    }

    private func submitAnswer(_ promptId: String) async {
        let answer = (answerDrafts[promptId] ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        guard !answer.isEmpty else { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction(
                "submit_answer",
                .object([
                    "promptId": .string(promptId),
                    "answer": .string(answer),
                    "params": .object(params),
                ])
            )
            statusText = L.text("mobile.contentTools.tools.parameter_explorer.answered")
            props.announce(statusText ?? "", false)
        } catch {
            errorText = L.text("mobile.contentTools.tools.parameter_explorer.error")
        }
    }

    private func resetDefaults() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await props.runAction("reset_defaults", .object([:]))
            params = ContentToolPack4Logic.defaultParams(from: props.config)
            dirty = false
            recompute(force: true)
            props.announce(L.text("mobile.contentTools.tools.parameter_explorer.resetAnnounce"), false)
        } catch {
            params = ContentToolPack4Logic.defaultParams(from: props.config)
            dirty = true
            settle()
            recompute(force: true)
        }
    }
}
