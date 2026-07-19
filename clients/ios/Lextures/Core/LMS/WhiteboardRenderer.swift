import SwiftUI

/// Read-only whiteboard canvas renderer (M7.5) — mirrors web `DrawEl` shapes.
enum WhiteboardRenderer {
    static func draw(elements: [WhiteboardElement], in context: GraphicsContext, size: CGSize, isDark: Bool) {
        drawGrid(in: context, size: size, isDark: isDark)
        for element in elements {
            draw(element: element, in: context)
        }
    }

    private static func drawGrid(in context: GraphicsContext, size: CGSize, isDark: Bool) {
        let background = isDark ? Color(red: 0.09, green: 0.09, blue: 0.09) : .white
        context.fill(Path(CGRect(origin: .zero, size: size)), with: .color(background))
        let dotColor = isDark ? Color.white.opacity(0.18) : Color.black.opacity(0.18)
        let spacing: CGFloat = 24
        var gridX = spacing
        while gridX < size.width {
            var gridY = spacing
            while gridY < size.height {
                let rect = CGRect(x: gridX - 1, y: gridY - 1, width: 2, height: 2)
                context.fill(Path(ellipseIn: rect), with: .color(dotColor))
                gridY += spacing
            }
            gridX += spacing
        }
    }

    private static func draw(element: WhiteboardElement, in context: GraphicsContext) {
        guard let color = Color(hex: element.color) else { return }
        var stroke = StrokeStyle(lineWidth: CGFloat(element.width), lineCap: .round, lineJoin: .round)
        switch element.type {
        case "stroke":
            guard element.points.count >= 2 else { return }
            var path = Path()
            path.move(to: element.points[0])
            for point in element.points.dropFirst() {
                path.addLine(to: point)
            }
            context.stroke(path, with: .color(color), style: stroke)
        case "rect":
            guard let rect = element.rect else { return }
            context.stroke(Path(rect), with: .color(color), style: stroke)
        case "circle":
            guard let ellipse = element.ellipse else { return }
            context.stroke(Path(ellipseIn: ellipse), with: .color(color), style: stroke)
        case "triangle":
            guard element.points.count == 3 else { return }
            var path = Path()
            path.move(to: element.points[0])
            path.addLine(to: element.points[1])
            path.addLine(to: element.points[2])
            path.closeSubpath()
            context.stroke(path, with: .color(color), style: stroke)
        case "line":
            guard element.points.count == 2 else { return }
            var path = Path()
            path.move(to: element.points[0])
            path.addLine(to: element.points[1])
            context.stroke(path, with: .color(color), style: stroke)
        default:
            break
        }
    }
}

struct WhiteboardElement: Codable, Hashable {
    var type: String
    var color: String
    var width: Double
    var pts: [[Double]]? = nil
    var rectX: Double? = nil
    var rectY: Double? = nil
    var rectWidth: Double? = nil
    var rectHeight: Double? = nil
    var centerX: Double? = nil
    var centerY: Double? = nil
    var radiusX: Double? = nil
    var radiusY: Double? = nil
    var point1X: Double? = nil
    var point1Y: Double? = nil
    var point2X: Double? = nil
    var point2Y: Double? = nil
    var point3X: Double? = nil
    var point3Y: Double? = nil

    enum CodingKeys: String, CodingKey {
        case type, color, width, pts
        case rectX = "x"
        case rectY = "y"
        case rectWidth = "w"
        case rectHeight = "h"
        case centerX = "cx"
        case centerY = "cy"
        case radiusX = "rx"
        case radiusY = "ry"
        case point1X = "x1"
        case point1Y = "y1"
        case point2X = "x2"
        case point2Y = "y2"
        case point3X = "x3"
        case point3Y = "y3"
    }

    var points: [CGPoint] {
        if let pts {
            return pts.compactMap { pair in
                guard pair.count >= 2 else { return nil }
                return CGPoint(x: pair[0], y: pair[1])
            }
        }
        if let point1X, let point1Y, let point2X, let point2Y, let point3X, let point3Y {
            return [CGPoint(x: point1X, y: point1Y), CGPoint(x: point2X, y: point2Y), CGPoint(x: point3X, y: point3Y)]
        }
        if let point1X, let point1Y, let point2X, let point2Y {
            return [CGPoint(x: point1X, y: point1Y), CGPoint(x: point2X, y: point2Y)]
        }
        return []
    }

    var rect: CGRect? {
        guard let rectX, let rectY, let rectWidth, let rectHeight else { return nil }
        return CGRect(x: rectX, y: rectY, width: rectWidth, height: rectHeight)
    }

    var ellipse: CGRect? {
        guard let centerX, let centerY, let radiusX, let radiusY else { return nil }
        return CGRect(x: centerX - abs(radiusX), y: centerY - abs(radiusY), width: abs(radiusX) * 2, height: abs(radiusY) * 2)
    }
}

private extension Color {
    init?(hex: String) {
        var raw = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        if raw.hasPrefix("#") { raw.removeFirst() }
        guard raw.count == 6, let value = UInt64(raw, radix: 16) else { return nil }
        let red = Double((value >> 16) & 0xFF) / 255
        let green = Double((value >> 8) & 0xFF) / 255
        let blue = Double(value & 0xFF) / 255
        self.init(red: red, green: green, blue: blue)
    }
}
