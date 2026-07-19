import Foundation
import CoreGraphics

/// Pure helpers for course whiteboard authoring (MOB.6) — mirrors web `whiteboard/types` + toolbar.
enum WhiteboardTool: String, CaseIterable, Equatable {
    case select
    case pen
    case line
    case rect
    case circle
    case triangle
    case eraser

    var accessibilityKey: String {
        "mobile.whiteboard.tool.\(rawValue)"
    }
}

enum WhiteboardLogic {
    static let colors: [String] = [
        "#1e293b", "#ef4444", "#f97316", "#eab308", "#22c55e",
        "#3b82f6", "#a855f7", "#ec4899", "#ffffff",
    ]
    static let strokeWidths: [Double] = [2, 4, 8]
    static let eraserSizes: [Double] = [8, 16, 32]
    static let undoLimit = 50
    static let autosaveDelayMs: UInt64 = 800
    static let pickHitRadius: Double = 8

    /// Edit when the mobile rollout flag is on and the viewer has staff/create rights.
    static func canEdit(viewerIsStaff: Bool, features: MobilePlatformFeatures) -> Bool {
        viewerIsStaff && features.ffMobileWhiteboardEdit
    }

    static func normalizeTitle(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    static func isValidTitle(_ raw: String) -> Bool {
        !normalizeTitle(raw).isEmpty
    }

    static func defaultTitle(existingCount: Int) -> String {
        "Whiteboard \(max(1, existingCount + 1))"
    }

    // MARK: - Undo / redo

    struct History {
        var undoStack: [[WhiteboardElement]] = []
        var redoStack: [[WhiteboardElement]] = []

        mutating func push(_ elements: [WhiteboardElement]) {
            undoStack.append(elements)
            if undoStack.count > WhiteboardLogic.undoLimit {
                undoStack.removeFirst()
            }
            redoStack.removeAll()
        }

        mutating func undo(current: [WhiteboardElement]) -> [WhiteboardElement]? {
            guard let previous = undoStack.popLast() else { return nil }
            redoStack.append(current)
            return previous
        }

        mutating func redo(current: [WhiteboardElement]) -> [WhiteboardElement]? {
            guard let next = redoStack.popLast() else { return nil }
            undoStack.append(current)
            return next
        }

        var canUndo: Bool { !undoStack.isEmpty }
        var canRedo: Bool { !redoStack.isEmpty }
    }

    // MARK: - Element builders (web-compatible schema)

    static func stroke(color: String, width: Double, pts: [CGPoint]) -> WhiteboardElement {
        WhiteboardElement(
            type: "stroke",
            color: color,
            width: width,
            pts: pts.map { [Double($0.x), Double($0.y)] }
        )
    }

    static func line(color: String, width: Double, from: CGPoint, to: CGPoint) -> WhiteboardElement {
        WhiteboardElement(
            type: "line",
            color: color,
            width: width,
            point1X: Double(from.x),
            point1Y: Double(from.y),
            point2X: Double(to.x),
            point2Y: Double(to.y)
        )
    }

    static func rect(color: String, width: Double, from: CGPoint, to: CGPoint) -> WhiteboardElement {
        let originX = min(from.x, to.x)
        let originY = min(from.y, to.y)
        return WhiteboardElement(
            type: "rect",
            color: color,
            width: width,
            rectX: Double(originX),
            rectY: Double(originY),
            rectWidth: Double(abs(to.x - from.x)),
            rectHeight: Double(abs(to.y - from.y))
        )
    }

    static func circle(color: String, width: Double, from: CGPoint, to: CGPoint) -> WhiteboardElement {
        WhiteboardElement(
            type: "circle",
            color: color,
            width: width,
            centerX: Double((from.x + to.x) / 2),
            centerY: Double((from.y + to.y) / 2),
            radiusX: Double(abs(to.x - from.x) / 2),
            radiusY: Double(abs(to.y - from.y) / 2)
        )
    }

    static func triangle(color: String, width: Double, from: CGPoint, to: CGPoint) -> WhiteboardElement {
        WhiteboardElement(
            type: "triangle",
            color: color,
            width: width,
            point1X: Double((from.x + to.x) / 2),
            point1Y: Double(min(from.y, to.y)),
            point2X: Double(from.x),
            point2Y: Double(max(from.y, to.y)),
            point3X: Double(to.x),
            point3Y: Double(max(from.y, to.y))
        )
    }

    // MARK: - Hit test / erase / select (parity with notebook + web pick)

    static func hitTest(_ element: WhiteboardElement, point: CGPoint, radius: Double) -> Bool {
        func distToSegment(_ probe: CGPoint, _ start: CGPoint, _ end: CGPoint) -> Double {
            let deltaX = end.x - start.x
            let deltaY = end.y - start.y
            let lengthSq = deltaX * deltaX + deltaY * deltaY
            if lengthSq == 0 { return Double(hypot(probe.x - start.x, probe.y - start.y)) }
            let projection = max(0, min(1, ((probe.x - start.x) * deltaX + (probe.y - start.y) * deltaY) / lengthSq))
            return Double(hypot(
                probe.x - (start.x + projection * deltaX),
                probe.y - (start.y + projection * deltaY)
            ))
        }

        switch element.type {
        case "stroke":
            let pts = element.points
            let hitRadius = radius + element.width / 2
            if pts.count == 1 {
                return Double(hypot(point.x - pts[0].x, point.y - pts[0].y)) <= hitRadius
            }
            for index in 0 ..< max(0, pts.count - 1) where distToSegment(point, pts[index], pts[index + 1]) <= hitRadius {
                return true
            }
            return false
        case "line":
            let pts = element.points
            guard pts.count == 2 else { return false }
            return distToSegment(point, pts[0], pts[1]) <= radius + element.width / 2
        case "rect":
            guard let rect = element.rect else { return false }
            let corners = [
                CGPoint(x: rect.minX, y: rect.minY),
                CGPoint(x: rect.maxX, y: rect.minY),
                CGPoint(x: rect.maxX, y: rect.maxY),
                CGPoint(x: rect.minX, y: rect.maxY),
            ]
            let hitRadius = radius + element.width / 2
            for index in 0 ..< 4 where distToSegment(point, corners[index], corners[(index + 1) % 4]) <= hitRadius {
                return true
            }
            return false
        case "circle":
            guard let ellipse = element.ellipse else { return false }
            let radiusX = Double(ellipse.width / 2)
            let radiusY = Double(ellipse.height / 2)
            guard radiusX > 0, radiusY > 0 else { return false }
            let center = CGPoint(x: ellipse.midX, y: ellipse.midY)
            let normX = (point.x - center.x) / radiusX
            let normY = (point.y - center.y) / radiusY
            let norm = Double(hypot(normX, normY))
            return abs(norm - 1) * min(radiusX, radiusY) <= radius + element.width / 2
        case "triangle":
            let pts = element.points
            guard pts.count == 3 else { return false }
            let hitRadius = radius + element.width / 2
            for index in 0 ..< 3 where distToSegment(point, pts[index], pts[(index + 1) % 3]) <= hitRadius {
                return true
            }
            return false
        default:
            return false
        }
    }

    static func erase(from elements: [WhiteboardElement], at point: CGPoint, radius: Double) -> [WhiteboardElement] {
        elements.filter { !hitTest($0, point: point, radius: radius) }
    }

    static func pickElement(_ elements: [WhiteboardElement], at point: CGPoint) -> Int? {
        for index in stride(from: elements.count - 1, through: 0, by: -1) {
            if hitTest(elements[index], point: point, radius: pickHitRadius) {
                return index
            }
        }
        return nil
    }

    static func translate(_ element: WhiteboardElement, dx: Double, dy: Double) -> WhiteboardElement {
        var next = element
        switch element.type {
        case "stroke":
            next.pts = element.pts?.map { pair in
                guard pair.count >= 2 else { return pair }
                return [pair[0] + dx, pair[1] + dy]
            }
        case "rect":
            if let rectX = element.rectX { next.rectX = rectX + dx }
            if let rectY = element.rectY { next.rectY = rectY + dy }
        case "circle":
            if let cx = element.centerX { next.centerX = cx + dx }
            if let cy = element.centerY { next.centerY = cy + dy }
        case "line", "triangle":
            if let x1 = element.point1X { next.point1X = x1 + dx }
            if let y1 = element.point1Y { next.point1Y = y1 + dy }
            if let x2 = element.point2X { next.point2X = x2 + dx }
            if let y2 = element.point2Y { next.point2Y = y2 + dy }
            if let x3 = element.point3X { next.point3X = x3 + dx }
            if let y3 = element.point3Y { next.point3Y = y3 + dy }
        default:
            break
        }
        return next
    }

    // MARK: - Wire JSON (byte-compatible with web serialize)

    static func serializeElements(_ elements: [WhiteboardElement]) -> [[String: Any]] {
        elements.compactMap(serializeElement)
    }

    static func serializeElement(_ element: WhiteboardElement) -> [String: Any]? {
        var base: [String: Any] = [
            "type": element.type,
            "color": element.color,
            "width": element.width,
        ]
        switch element.type {
        case "stroke":
            guard let pts = element.pts else { return nil }
            base["pts"] = pts
        case "rect":
            guard let rectX = element.rectX, let rectY = element.rectY,
                  let rectW = element.rectWidth, let rectH = element.rectHeight else { return nil }
            base["x"] = rectX
            base["y"] = rectY
            base["w"] = rectW
            base["h"] = rectH
        case "circle":
            guard let centerX = element.centerX, let centerY = element.centerY,
                  let radiusX = element.radiusX, let radiusY = element.radiusY else { return nil }
            base["cx"] = centerX
            base["cy"] = centerY
            base["rx"] = radiusX
            base["ry"] = radiusY
        case "line":
            guard let point1X = element.point1X, let point1Y = element.point1Y,
                  let point2X = element.point2X, let point2Y = element.point2Y else { return nil }
            base["x1"] = point1X
            base["y1"] = point1Y
            base["x2"] = point2X
            base["y2"] = point2Y
        case "triangle":
            guard let point1X = element.point1X, let point1Y = element.point1Y,
                  let point2X = element.point2X, let point2Y = element.point2Y,
                  let point3X = element.point3X, let point3Y = element.point3Y else { return nil }
            base["x1"] = point1X
            base["y1"] = point1Y
            base["x2"] = point2X
            base["y2"] = point2Y
            base["x3"] = point3X
            base["y3"] = point3Y
        default:
            return nil
        }
        return base
    }

    static func canvasDataJSONString(_ elements: [WhiteboardElement]) -> String {
        let array = serializeElements(elements)
        guard let data = try? JSONSerialization.data(withJSONObject: array) else { return "[]" }
        return String(data: data, encoding: .utf8) ?? "[]"
    }

    static func parseElements(fromJSON json: String) -> [WhiteboardElement] {
        guard let data = json.data(using: .utf8) else { return [] }
        return (try? JSONDecoder().decode([WhiteboardElement].self, from: data)) ?? []
    }

    /// Finger vs stylus: when a stylus is active, ignore finger ink (palm rejection).
    static func shouldAcceptTouch(isStylus: Bool, stylusExclusiveDrawing: Bool) -> Bool {
        if !stylusExclusiveDrawing { return true }
        return isStylus
    }
}

enum WhiteboardObservability {
    private static var counters: [String: Int] = [:]
    private static let lock = NSLock()

    static func record(_ event: String, attributes: [String: String] = [:]) {
        lock.lock()
        defer { lock.unlock() }
        let key = attributes.isEmpty
            ? event
            : event + "|" + attributes.keys.sorted().map { "\($0)=\(attributes[$0] ?? "")" }.joined(separator: ",")
        counters[key, default: 0] += 1
    }

    static func count(for event: String) -> Int {
        lock.lock()
        defer { lock.unlock() }
        return counters.filter { $0.key == event || $0.key.hasPrefix(event + "|") }.values.reduce(0, +)
    }

    #if DEBUG
    static func resetForTests() {
        lock.lock()
        counters.removeAll()
        lock.unlock()
    }
    #endif
}
