import Foundation
import SwiftUI

/// One whiteboard element from a ```drawing fenced block (parity with web `whiteboard/types`).
enum NotebookDrawEl: Equatable {
    case stroke(color: String, width: Double, pts: [CGPoint])
    case rect(color: String, width: Double, x: Double, y: Double, w: Double, h: Double)
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
                guard pair.count >= 2, let x = num(pair[0]), let y = num(pair[1]) else { return nil }
                return CGPoint(x: x, y: y)
            }
            return .stroke(color: color, width: width, pts: pts)
        case "rect":
            guard let x = num(obj["x"]), let y = num(obj["y"]), let w = num(obj["w"]), let h = num(obj["h"]) else { return nil }
            return .rect(color: color, width: width, x: x, y: y, w: w, h: h)
        case "circle":
            guard let cx = num(obj["cx"]), let cy = num(obj["cy"]), let rx = num(obj["rx"]), let ry = num(obj["ry"]) else { return nil }
            return .circle(color: color, width: width, cx: cx, cy: cy, rx: rx, ry: ry)
        case "triangle":
            guard
                let x1 = num(obj["x1"]), let y1 = num(obj["y1"]),
                let x2 = num(obj["x2"]), let y2 = num(obj["y2"]),
                let x3 = num(obj["x3"]), let y3 = num(obj["y3"])
            else { return nil }
            return .triangle(color: color, width: width, x1: x1, y1: y1, x2: x2, y2: y2, x3: x3, y3: y3)
        case "line":
            guard let x1 = num(obj["x1"]), let y1 = num(obj["y1"]), let x2 = num(obj["x2"]), let y2 = num(obj["y2"]) else { return nil }
            return .line(color: color, width: width, x1: x1, y1: y1, x2: x2, y2: y2)
        default:
            return nil
        }
    }

    static func serializeElements(_ elements: [NotebookDrawEl]) -> String {
        let array: [[String: Any]] = elements.map { el in
            switch el {
            case .stroke(let color, let width, let pts):
                return ["type": "stroke", "color": color, "width": width, "pts": pts.map { [$0.x, $0.y] }]
            case .rect(let color, let width, let x, let y, let w, let h):
                return ["type": "rect", "color": color, "width": width, "x": x, "y": y, "w": w, "h": h]
            case .circle(let color, let width, let cx, let cy, let rx, let ry):
                return ["type": "circle", "color": color, "width": width, "cx": cx, "cy": cy, "rx": rx, "ry": ry]
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
        func grow(_ x: Double, _ y: Double) {
            maxX = max(maxX, x)
            maxY = max(maxY, y)
        }
        for el in elements {
            switch el {
            case .stroke(_, _, let pts):
                for p in pts { grow(p.x, p.y) }
            case .rect(_, _, let x, let y, let w, let h):
                grow(x + w, y + h)
                grow(x, y)
            case .circle(_, _, let cx, let cy, let rx, let ry):
                grow(cx + rx, cy + ry)
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
        for el in elements {
            switch el {
            case .stroke(let color, let width, let pts):
                guard pts.count > 1 else {
                    if let p = pts.first {
                        let r = max(width, 2) * scale / 2
                        let dot = Path(ellipseIn: CGRect(x: p.x * scale - r, y: p.y * scale - r, width: r * 2, height: r * 2))
                        context.fill(dot, with: .color(NotebookDrawing.color(from: color)))
                    }
                    continue
                }
                var path = Path()
                path.move(to: CGPoint(x: pts[0].x * scale, y: pts[0].y * scale))
                for p in pts.dropFirst() {
                    path.addLine(to: CGPoint(x: p.x * scale, y: p.y * scale))
                }
                context.stroke(
                    path,
                    with: .color(NotebookDrawing.color(from: color)),
                    style: StrokeStyle(lineWidth: width * scale, lineCap: .round, lineJoin: .round)
                )
            case .rect(let color, let width, let x, let y, let w, let h):
                let rect = CGRect(x: x * scale, y: y * scale, width: w * scale, height: h * scale)
                context.stroke(Path(rect), with: .color(NotebookDrawing.color(from: color)), lineWidth: width * scale)
            case .circle(let color, let width, let cx, let cy, let rx, let ry):
                let rect = CGRect(x: (cx - rx) * scale, y: (cy - ry) * scale, width: rx * 2 * scale, height: ry * 2 * scale)
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
    static func hitTest(_ el: NotebookDrawEl, point: CGPoint, radius: Double) -> Bool {
        func distToSegment(_ p: CGPoint, _ a: CGPoint, _ b: CGPoint) -> Double {
            let dx = b.x - a.x, dy = b.y - a.y
            let lengthSq = dx * dx + dy * dy
            if lengthSq == 0 { return Double(hypot(p.x - a.x, p.y - a.y)) }
            let t = max(0, min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / lengthSq))
            return Double(hypot(p.x - (a.x + t * dx), p.y - (a.y + t * dy)))
        }
        switch el {
        case .stroke(_, let width, let pts):
            let r = radius + width / 2
            if pts.count == 1 { return Double(hypot(point.x - pts[0].x, point.y - pts[0].y)) <= r }
            for i in 0 ..< max(0, pts.count - 1) where distToSegment(point, pts[i], pts[i + 1]) <= r {
                return true
            }
            return false
        case .line(_, let width, let x1, let y1, let x2, let y2):
            return distToSegment(point, CGPoint(x: x1, y: y1), CGPoint(x: x2, y: y2)) <= radius + width / 2
        case .rect(_, let width, let x, let y, let w, let h):
            let corners = [
                CGPoint(x: x, y: y), CGPoint(x: x + w, y: y),
                CGPoint(x: x + w, y: y + h), CGPoint(x: x, y: y + h),
            ]
            let r = radius + width / 2
            for i in 0 ..< 4 where distToSegment(point, corners[i], corners[(i + 1) % 4]) <= r {
                return true
            }
            return false
        case .circle(_, let width, let cx, let cy, let rx, let ry):
            guard rx > 0, ry > 0 else { return false }
            let nx = (point.x - cx) / rx
            let ny = (point.y - cy) / ry
            let norm = Double(hypot(nx, ny))
            return abs(norm - 1) * min(rx, ry) <= radius + width / 2
        case .triangle(_, let width, let x1, let y1, let x2, let y2, let x3, let y3):
            let pts = [CGPoint(x: x1, y: y1), CGPoint(x: x2, y: y2), CGPoint(x: x3, y: y3)]
            let r = radius + width / 2
            for i in 0 ..< 3 where distToSegment(point, pts[i], pts[(i + 1) % 3]) <= r {
                return true
            }
            return false
        }
    }
}
