// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct SortSequenceToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var engine = ContentToolPack3Logic.createInitialEngineState(
        mode: .categorize,
        itemIds: []
    )
    @State private var settledPlacement: ContentToolPack3Logic.Placement = .categorize([:])
    @State private var lastCheck: ContentToolPack3Logic.CheckResultView?
    @State private var busy = false
    @State private var errorText: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !prompt.isEmpty {
                CourseMarkdownContentView(markdown: prompt, compact: true)
            }

            if !attemptsExhausted && !props.readOnly {
                DragOrTapAssignBar(
                    selectedLabel: selectedItemLabel,
                    helperText: L.text(
                        mode == .order
                            ? "mobile.contentTools.tools.sort_sequence.tapOrderHint"
                            : "mobile.contentTools.tools.sort_sequence.tapCategorizeHint"
                    )
                )
            }

            if mode == .categorize {
                categorizeBoard
            } else {
                orderBoard
            }

            if let lastCheck {
                feedbackBlock(lastCheck)
            }

            if canEdit {
                HStack(spacing: 12) {
                    Button(L.text("mobile.contentTools.runtime.checkAnswer")) {
                        Task { await check() }
                    }
                    .disabled(busy || !allPlaced)
                    Button(L.text("mobile.contentTools.runtime.reset")) {
                        Task { await resetAttempt() }
                    }
                    .disabled(busy)
                }
            }

            if let errorText {
                Text(errorText)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .onAppear { hydrate() }
        .onChange(of: props.state) { _, _ in hydrate() }
        .onChange(of: props.config) { _, _ in hydrate() }
    }

    // MARK: - Config / state

    private var mode: ContentToolPack3Logic.PlacementMode {
        ContentToolHostLogic.stringField(props.config, key: "mode") == "order" ? .order : .categorize
    }

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt") ?? ""
    }

    private var items: [(id: String, text: String)] {
        ContentToolPack3Logic.arrayField(props.config, key: "items").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let text) = o["text"] else { return nil }
            return (id, text)
        }
    }

    private var buckets: [(id: String, label: String)] {
        ContentToolPack3Logic.arrayField(props.config, key: "buckets").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let label) = o["label"] else { return nil }
            return (id, label)
        }
    }

    private var itemIds: [String] { items.map(\.id) }

    private var lockedItemIds: [String] {
        ContentToolPack3Logic.arrayField(props.state, key: "lockedItemIds").compactMap {
            if case .string(let s) = $0 { return s }
            return nil
        }
    }

    private var attemptsUsed: Int {
        ContentToolPack3Logic.arrayField(props.state, key: "attempts").count
    }

    private var maxAttempts: Int? {
        ContentToolPack3Logic.parseAttemptsConfig(
            ContentToolPack3Logic.objectMap(props.config)["attempts"]
        )
    }

    private var attemptsExhausted: Bool {
        guard let maxAttempts else { return false }
        return attemptsUsed >= maxAttempts
    }

    private var canEdit: Bool { !props.readOnly && !attemptsExhausted }

    private var allPlaced: Bool {
        ContentToolPack3Logic.allPlaced(mode: mode, itemIds: itemIds, placement: engine.placement)
    }

    private var selectedItemLabel: String? {
        guard let id = engine.grabbedId else { return nil }
        return items.first(where: { $0.id == id })?.text
    }

    private var lastPerItem: [String: Bool] {
        lastCheck?.perItem ?? {
            var out: [String: Bool] = [:]
            let map = ContentToolPack3Logic.objectMap(
                ContentToolPack3Logic.objectMap(props.state)["lastPerItem"]
            )
            for (k, v) in map {
                if case .bool(let flag) = v { out[k] = flag }
            }
            return out
        }()
    }

    // MARK: - Boards

    @ViewBuilder
    private var categorizeBoard: some View {
        let tray = ContentToolPack3Logic.trayItemIds(
            mode: .categorize, itemIds: itemIds, placement: engine.placement
        )
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.contentTools.tools.sort_sequence.tray"))
                .font(.caption.weight(.semibold))
            ForEach(tray, id: \.self) { id in
                itemChip(id: id, correct: lastPerItem[id])
            }
            if tray.isEmpty {
                Text(L.text("mobile.contentTools.tools.sort_sequence.trayEmpty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            ForEach(buckets, id: \.id) { bucket in
                VStack(alignment: .leading, spacing: 6) {
                    Button {
                        tap(.bucket(bucket.id))
                    } label: {
                        Text(bucket.label)
                            .font(.subheadline.weight(.semibold))
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .frame(minHeight: 44)
                    }
                    .buttonStyle(.plain)
                    .disabled(!canEdit)
                    .accessibilityHint(L.text("mobile.contentTools.interaction.tapToPlace"))

                    let inBucket = ContentToolPack3Logic.itemsInBucket(
                        placement: engine.placement, bucketId: bucket.id
                    )
                    ForEach(inBucket, id: \.self) { id in
                        itemChip(id: id, correct: lastPerItem[id])
                    }
                }
                .padding(8)
                .background(
                    RoundedRectangle(cornerRadius: 8)
                        .strokeBorder(LexturesTheme.fieldBorder(for: colorScheme))
                )
            }
        }
    }

    @ViewBuilder
    private var orderBoard: some View {
        let tray = ContentToolPack3Logic.trayItemIds(
            mode: .order, itemIds: itemIds, placement: engine.placement
        )
        let order = engine.placement.asOrder ?? []
        VStack(alignment: .leading, spacing: 10) {
            Text(L.text("mobile.contentTools.tools.sort_sequence.tray"))
                .font(.caption.weight(.semibold))
            ForEach(tray, id: \.self) { id in
                itemChip(id: id, correct: lastPerItem[id])
            }

            Text(L.text("mobile.contentTools.tools.sort_sequence.sequence"))
                .font(.caption.weight(.semibold))
            ForEach(Array(order.enumerated()), id: \.element) { index, id in
                HStack(spacing: 8) {
                    Button {
                        tap(.position(index))
                    } label: {
                        Text("\(index + 1).")
                            .font(.caption.monospacedDigit())
                            .frame(width: 28, alignment: .trailing)
                    }
                    .buttonStyle(.plain)
                    .disabled(!canEdit)
                    itemChip(id: id, correct: lastPerItem[id])
                    if canEdit && !lockedItemIds.contains(id) {
                        VStack(spacing: 2) {
                            Button {
                                move(id, direction: -1)
                            } label: {
                                Image(systemName: "chevron.up")
                                    .frame(width: 44, height: 32)
                            }
                            .accessibilityLabel(L.text("mobile.contentTools.interaction.moveUp"))
                            Button {
                                move(id, direction: 1)
                            } label: {
                                Image(systemName: "chevron.down")
                                    .frame(width: 44, height: 32)
                            }
                            .accessibilityLabel(L.text("mobile.contentTools.interaction.moveDown"))
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            if canEdit {
                Button {
                    tap(.position(order.count))
                } label: {
                    Text(L.text("mobile.contentTools.tools.sort_sequence.insertEnd"))
                        .font(.caption)
                        .frame(maxWidth: .infinity, minHeight: 44)
                }
                .buttonStyle(.plain)
                .disabled(engine.grabbedId == nil)
            }
        }
    }

    private func itemChip(id: String, correct: Bool?) -> some View {
        let text = items.first(where: { $0.id == id })?.text ?? id
        return PlacementChip(
            title: text,
            selected: engine.grabbedId == id,
            locked: lockedItemIds.contains(id),
            correct: correct,
            disabled: !canEdit
        ) {
            tap(.item(id))
        }
        .accessibilityHint(
            engine.grabbedId == nil
                ? L.text("mobile.contentTools.interaction.tapToSelect")
                : L.text("mobile.contentTools.interaction.tapToPlace")
        )
    }

    @ViewBuilder
    private func feedbackBlock(_ result: ContentToolPack3Logic.CheckResultView) -> some View {
        if let score = result.scorePct {
            Label(
                L.format("mobile.contentTools.tools.sort_sequence.scorePct", Int(score.rounded())),
                systemImage: score >= 100 ? "checkmark.circle.fill" : "info.circle"
            )
            .font(.caption.weight(.semibold))
            .foregroundStyle(score >= 100 ? LexturesTheme.primary : LexturesTheme.textSecondary(for: colorScheme))
        }
        if let err = result.error {
            Text(result.message ?? err.rawValue)
                .font(.caption)
                .foregroundStyle(LexturesTheme.coral)
        }
    }

    // MARK: - Actions

    private func hydrate() {
        let placement: ContentToolPack3Logic.Placement
        let raw = ContentToolPack3Logic.objectMap(props.state)["placement"]
        if mode == .order {
            placement = .order(ContentToolPack3Logic.parseOrderPlacement(raw))
        } else {
            var map = ContentToolPack3Logic.parseCategorizePlacement(raw)
            if map.isEmpty {
                for id in itemIds { map[id] = nil }
            }
            placement = .categorize(map)
        }
        engine = ContentToolPack3Logic.createInitialEngineState(
            mode: mode, itemIds: itemIds, existing: placement
        )
        settledPlacement = placement
    }

    private func tap(_ hit: ContentToolPack3Logic.PlacementHit) {
        guard canEdit else { return }
        let next = ContentToolPack3Logic.tapItemOrTarget(
            engine, mode: mode, lockedItemIds: lockedItemIds, hit: hit
        )
        if next.grabbedId != engine.grabbedId {
            Haptics.trigger(next.grabbedId == nil ? .selection : .tap)
        }
        engine = next
        if next.grabbedId == nil {
            settledPlacement = next.placement
            persist()
            announcePlacement()
        }
    }

    private func move(_ itemId: String, direction: Int) {
        guard canEdit, mode == .order else { return }
        let order = engine.placement.asOrder ?? []
        let next = ContentToolPack3Logic.moveInOrder(
            order: order, itemId: itemId, direction: direction, lockedItemIds: lockedItemIds
        )
        guard next != order else { return }
        engine.placement = .order(next)
        settledPlacement = engine.placement
        Haptics.trigger(.selection)
        persist()
        props.announce(L.text("mobile.contentTools.interaction.moved"), false)
    }

    private func announcePlacement() {
        props.announce(L.text("mobile.contentTools.interaction.placed"), false)
    }

    private func persist() {
        guard !props.readOnly else { return }
        let placementJSON: JSONValue = mode == .order
            ? ContentToolPack3Logic.orderPlacementJSON(engine.placement.asOrder ?? [])
            : ContentToolPack3Logic.categorizePlacementJSON(engine.placement.asCategorize ?? [:])
        let patch = ContentToolPack3Logic.mergePreservingUnknown(
            base: ContentToolPack3Logic.objectMap(props.state),
            patch: [
                "v": .number(1),
                "placement": placementJSON,
                "attempts": ContentToolPack3Logic.objectMap(props.state)["attempts"] ?? .array([]),
                "lockedItemIds": ContentToolPack3Logic.objectMap(props.state)["lockedItemIds"] ?? .array([]),
            ]
        )
        props.save(patch)
    }

    private func check() async {
        guard ContentToolPack3Logic.canCheck(
            attemptsUsed: attemptsUsed, maxAttempts: maxAttempts, readOnly: props.readOnly
        ) else {
            errorText = L.text("mobile.contentTools.tools.sort_sequence.error.maxAttempts")
            return
        }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let placementJSON: JSONValue = mode == .order
                ? ContentToolPack3Logic.orderPlacementJSON(engine.placement.asOrder ?? [])
                : ContentToolPack3Logic.categorizePlacementJSON(engine.placement.asCategorize ?? [:])
            let result = try await props.runAction(
                "check",
                .object(["placement": placementJSON])
            )
            let parsed = ContentToolPack3Logic.parseCheckResult(result)
            lastCheck = parsed
            if parsed.error == .maxAttempts {
                errorText = parsed.message
                    ?? L.text("mobile.contentTools.tools.sort_sequence.error.maxAttempts")
                Haptics.trigger(.error)
            } else {
                let perfect = (parsed.scorePct ?? 0) >= 100
                Haptics.trigger(perfect ? .success : .selection)
                props.announce(
                    perfect
                        ? L.text("mobile.contentTools.tools.sort_sequence.allCorrect")
                        : L.text("mobile.contentTools.tools.sort_sequence.checked"),
                    false
                )
            }
            hydrate()
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
            Haptics.trigger(.error)
        }
    }

    private func resetAttempt() async {
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            _ = try await props.runAction("reset_attempt", .object([:]))
            lastCheck = nil
            engine = ContentToolPack3Logic.restoreAfterDragInterrupt(
                settled: ContentToolPack3Logic.createInitialEngineState(mode: mode, itemIds: itemIds)
            )
            props.announce(L.text("mobile.contentTools.tools.sort_sequence.reset"), false)
            hydrate()
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
        }
    }
}
