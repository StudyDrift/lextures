import Foundation
import SwiftUI

/// One whiteboard element from a ```drawing fenced block (parity with web `whiteboard/types`).
enum NotebookDrawEl: Equatable {
    case stroke(color: String, width: Double, pts: [CGPoint])
    case rect(color: String, width: Double, rectX: Double, rectY: Double, rectWidth: Double, rectHeight: Double)
    case circle(color: String, width: Double, cx: Double, cy: Double, rx: Double, ry: Double)
    case triangle(color: String, width: Double, x1: Double, y1: Double, x2: Double, y2: Double, x3: Double, y3: Double)
    case line(color: String, width: Double, x1: Double, y1: Double, x2: Double, y2: Double)
}

enum NotebookDrawing {
    static let colors: [String] = [
        "#1e293b", "#ef4444", "#f97316", "#eab308", "#22c55e",
        "#3b82f6", "#a855f7", "#ec4899", "#ffffff",
    ]
    static let strokeWidths: [Double] = [2, 4, 8]

    // MARK: - JSON (parity with web `whiteboard/serialize`)

    static func parseElements(_ raw: String) -> [NotebookDrawEl] {
        guard
            let data = raw.data(using: .utf8),
            let array = (try? JSONSerialization.jsonObject(with: data)) as? [[String: Any]]
        else { return [] }
        return array.compactMap(parseElement)
    }

    private static func num(_ value: Any?) -> Double? {
        (value as? NSNumber)?.doubleValue
    }

    private static func parseElement(_ obj: [String: Any]) -> NotebookDrawEl? {
        let color = obj["color"] as? String ?? "#1e293b"
        let width = num(obj["width"]) ?? 2
        switch obj["type"] as? String {
        case "stroke":
            let pts = (obj["pts"] as? [[Any]] ?? []).compactMap { pair -> CGPoint? in
                guard pair.count >= 2, let pointX = num(pair[0]), let pointY = num(pair[1]) else { return nil }
                return CGPoint(x: pointX, y: pointY)
            }
            return .stroke(color: color, width: width, pts: pts)
        case "rect":
            guard
                let rectX = num(obj["x"]), let rectY = num(obj["y"]),
                let rectW = num(obj["w"]), let rectH = num(obj["h"])
            else { return nil }
            return .rect(color: color, width: width, rectX: rectX, rectY: rectY, rectWidth: rectW, rectHeight: rectH)
        case "circle":
            guard
                let centerX = num(obj["cx"]), let centerY = num(obj["cy"]),
                let radiusX = num(obj["rx"]), let radiusY = num(obj["ry"])
            else { return nil }
            return .circle(color: color, width: width, cx: centerX, cy: centerY, rx: radiusX, ry: radiusY)
        case "triangle":
            guard
                let x1 = num(obj["x1"]), let y1 = num(obj["y1"]),
                let x2 = num(obj["x2"]), let y2 = num(obj["y2"]),
                let x3 = num(obj["x3"]), let y3 = num(obj["y3"])
            else { return nil }
            return .triangle(color: color, width: width, x1: x1, y1: y1, x2: x2, y2: y2, x3: x3, y3: y3)
        case "line":
            guard
                let x1 = num(obj["x1"]), let y1 = num(obj["y1"]),
                let x2 = num(obj["x2"]), let y2 = num(obj["y2"])
            else { return nil }
            return .line(color: color, width: width, x1: x1, y1: y1, x2: x2, y2: y2)
        default:
            return nil
        }
    }

    static func serializeElements(_ elements: [NotebookDrawEl]) -> String {
        let array: [[String: Any]] = elements.map { element in
            switch element {
            case .stroke(let color, let width, let pts):
                return ["type": "stroke", "color": color, "width": width, "pts": pts.map { [$0.x, $0.y] }]
            case .rect(let color, let width, let rectX, let rectY, let rectWidth, let rectHeight):
                return ["type": "rect", "color": color, "width": width, "x": rectX, "y": rectY, "w": rectWidth, "h": rectHeight]
            case .circle(let color, let width, let centerX, let centerY, let radiusX, let radiusY):
                return ["type": "circle", "color": color, "width": width, "cx": centerX, "cy": centerY, "rx": radiusX, "ry": radiusY]
            case .triangle(let color, let width, let x1, let y1, let x2, let y2, let x3, let y3):
                return ["type": "triangle", "color": color, "width": width, "x1": x1, "y1": y1, "x2": x2, "y2": y2, "x3": x3, "y3": y3]
            case .line(let color, let width, let x1, let y1, let x2, let y2):
                return ["type": "line", "color": color, "width": width, "x1": x1, "y1": y1, "x2": x2, "y2": y2]
            }
        }
        guard let data = try? JSONSerialization.data(withJSONObject: array) else { return "[]" }
        return String(data: data, encoding: .utf8) ?? "[]"
    }

    // MARK: - Geometry

    /// Content extent of the elements (origin-anchored), with a sensible floor for empty boards.
    static func contentSize(_ elements: [NotebookDrawEl], minSize: CGSize = CGSize(width: 320, height: 220)) -> CGSize {
        var maxX = minSize.width
        var maxY = minSize.height
        func grow(_ coordX: Double, _ coordY: Double) {
            maxX = max(maxX, coordX)
            maxY = max(maxY, coordY)
        }
        for element in elements {
            switch element {
            case .stroke(_, _, let pts):
                for point in pts { grow(point.x, point.y) }
            case .rect(_, _, let rectX, let rectY, let rectWidth, let rectHeight):
                grow(rectX + rectWidth, rectY + rectHeight)
                grow(rectX, rectY)
            case .circle(_, _, let centerX, let centerY, let radiusX, let radiusY):
                grow(centerX + radiusX, centerY + radiusY)
            case .triangle(_, _, let x1, let y1, let x2, let y2, let x3, let y3):
                grow(x1, y1); grow(x2, y2); grow(x3, y3)
            case .line(_, _, let x1, let y1, let x2, let y2):
                grow(x1, y1); grow(x2, y2)
            }
        }
        return CGSize(width: maxX, height: maxY)
    }

    static func color(from hex: String, fallback: Color = .primary) -> Color {
        var value = hex.trimmingCharacters(in: .whitespaces)
        guard value.hasPrefix("#") else { return fallback }
        value.removeFirst()
        guard value.count == 6, let rgb = UInt32(value, radix: 16) else { return fallback }
        return Color(
            red: Double((rgb >> 16) & 0xFF) / 255,
            green: Double((rgb >> 8) & 0xFF) / 255,
            blue: Double(rgb & 0xFF) / 255
        )
    }

    /// Draw all elements into a SwiftUI canvas context at the given scale.
    static func draw(_ elements: [NotebookDrawEl], in context: GraphicsContext, scale: CGFloat) {
        for element in elements {
            switch element {
            case .stroke(let color, let width, let pts):
                guard pts.count > 1 else {
                    if let point = pts.first {
                        let dotRadius = max(width, 2) * scale / 2
                        let dot = Path(ellipseIn: CGRect(
                            x: point.x * scale - dotRadius,
                            y: point.y * scale - dotRadius,
                            width: dotRadius * 2,
                            height: dotRadius * 2
                        ))
                        context.fill(dot, with: .color(NotebookDrawing.color(from: color)))
                    }
                    continue
                }
                var path = Path()
                path.move(to: CGPoint(x: pts[0].x * scale, y: pts[0].y * scale))
                for point in pts.dropFirst() {
                    path.addLine(to: CGPoint(x: point.x * scale, y: point.y * scale))
                }
                context.stroke(
                    path,
                    with: .color(NotebookDrawing.color(from: color)),
                    style: StrokeStyle(lineWidth: width * scale, lineCap: .round, lineJoin: .round)
                )
            case .rect(let color, let width, let rectX, let rectY, let rectWidth, let rectHeight):
                let rect = CGRect(x: rectX * scale, y: rectY * scale, width: rectWidth * scale, height: rectHeight * scale)
                context.stroke(Path(rect), with: .color(NotebookDrawing.color(from: color)), lineWidth: width * scale)
            case .circle(let color, let width, let centerX, let centerY, let radiusX, let radiusY):
                let rect = CGRect(
                    x: (centerX - radiusX) * scale,
                    y: (centerY - radiusY) * scale,
                    width: radiusX * 2 * scale,
                    height: radiusY * 2 * scale
                )
                context.stroke(Path(ellipseIn: rect), with: .color(NotebookDrawing.color(from: color)), lineWidth: width * scale)
            case .triangle(let color, let width, let x1, let y1, let x2, let y2, let x3, let y3):
                var path = Path()
                path.move(to: CGPoint(x: x1 * scale, y: y1 * scale))
                path.addLine(to: CGPoint(x: x2 * scale, y: y2 * scale))
                path.addLine(to: CGPoint(x: x3 * scale, y: y3 * scale))
                path.closeSubpath()
                context.stroke(
                    path,
                    with: .color(NotebookDrawing.color(from: color)),
                    style: StrokeStyle(lineWidth: width * scale, lineJoin: .round)
                )
            case .line(let color, let width, let x1, let y1, let x2, let y2):
                var path = Path()
                path.move(to: CGPoint(x: x1 * scale, y: y1 * scale))
                path.addLine(to: CGPoint(x: x2 * scale, y: y2 * scale))
                context.stroke(
                    path,
                    with: .color(NotebookDrawing.color(from: color)),
                    style: StrokeStyle(lineWidth: width * scale, lineCap: .round)
                )
            }
        }
    }

    /// Whether a point (content coordinates) is within `radius` of an element — eraser hit test.
    static func hitTest(_ element: NotebookDrawEl, point: CGPoint, radius: Double) -> Bool {
        func distToSegment(_ probe: CGPoint, _ start: CGPoint, _ end: CGPoint) -> Double {
            let deltaX = end.x - start.x, deltaY = end.y - start.y
            let lengthSq = deltaX * deltaX + deltaY * deltaY
            if lengthSq == 0 { return Double(hypot(probe.x - start.x, probe.y - start.y)) }
            let projection = max(0, min(1, ((probe.x - start.x) * deltaX + (probe.y - start.y) * deltaY) / lengthSq))
            return Double(hypot(
                probe.x - (start.x + projection * deltaX),
                probe.y - (start.y + projection * deltaY)
            ))
        }
        switch element {
        case .stroke(_, let width, let pts):
            let hitRadius = radius + width / 2
            if pts.count == 1 { return Double(hypot(point.x - pts[0].x, point.y - pts[0].y)) <= hitRadius }
            for segIndex in 0 ..< max(0, pts.count - 1) where distToSegment(point, pts[segIndex], pts[segIndex + 1]) <= hitRadius {
                return true
            }
            return false
        case .line(_, let width, let x1, let y1, let x2, let y2):
            return distToSegment(point, CGPoint(x: x1, y: y1), CGPoint(x: x2, y: y2)) <= radius + width / 2
        case .rect(_, let width, let rectX, let rectY, let rectWidth, let rectHeight):
            let corners = [
                CGPoint(x: rectX, y: rectY), CGPoint(x: rectX + rectWidth, y: rectY),
                CGPoint(x: rectX + rectWidth, y: rectY + rectHeight), CGPoint(x: rectX, y: rectY + rectHeight),
            ]
            let hitRadius = radius + width / 2
            for cornerIndex in 0 ..< 4 where distToSegment(point, corners[cornerIndex], corners[(cornerIndex + 1) % 4]) <= hitRadius {
                return true
            }
            return false
        case .circle(_, let width, let centerX, let centerY, let radiusX, let radiusY):
            guard radiusX > 0, radiusY > 0 else { return false }
            let normX = (point.x - centerX) / radiusX
            let normY = (point.y - centerY) / radiusY
            let norm = Double(hypot(normX, normY))
            return abs(norm - 1) * min(radiusX, radiusY) <= radius + width / 2
        case .triangle(_, let width, let x1, let y1, let x2, let y2, let x3, let y3):
            let pts = [CGPoint(x: x1, y: y1), CGPoint(x: x2, y: y2), CGPoint(x: x3, y: y3)]
            let hitRadius = radius + width / 2
            for cornerIndex in 0 ..< 3 where distToSegment(point, pts[cornerIndex], pts[(cornerIndex + 1) % 3]) <= hitRadius {
                return true
            }
            return false
        }
    }
}
