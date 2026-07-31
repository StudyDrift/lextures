// swiftlint:disable identifier_name large_tuple
import SwiftUI

struct DiagramHotspotToolView: View {
    @Environment(\.colorScheme) private var colorScheme
    let props: ContentToolRendererProps

    @State private var selectedItemId: String?
    @State private var assignments: [String: String?] = [:]
    @State private var zoom: CGFloat = 1
    @State private var pan: CGSize = .zero
    @State private var lastCheck: ContentToolPack3Logic.CheckResultView?
    @State private var busy = false
    @State private var errorText: String?
    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !prompt.isEmpty {
                CourseMarkdownContentView(markdown: prompt, compact: true)
            }

            if !imageURL.isEmpty {
                ZoomableImageCanvas(
                    urlString: imageURL,
                    alt: imageAlt,
                    naturalWidth: naturalWidth,
                    naturalHeight: naturalHeight,
                    zoom: $zoom,
                    pan: $pan
                ) { size in
                    regionOverlay(size: size)
                }
                .frame(maxWidth: .infinity)
            } else {
                Text(imageAlt.isEmpty ? L.text("mobile.contentTools.tools.diagram_hotspot.imageMissing") : imageAlt)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            if canEdit {
                DragOrTapAssignBar(
                    selectedLabel: selectedItemLabel,
                    helperText: L.text("mobile.contentTools.tools.diagram_hotspot.listPickerHint")
                )
            }

            // List-based placement (FR-18) — always visible, not a11y-only.
            listPicker

            if let lastCheck {
                feedbackBlock(lastCheck)
            }

            // Text list of placements / results (FR-20)
            placementSummary

            if canEdit {
                HStack(spacing: 12) {
                    Button(L.text("mobile.contentTools.runtime.checkAnswer")) {
                        Task { await check() }
                    }
                    .disabled(busy || !allAssigned)
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
    }

    // MARK: - Config

    private var prompt: String {
        ContentToolHostLogic.stringField(props.config, key: "prompt") ?? ""
    }

    private var imageObj: [String: JSONValue] {
        ContentToolPack3Logic.objectMap(ContentToolPack3Logic.objectMap(props.config)["image"])
    }

    private var imageURL: String {
        if case .string(let u) = imageObj["url"] { return u }
        return ""
    }

    private var imageAlt: String {
        if case .string(let a) = imageObj["alt"] { return a }
        return ""
    }

    private var naturalWidth: Double {
        ContentToolPack3Logic.numberField(.object(imageObj), key: "naturalWidth") ?? 1
    }

    private var naturalHeight: Double {
        ContentToolPack3Logic.numberField(.object(imageObj), key: "naturalHeight") ?? 1
    }

    private var regions: [ContentToolPack3Logic.DiagramRegion] {
        ContentToolPack3Logic.arrayField(props.config, key: "regions").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"],
                  case .string(let label) = o["label"],
                  case .string(let description) = o["description"],
                  let shape = parseShape(o["shape"])
            else { return nil }
            return ContentToolPack3Logic.DiagramRegion(
                id: id, label: label, description: description, shape: shape
            )
        }
    }

    private var placeableItems: [(id: String, text: String)] {
        let labels = ContentToolPack3Logic.arrayField(props.config, key: "labels").compactMap { raw -> (String, String)? in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let text) = o["text"] else { return nil }
            return (id, text)
        }
        if !labels.isEmpty { return labels }
        return ContentToolPack3Logic.arrayField(props.config, key: "prompts").compactMap { raw in
            let o = ContentToolPack3Logic.objectMap(raw)
            guard case .string(let id) = o["id"], case .string(let text) = o["text"] else { return nil }
            return (id, text)
        }
    }

    private var lockedIds: [String] {
        ContentToolPack3Logic.arrayField(props.state, key: "lockedIds").compactMap {
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

    private var allAssigned: Bool {
        placeableItems.allSatisfy { item in
            if let region = assignments[item.id], let region, !region.isEmpty { return true }
            return false
        }
    }

    private var selectedItemLabel: String? {
        guard let id = selectedItemId else { return nil }
        return placeableItems.first(where: { $0.id == id })?.text
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

    // MARK: - UI

    @ViewBuilder
    private func regionOverlay(size: CGSize) -> some View {
        ForEach(regions, id: \.id) { region in
            Button {
                placeOnRegion(region.id)
            } label: {
                Circle()
                    .fill(Color.clear)
                    .frame(width: 44, height: 44)
                    .contentShape(Rectangle().inset(by: -4))
            }
            .buttonStyle(.plain)
            .position(regionCenter(region: region, in: size))
            .disabled(!canEdit || selectedItemId == nil)
            .accessibilityLabel(region.label)
            .accessibilityHint(region.description)
        }
    }

    @ViewBuilder
    private var listPicker: some View {
        Text(L.text("mobile.contentTools.tools.diagram_hotspot.labels"))
            .font(.caption.weight(.semibold))
        ForEach(placeableItems, id: \.id) { item in
            PlacementChip(
                title: item.text,
                selected: selectedItemId == item.id,
                locked: lockedIds.contains(item.id),
                correct: lastPerItem[item.id],
                disabled: !canEdit
            ) {
                if selectedItemId == item.id {
                    selectedItemId = nil
                } else {
                    selectedItemId = item.id
                    Haptics.trigger(.tap)
                }
            }
        }

        Text(L.text("mobile.contentTools.tools.diagram_hotspot.targets"))
            .font(.caption.weight(.semibold))
        ForEach(regions, id: \.id) { region in
            Button {
                placeOnRegion(region.id)
            } label: {
                HStack {
                    Image(systemName: "mappin.circle")
                        .foregroundStyle(LexturesTheme.primary)
                        .accessibilityHidden(true)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(region.label)
                            .font(.subheadline.weight(.semibold))
                        Text(region.description)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                }
                .frame(minHeight: 44)
            }
            .buttonStyle(.plain)
            .disabled(!canEdit || selectedItemId == nil || lockedIds.contains(selectedItemId ?? ""))
            .accessibilityHint(L.text("mobile.contentTools.interaction.tapToPlace"))
        }
    }

    @ViewBuilder
    private var placementSummary: some View {
        Text(L.text("mobile.contentTools.tools.diagram_hotspot.reviewPlacements"))
            .font(.caption.weight(.semibold))
        ForEach(placeableItems, id: \.id) { item in
            let regionId = assignments[item.id] ?? nil
            let regionLabel = regions.first(where: { $0.id == regionId })?.label
            let correct = lastPerItem[item.id]
            HStack(spacing: 6) {
                if let correct {
                    Image(systemName: correct ? "checkmark.circle.fill" : "xmark.circle.fill")
                        .foregroundStyle(correct ? LexturesTheme.primary : LexturesTheme.coral)
                        .accessibilityHidden(true)
                }
                Text(item.text)
                Text("→")
                Text(regionLabel ?? L.text("mobile.contentTools.tools.diagram_hotspot.unplaced"))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .font(.caption)
            .accessibilityLabel(
                L.format(
                    "mobile.contentTools.tools.diagram_hotspot.placementA11y",
                    item.text,
                    regionLabel ?? L.text("mobile.contentTools.tools.diagram_hotspot.unplaced")
                )
            )
        }
    }

    @ViewBuilder
    private func feedbackBlock(_ result: ContentToolPack3Logic.CheckResultView) -> some View {
        if let score = result.scorePct {
            Label(
                L.format("mobile.contentTools.tools.diagram_hotspot.scorePct", Int(score.rounded())),
                systemImage: score >= 100 ? "checkmark.circle.fill" : "info.circle"
            )
            .font(.caption.weight(.semibold))
        }
        if let err = result.error {
            Text(result.message ?? err.rawValue)
                .font(.caption)
                .foregroundStyle(LexturesTheme.coral)
        }
    }

    // MARK: - Actions

    private func hydrate() {
        let raw = ContentToolPack3Logic.objectMap(props.state)["assignments"]
        var map = ContentToolPack3Logic.parseCategorizePlacement(raw)
        for item in placeableItems where map[item.id] == nil {
            map[item.id] = nil
        }
        assignments = map
    }

    private func placeOnRegion(_ regionId: String) {
        guard canEdit, let itemId = selectedItemId, !lockedIds.contains(itemId) else { return }
        assignments[itemId] = regionId
        selectedItemId = nil
        Haptics.trigger(.success)
        persist(usedListMode: true)
        props.announce(L.text("mobile.contentTools.interaction.placed"), false)
    }

    private func persist(usedListMode: Bool) {
        guard !props.readOnly else { return }
        let patch = ContentToolPack3Logic.mergePreservingUnknown(
            base: ContentToolPack3Logic.objectMap(props.state),
            patch: [
                "v": .number(1),
                "assignments": ContentToolPack3Logic.categorizePlacementJSON(assignments),
                "usedListMode": .bool(usedListMode),
                "attempts": ContentToolPack3Logic.objectMap(props.state)["attempts"] ?? .array([]),
                "lockedIds": ContentToolPack3Logic.objectMap(props.state)["lockedIds"] ?? .array([]),
            ]
        )
        props.save(patch)
    }

    private func check() async {
        guard ContentToolPack3Logic.canCheck(
            attemptsUsed: attemptsUsed, maxAttempts: maxAttempts, readOnly: props.readOnly
        ) else {
            errorText = L.text("mobile.contentTools.tools.diagram_hotspot.error.maxAttempts")
            return
        }
        busy = true
        errorText = nil
        defer { busy = false }
        do {
            let result = try await props.runAction(
                "check",
                .object([
                    "assignments": ContentToolPack3Logic.categorizePlacementJSON(assignments),
                    "usedListMode": .bool(true),
                ])
            )
            let parsed = ContentToolPack3Logic.parseCheckResult(result)
            lastCheck = parsed
            if parsed.error != nil {
                Haptics.trigger(.error)
                errorText = parsed.message
            } else {
                Haptics.trigger((parsed.scorePct ?? 0) >= 100 ? .success : .selection)
                props.announce(L.text("mobile.contentTools.tools.diagram_hotspot.checked"), false)
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
            props.announce(L.text("mobile.contentTools.tools.diagram_hotspot.reset"), false)
            hydrate()
        } catch {
            errorText = L.text("mobile.contentTools.runtime.retry")
        }
    }

    private func regionCenter(region: ContentToolPack3Logic.DiagramRegion, in size: CGSize) -> CGPoint {
        let (nx, ny): (Double, Double)
        switch region.shape {
        case .rect(let x, let y, let w, let h):
            nx = x + w / 2; ny = y + h / 2
        case .circle(let cx, let cy, _):
            nx = cx; ny = cy
        case .polygon(let points):
            if points.isEmpty { return CGPoint(x: size.width / 2, y: size.height / 2) }
            nx = points.map(\.x).reduce(0, +) / Double(points.count)
            ny = points.map(\.y).reduce(0, +) / Double(points.count)
        }
        // object-fit: contain letterboxing
        let scale = min(size.width / CGFloat(naturalWidth), size.height / CGFloat(naturalHeight))
        let drawW = CGFloat(naturalWidth) * scale
        let drawH = CGFloat(naturalHeight) * scale
        let offsetX = (size.width - drawW) / 2
        let offsetY = (size.height - drawH) / 2
        return CGPoint(x: offsetX + CGFloat(nx) * drawW, y: offsetY + CGFloat(ny) * drawH)
    }

    private func parseShape(_ value: JSONValue?) -> ContentToolPack3Logic.RegionShape? {
        let o = ContentToolPack3Logic.objectMap(value)
        guard case .string(let kind) = o["kind"] else { return nil }
        switch kind {
        case "rect":
            guard let x = ContentToolPack3Logic.numberField(value, key: "x"),
                  let y = ContentToolPack3Logic.numberField(value, key: "y"),
                  let w = ContentToolPack3Logic.numberField(value, key: "w"),
                  let h = ContentToolPack3Logic.numberField(value, key: "h")
            else { return nil }
            return .rect(x: x, y: y, w: w, h: h)
        case "circle":
            guard let cx = ContentToolPack3Logic.numberField(value, key: "cx"),
                  let cy = ContentToolPack3Logic.numberField(value, key: "cy"),
                  let r = ContentToolPack3Logic.numberField(value, key: "r")
            else { return nil }
            return .circle(cx: cx, cy: cy, r: r)
        case "polygon":
            let points: [ContentToolPack3Logic.NormPoint] = ContentToolPack3Logic.arrayField(value, key: "points").compactMap { pt in
                guard case .array(let arr) = pt, arr.count >= 2,
                      case .number(let x) = arr[0], case .number(let y) = arr[1]
                else { return nil }
                return ContentToolPack3Logic.NormPoint(x: x, y: y)
            }
            return .polygon(points: points)
        default:
            return nil
        }
    }
}
