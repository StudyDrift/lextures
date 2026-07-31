// swiftlint:disable identifier_name large_tuple force_try type_body_length file_length
import Foundation

/// Pure CT.M7 pack-3 decisions — placement engine, quote anchors, normalised
/// geometry, hit-target expansion, attempt gating, and client allowlist. No networking.
enum ContentToolPack3Logic {
    static let pack3ToolIds: Set<String> = [
        "sort_sequence",
        "highlight_annotate",
        "diagram_hotspot",
    ]

    /// Per-tool client allowlist (rollout). Empty entry removes a renderer without a release.
    static var clientAllowlist: Set<String> = pack3ToolIds

    static let contextLen = 32
    static let minTouchTargetPt: Double = 44
    static let defaultAttempts = 3

    // MARK: - Placement types

    enum PlacementMode: String, Equatable {
        case categorize
        case order
    }

    enum PlacementTarget: Equatable {
        case tray
        case bucket(String)
        case position(Int)
    }

    enum PlacementHit: Equatable {
        case item(String)
        case bucket(String)
        case tray
        case position(Int)
    }

    /// Categorize uses a map; order uses an ordered id list (tray items omitted).
    enum Placement: Equatable {
        case categorize([String: String?])
        case order([String])

        var asCategorize: [String: String?]? {
            if case .categorize(let map) = self { return map }
            return nil
        }

        var asOrder: [String]? {
            if case .order(let ids) = self { return ids }
            return nil
        }
    }

    struct EngineState: Equatable {
        var grabbedId: String?
        var target: PlacementTarget?
        var placement: Placement
    }

    struct QuoteAnchor: Equatable {
        var prefix: String
        var suffix: String
        var approxOffset: Int
        var unitIndex: Int?
    }

    struct ResolvedRange: Equatable {
        var start: Int
        var end: Int
    }

    struct PassageUnit: Equatable {
        var index: Int
        var text: String
        var start: Int
        var end: Int
    }

    struct NormPoint: Equatable {
        var x: Double
        var y: Double
    }

    enum RegionShape: Equatable {
        case rect(x: Double, y: Double, w: Double, h: Double)
        case circle(cx: Double, cy: Double, r: Double)
        case polygon(points: [NormPoint])
    }

    struct DiagramRegion: Equatable {
        var id: String
        var label: String
        var description: String
        var shape: RegionShape
    }

    enum CheckError: String, Equatable {
        case maxAttempts = "max_attempts"
        case invalidPlacement = "invalid_placement"
        case incomplete = "incomplete"
        case invalidConfig = "invalid_config"
    }

    struct CheckResultView: Equatable {
        var scorePct: Double?
        var attemptsRemaining: Int?
        var showPerItem: Bool
        var error: CheckError?
        var message: String?
        var perItem: [String: Bool]
    }

    // MARK: - Allowlist / registry

    static func isClientAllowlisted(
        _ toolId: String,
        allowlist: Set<String> = clientAllowlist
    ) -> Bool {
        allowlist.contains(toolId)
    }

    static func allowlistedToolIds(allowlist: Set<String> = clientAllowlist) -> Set<String> {
        pack3ToolIds.intersection(allowlist)
    }

    static func conflictPolicy(for toolId: String) -> ContentToolHostLogic.ConflictPolicy {
        switch toolId {
        case "highlight_annotate":
            return .merge
        default:
            return .serverWins
        }
    }

    /// Pack-3 actions are never queued offline (CT.M3 FR-11).
    static func canQueueActionOffline(toolId: String, action: String) -> Bool {
        _ = toolId
        _ = action
        return false
    }

    // MARK: - Attempts

    static func parseAttemptsConfig(_ raw: JSONValue?) -> Int? {
        guard let raw else { return defaultAttempts }
        switch raw {
        case .string(let text) where text.lowercased() == "unlimited":
            return nil
        case .string(let text):
            return Int(text).map { max(1, $0) } ?? defaultAttempts
        case .number(let number):
            return max(1, Int(number.rounded()))
        default:
            return defaultAttempts
        }
    }

    static func canCheck(attemptsUsed: Int, maxAttempts: Int?, readOnly: Bool) -> Bool {
        if readOnly { return false }
        guard let maxAttempts else { return true }
        return attemptsUsed < maxAttempts
    }

    static func attemptsRemaining(attemptsUsed: Int, maxAttempts: Int?) -> Int {
        guard let maxAttempts else { return -1 }
        return max(0, maxAttempts - attemptsUsed)
    }

    // MARK: - Sort / placement engine

    static func emptyPlacement(mode: PlacementMode, itemIds: [String]) -> Placement {
        switch mode {
        case .order:
            return .order([])
        case .categorize:
            var map: [String: String?] = [:]
            for id in itemIds { map[id] = nil as String? }
            return .categorize(map)
        }
    }

    static func createInitialEngineState(
        mode: PlacementMode,
        itemIds: [String],
        existing: Placement? = nil
    ) -> EngineState {
        EngineState(
            grabbedId: nil,
            target: nil,
            placement: existing ?? emptyPlacement(mode: mode, itemIds: itemIds)
        )
    }

    static func isLocked(_ locked: [String], id: String) -> Bool {
        locked.contains(id)
    }

    static func trayItemIds(mode: PlacementMode, itemIds: [String], placement: Placement) -> [String] {
        switch mode {
        case .order:
            let order = placement.asOrder ?? []
            let placed = Set(order)
            return itemIds.filter { !placed.contains($0) }
        case .categorize:
            let map = placement.asCategorize ?? [:]
            return itemIds.filter { map[$0] == nil || map[$0]! == nil }
        }
    }

    static func itemsInBucket(placement: Placement, bucketId: String) -> [String] {
        guard let map = placement.asCategorize else { return [] }
        return map.compactMap { id, bucket in
            guard let bucket, bucket == bucketId else { return nil }
            return id
        }
    }

    static func allPlaced(mode: PlacementMode, itemIds: [String], placement: Placement) -> Bool {
        if itemIds.isEmpty { return false }
        switch mode {
        case .order:
            let order = placement.asOrder ?? []
            return itemIds.allSatisfy { order.contains($0) }
        case .categorize:
            let map = placement.asCategorize ?? [:]
            return itemIds.allSatisfy { id in
                if let bucket = map[id], let bucket, !bucket.isEmpty { return true }
                return false
            }
        }
    }

    static func pickUp(
        _ state: EngineState,
        mode: PlacementMode,
        lockedItemIds: [String],
        itemId: String
    ) -> EngineState {
        if isLocked(lockedItemIds, id: itemId) { return state }
        let target: PlacementTarget = mode == .order
            ? .position(state.placement.asOrder?.count ?? 0)
            : .tray
        var next = state
        next.grabbedId = itemId
        next.target = target
        return next
    }

    static func cancelGrab(_ state: EngineState) -> EngineState {
        guard state.grabbedId != nil else { return state }
        var next = state
        next.grabbedId = nil
        next.target = nil
        return next
    }

    /// Interrupted drag restores last settled placement (never a half-applied one).
    static func restoreAfterDragInterrupt(settled: EngineState) -> EngineState {
        EngineState(grabbedId: nil, target: nil, placement: settled.placement)
    }

    static func drop(
        _ state: EngineState,
        mode: PlacementMode,
        lockedItemIds: [String]
    ) -> EngineState {
        guard let itemId = state.grabbedId, let target = state.target else { return state }
        if isLocked(lockedItemIds, id: itemId) {
            return cancelGrab(state)
        }
        switch mode {
        case .order:
            var order = state.placement.asOrder ?? []
            order.removeAll { $0 == itemId }
            let index: Int
            if case .position(let i) = target {
                index = max(0, min(order.count, i))
            } else {
                index = order.count
            }
            order.insert(itemId, at: index)
            return EngineState(grabbedId: nil, target: nil, placement: .order(order))
        case .categorize:
            var map = state.placement.asCategorize ?? [:]
            switch target {
            case .tray:
                map[itemId] = nil as String?
            case .bucket(let bucketId):
                map[itemId] = bucketId
            case .position:
                break
            }
            return EngineState(grabbedId: nil, target: nil, placement: .categorize(map))
        }
    }

    static func tapItemOrTarget(
        _ state: EngineState,
        mode: PlacementMode,
        lockedItemIds: [String],
        hit: PlacementHit
    ) -> EngineState {
        guard let grabbed = state.grabbedId else {
            if case .item(let id) = hit {
                return pickUp(state, mode: mode, lockedItemIds: lockedItemIds, itemId: id)
            }
            return state
        }
        if case .item(let id) = hit, id == grabbed {
            return cancelGrab(state)
        }
        let target: PlacementTarget
        switch hit {
        case .tray:
            target = .tray
        case .bucket(let id):
            target = .bucket(id)
        case .position(let index):
            target = .position(index)
        case .item(let id):
            if mode == .order, let order = state.placement.asOrder {
                let idx = order.firstIndex(of: id) ?? order.count
                target = .position(idx)
            } else if let map = state.placement.asCategorize, let bucket = map[id], let bucket {
                target = .bucket(bucket)
            } else {
                target = .tray
            }
        }
        var next = state
        next.target = target
        return drop(next, mode: mode, lockedItemIds: lockedItemIds)
    }

    static func placeViaPointer(
        _ state: EngineState,
        mode: PlacementMode,
        lockedItemIds: [String],
        itemId: String,
        target: PlacementTarget
    ) -> EngineState {
        if isLocked(lockedItemIds, id: itemId) { return state }
        var next = state
        next.grabbedId = itemId
        next.target = target
        return drop(next, mode: mode, lockedItemIds: lockedItemIds)
    }

    /// Move up (-1) / move down (+1) in order mode.
    static func moveInOrder(
        order: [String],
        itemId: String,
        direction: Int,
        lockedItemIds: [String]
    ) -> [String] {
        guard !isLocked(lockedItemIds, id: itemId),
              let idx = order.firstIndex(of: itemId)
        else { return order }
        let next = idx + direction
        guard next >= 0, next < order.count else { return order }
        if isLocked(lockedItemIds, id: order[next]) { return order }
        var out = order
        out.swapAt(idx, next)
        return out
    }

    // MARK: - Highlight / quote anchors (UTF-16 offsets, web parity)

    static func utf16Length(_ text: String) -> Int {
        text.utf16.count
    }

    static func utf16Slice(_ text: String, start: Int, end: Int) -> String? {
        let utf16 = text.utf16
        guard start >= 0, end >= start, end <= utf16.count else { return nil }
        let startIdx = utf16.index(utf16.startIndex, offsetBy: start)
        let endIdx = utf16.index(utf16.startIndex, offsetBy: end)
        return String(utf16[startIdx ..< endIdx])
    }

    static func buildQuoteAnchor(
        passage: String,
        start: Int,
        end: Int,
        unitIndex: Int? = nil
    ) -> (quote: String, anchor: QuoteAnchor)? {
        guard let quote = utf16Slice(passage, start: start, end: end),
              !quote.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        else { return nil }
        let prefixStart = max(0, start - contextLen)
        let suffixEnd = min(utf16Length(passage), end + contextLen)
        var anchor = QuoteAnchor(
            prefix: utf16Slice(passage, start: prefixStart, end: start) ?? "",
            suffix: utf16Slice(passage, start: end, end: suffixEnd) ?? "",
            approxOffset: start,
            unitIndex: nil
        )
        if let unitIndex, unitIndex >= 0 {
            anchor.unitIndex = unitIndex
        }
        return (quote, anchor)
    }

    private static func contextScore(
        full: String,
        idx: Int,
        quote: String,
        anchor: QuoteAnchor
    ) -> Double {
        var score = 0.0
        if !anchor.prefix.isEmpty {
            let beforeStart = max(0, idx - utf16Length(anchor.prefix))
            let before = utf16Slice(full, start: beforeStart, end: idx) ?? ""
            if before.hasSuffix(anchor.prefix) {
                score += 2
            } else if !before.isEmpty {
                let n = min(8, before.count)
                let tail = String(before.suffix(n))
                if anchor.prefix.hasSuffix(tail) { score += 1 }
            }
        }
        if !anchor.suffix.isEmpty {
            let afterStart = idx + utf16Length(quote)
            let afterEnd = afterStart + utf16Length(anchor.suffix)
            let after = utf16Slice(full, start: afterStart, end: min(utf16Length(full), afterEnd)) ?? ""
            if after.hasPrefix(anchor.suffix) {
                score += 2
            } else if !after.isEmpty {
                let n = min(8, after.count)
                let head = String(after.prefix(n))
                if anchor.suffix.hasPrefix(head) { score += 1 }
            }
        }
        score -= abs(Double(idx - anchor.approxOffset)) / 1e6
        return score
    }

    private static func bestQuoteIndex(full: String, quote: String, anchor: QuoteAnchor) -> Int {
        guard !quote.isEmpty else { return -1 }
        var best = -1
        var bestScore = -Double.infinity
        var from = 0
        let fullLen = utf16Length(full)
        let quoteLen = utf16Length(quote)
        while from + quoteLen <= fullLen {
            if utf16Slice(full, start: from, end: from + quoteLen) == quote {
                let score = contextScore(full: full, idx: from, quote: quote, anchor: anchor)
                if score > bestScore {
                    bestScore = score
                    best = from
                }
                from += 1
            } else {
                from += 1
            }
        }
        return best
    }

    static func resolveQuoteAnchor(
        passage: String,
        quote: String,
        anchor: QuoteAnchor
    ) -> ResolvedRange? {
        guard !quote.isEmpty else { return nil }
        let end = anchor.approxOffset + utf16Length(quote)
        if anchor.approxOffset >= 0,
           end <= utf16Length(passage),
           utf16Slice(passage, start: anchor.approxOffset, end: end) == quote {
            return ResolvedRange(start: anchor.approxOffset, end: end)
        }
        let idx = bestQuoteIndex(full: passage, quote: quote, anchor: anchor)
        if idx >= 0 {
            return ResolvedRange(start: idx, end: idx + utf16Length(quote))
        }
        return nil
    }

    static func segmentPassage(
        _ passage: String,
        granularity: String = "sentence"
    ) -> [PassageUnit] {
        let text = passage
        if text.isEmpty { return [] }
        switch granularity {
        case "paragraph":
            return segmentParagraphs(text)
        case "line":
            return segmentLines(text)
        default:
            return segmentSentences(text)
        }
    }

    private static func segmentParagraphs(_ text: String) -> [PassageUnit] {
        var units: [PassageUnit] = []
        var idx = 0
        let ns = text as NSString
        let pattern = try! NSRegularExpression(pattern: "[^\\n]+")
        let range = NSRange(location: 0, length: ns.length)
        pattern.enumerateMatches(in: text, options: [], range: range) { match, _, _ in
            guard let match else { return }
            let chunk = ns.substring(with: match.range)
            if chunk.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return }
            units.append(PassageUnit(index: idx, text: chunk, start: match.range.location, end: match.range.location + match.range.length))
            idx += 1
        }
        return units.isEmpty ? [PassageUnit(index: 0, text: text, start: 0, end: utf16Length(text))] : units
    }

    private static func segmentLines(_ text: String) -> [PassageUnit] {
        var units: [PassageUnit] = []
        var start = 0
        var idx = 0
        let utf16 = text.utf16
        let count = utf16.count
        for i in 0 ... count {
            if i == count || utf16[utf16.index(utf16.startIndex, offsetBy: i)] == 10 {
                if let chunk = utf16Slice(text, start: start, end: i),
                   !chunk.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    units.append(PassageUnit(index: idx, text: chunk, start: start, end: i))
                    idx += 1
                }
                start = i + 1
            }
        }
        return units.isEmpty ? [PassageUnit(index: 0, text: text, start: 0, end: count)] : units
    }

    private static func segmentSentences(_ text: String) -> [PassageUnit] {
        var units: [PassageUnit] = []
        let ns = text as NSString
        let pattern = try! NSRegularExpression(pattern: "[^.!?…。！？]+(?:[.!?…。！？]+|$)")
        let range = NSRange(location: 0, length: ns.length)
        var idx = 0
        pattern.enumerateMatches(in: text, options: [], range: range) { match, _, _ in
            guard let match else { return }
            let raw = ns.substring(with: match.range)
            let leading = raw.prefix(while: { $0.isWhitespace }).count
            let trailing = raw.reversed().prefix(while: { $0.isWhitespace }).count
            let start = match.range.location + leading
            let end = match.range.location + match.range.length - trailing
            guard end > start else { return }
            let chunk = ns.substring(with: NSRange(location: start, length: end - start))
            if chunk.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty { return }
            units.append(PassageUnit(index: idx, text: chunk, start: start, end: end))
            idx += 1
        }
        return units.isEmpty ? [PassageUnit(index: 0, text: text, start: 0, end: utf16Length(text))] : units
    }

    static func plainPassageFromMarkdown(_ md: String) -> String {
        var s = md
        if let re = try? NSRegularExpression(pattern: "```[\\s\\S]*?```") {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: " "
            )
        }
        if let re = try? NSRegularExpression(pattern: "`([^`]+)`") {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: "$1"
            )
        }
        if let re = try? NSRegularExpression(pattern: "!\\[[^\\]]*\\]\\([^)]*\\)") {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: " "
            )
        }
        if let re = try? NSRegularExpression(pattern: "\\[([^\\]]*)\\]\\([^)]*\\)") {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: "$1"
            )
        }
        if let re = try? NSRegularExpression(pattern: "^#{1,6}\\s+", options: .anchorsMatchLines) {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: ""
            )
        }
        if let re = try? NSRegularExpression(pattern: "[*_~]+") {
            s = re.stringByReplacingMatches(
                in: s, options: [], range: NSRange(location: 0, length: (s as NSString).length), withTemplate: ""
            )
        }
        s = s.replacingOccurrences(of: "\r\n", with: "\n")
        return s.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    // MARK: - Diagram geometry

    static func clamp01(_ v: Double) -> Double {
        if v < 0 { return 0 }
        if v > 1 { return 1 }
        return v
    }

    static func pointInShape(x: Double, y: Double, shape: RegionShape) -> Bool {
        switch shape {
        case .rect(let rx, let ry, let w, let h):
            return x >= rx && x <= rx + w && y >= ry && y <= ry + h
        case .circle(let cx, let cy, let r):
            let dx = x - cx
            let dy = y - cy
            return dx * dx + dy * dy <= r * r
        case .polygon(let points):
            return pointInPolygon(x: x, y: y, points: points)
        }
    }

    private static func pointInPolygon(x: Double, y: Double, points: [NormPoint]) -> Bool {
        guard points.count >= 3 else { return false }
        var inside = false
        var j = points.count - 1
        for i in 0 ..< points.count {
            let xi = points[i].x, yi = points[i].y
            let xj = points[j].x, yj = points[j].y
            let intersect = (yi > y) != (yj > y)
                && x < ((xj - xi) * (y - yi)) / (yj - yi + 1e-15) + xi
            if intersect { inside = !inside }
            j = i
        }
        return inside
    }

    static func shapeArea(_ shape: RegionShape) -> Double {
        switch shape {
        case .rect(_, _, let w, let h):
            return abs(w * h)
        case .circle(_, _, let r):
            return Double.pi * r * r
        case .polygon(let points):
            return abs(polygonArea(points))
        }
    }

    private static func polygonArea(_ points: [NormPoint]) -> Double {
        guard points.count >= 3 else { return 0 }
        var sum = 0.0
        var j = points.count - 1
        for i in 0 ..< points.count {
            sum += (points[j].x + points[i].x) * (points[j].y - points[i].y)
            j = i
        }
        return sum / 2
    }

    static func hitTestRegions(regions: [DiagramRegion], x: Double, y: Double) -> DiagramRegion? {
        var best: DiagramRegion?
        var bestArea = Double.infinity
        for region in regions {
            guard pointInShape(x: x, y: y, shape: region.shape) else { continue }
            let area = shapeArea(region.shape)
            if area < bestArea {
                bestArea = area
                best = region
            }
        }
        return best
    }

    /// Expand hit area to ≥ minTargetPt without changing stored coordinates.
    /// Returns whether a normalised point hits the expanded target.
    static func pointInExpandedHitTarget(
        x: Double,
        y: Double,
        shape: RegionShape,
        imageDisplayWidthPt: Double,
        imageDisplayHeightPt: Double,
        minTargetPt: Double = minTouchTargetPt
    ) -> Bool {
        if pointInShape(x: x, y: y, shape: shape) { return true }
        guard imageDisplayWidthPt > 0, imageDisplayHeightPt > 0 else { return false }
        let (cx, cy, halfW, halfH) = boundingCenterHalfExtents(shape)
        let minHalfX = (minTargetPt / 2) / imageDisplayWidthPt
        let minHalfY = (minTargetPt / 2) / imageDisplayHeightPt
        let hw = max(halfW, minHalfX)
        let hh = max(halfH, minHalfY)
        return abs(x - cx) <= hw && abs(y - cy) <= hh
    }

    private static func boundingCenterHalfExtents(_ shape: RegionShape) -> (Double, Double, Double, Double) {
        switch shape {
        case .rect(let x, let y, let w, let h):
            return (x + w / 2, y + h / 2, abs(w) / 2, abs(h) / 2)
        case .circle(let cx, let cy, let r):
            return (cx, cy, abs(r), abs(r))
        case .polygon(let points):
            guard !points.isEmpty else { return (0.5, 0.5, 0, 0) }
            let xs = points.map(\.x)
            let ys = points.map(\.y)
            let minX = xs.min() ?? 0
            let maxX = xs.max() ?? 0
            let minY = ys.min() ?? 0
            let maxY = ys.max() ?? 0
            return ((minX + maxX) / 2, (minY + maxY) / 2, (maxX - minX) / 2, (maxY - minY) / 2)
        }
    }

    /// Map a view-local pointer into normalised image coords (object-fit: contain + zoom).
    static func pointerToNormalized(
        clientX: Double,
        clientY: Double,
        viewWidth: Double,
        viewHeight: Double,
        naturalWidth: Double,
        naturalHeight: Double,
        zoom: Double = 1,
        panX: Double = 0,
        panY: Double = 0
    ) -> (Double, Double)? {
        guard viewWidth > 0, viewHeight > 0, naturalWidth > 0, naturalHeight > 0, zoom > 0 else {
            return nil
        }
        let cx = viewWidth / 2
        let cy = viewHeight / 2
        let localX = (clientX - cx) / zoom - panX + viewWidth / 2
        let localY = (clientY - cy) / zoom - panY + viewHeight / 2
        let scale = min(viewWidth / naturalWidth, viewHeight / naturalHeight)
        let drawW = naturalWidth * scale
        let drawH = naturalHeight * scale
        let offsetX = (viewWidth - drawW) / 2
        let offsetY = (viewHeight - drawH) / 2
        let imgX = localX - offsetX
        let imgY = localY - offsetY
        if imgX < 0 || imgY < 0 || imgX > drawW || imgY > drawH { return nil }
        return (clamp01(imgX / drawW), clamp01(imgY / drawH))
    }

    // MARK: - Check result parsing

    static func parseCheckResult(_ value: JSONValue?) -> CheckResultView {
        let obj = objectMap(value)
        var perItem: [String: Bool] = [:]
        if case .object(let map)? = obj["perItem"] {
            for (id, entry) in map {
                if case .object(let row) = entry, case .bool(let flag) = row["correct"] {
                    perItem[id] = flag
                } else if case .bool(let flag) = entry {
                    perItem[id] = flag
                }
            }
        }
        let errorRaw = stringField(value, key: "error")
        let error = errorRaw.flatMap { CheckError(rawValue: $0) }
        return CheckResultView(
            scorePct: numberField(value, key: "scorePct"),
            attemptsRemaining: numberField(value, key: "attemptsRemaining").map { Int($0.rounded()) },
            showPerItem: boolField(value, key: "showPerItem") ?? true,
            error: error,
            message: stringField(value, key: "message"),
            perItem: perItem
        )
    }

    // MARK: - JSON helpers

    static func mergePreservingUnknown(
        base: [String: JSONValue],
        patch: [String: JSONValue]
    ) -> [String: JSONValue] {
        var out = base
        for (k, v) in patch { out[k] = v }
        return out
    }

    static func objectMap(_ value: JSONValue?) -> [String: JSONValue] {
        guard case .object(let obj) = value else { return [:] }
        return obj
    }

    static func arrayField(_ value: JSONValue?, key: String) -> [JSONValue] {
        guard case .object(let obj) = value, case .array(let arr) = obj[key] else { return [] }
        return arr
    }

    static func boolField(_ value: JSONValue?, key: String) -> Bool? {
        guard case .object(let obj) = value, let field = obj[key] else { return nil }
        if case .bool(let flag) = field { return flag }
        return nil
    }

    static func numberField(_ value: JSONValue?, key: String) -> Double? {
        guard case .object(let obj) = value, let field = obj[key] else { return nil }
        if case .number(let number) = field { return number }
        if case .string(let text) = field { return Double(text) }
        return nil
    }

    static func stringField(_ value: JSONValue?, key: String) -> String? {
        ContentToolHostLogic.stringField(value, key: key)
    }

    static func parseCategorizePlacement(_ value: JSONValue?) -> [String: String?] {
        guard case .object(let obj) = value else { return [:] }
        var out: [String: String?] = [:]
        for (k, v) in obj {
            switch v {
            case .string(let s): out[k] = s
            case .null: out[k] = nil as String?
            default: break
            }
        }
        return out
    }

    static func parseOrderPlacement(_ value: JSONValue?) -> [String] {
        guard case .array(let arr) = value else { return [] }
        return arr.compactMap {
            if case .string(let s) = $0 { return s }
            return nil
        }
    }

    static func categorizePlacementJSON(_ map: [String: String?]) -> JSONValue {
        var obj: [String: JSONValue] = [:]
        for (k, v) in map {
            if let v { obj[k] = .string(v) } else { obj[k] = .null }
        }
        return .object(obj)
    }

    static func orderPlacementJSON(_ ids: [String]) -> JSONValue {
        .array(ids.map { .string($0) })
    }
}
