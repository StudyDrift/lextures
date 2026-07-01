import SwiftUI

/// Read-only overlay for instructor markups on PDF/image previews (M6.1).
struct MarkupOverlayView: View {
    let annotations: [SubmissionAnnotation]
    var page: Int = 1

    var body: some View {
        GeometryReader { geo in
            let pageAnnotations = annotations.filter { $0.page == page }
            ForEach(pageAnnotations) { annotation in
                annotationShape(annotation, in: geo.size)
            }
        }
        .allowsHitTesting(false)
        .accessibilityHidden(true)
    }

    @ViewBuilder
    private func annotationShape(_ annotation: SubmissionAnnotation, in size: CGSize) -> some View {
        let color = Color(hex: annotation.colour) ?? .yellow
        switch annotation.toolType {
        case "highlight":
            ForEach(Array(highlightRects(annotation).enumerated()), id: \.offset) { _, rect in
                Rectangle()
                    .fill(color.opacity(0.35))
                    .frame(
                        width: max(1, (rect.x2 - rect.x1) * size.width),
                        height: max(1, (rect.y2 - rect.y1) * size.height)
                    )
                    .position(
                        x: (rect.x1 + rect.x2) / 2 * size.width,
                        y: (rect.y1 + rect.y2) / 2 * size.height
                    )
            }
        case "draw":
            if let points = annotation.coordsJson?.points, points.count >= 2 {
                Path { path in
                    path.move(to: CGPoint(x: points[0].coordX * size.width, y: points[0].coordY * size.height))
                    for point in points.dropFirst() {
                        path.addLine(to: CGPoint(x: point.coordX * size.width, y: point.coordY * size.height))
                    }
                }
                .stroke(color, lineWidth: 2)
            }
        case "pin", "text":
            if let coords = annotation.coordsJson,
               let pinX = coords.pinX ?? coords.x1,
               let pinY = coords.pinY ?? coords.y1 {
                Circle()
                    .fill(color)
                    .frame(width: 10, height: 10)
                    .position(x: pinX * size.width, y: pinY * size.height)
            }
        default:
            EmptyView()
        }
    }

    private func highlightRects(_ annotation: SubmissionAnnotation) -> [AnnotationRect] {
        guard let coords = annotation.coordsJson else { return [] }
        if let rects = coords.rects, !rects.isEmpty { return rects }
        if let x1 = coords.x1, let y1 = coords.y1, let x2 = coords.x2, let y2 = coords.y2 {
            return [AnnotationRect(x1: x1, y1: y1, x2: x2, y2: y2)]
        }
        return []
    }
}

private extension Color {
    init?(hex: String) {
        var cleaned = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        if cleaned.hasPrefix("#") { cleaned.removeFirst() }
        guard cleaned.count == 6, let value = UInt64(cleaned, radix: 16) else { return nil }
        let red = Double((value >> 16) & 0xFF) / 255
        let green = Double((value >> 8) & 0xFF) / 255
        let blue = Double(value & 0xFF) / 255
        self.init(red: red, green: green, blue: blue)
    }
}
