import SwiftUI

/// Simple horizontal flow for citation chips.
struct TutorFlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let width = proposal.width ?? 0
        var offsetX: CGFloat = 0
        var offsetY: CGFloat = 0
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if offsetX + size.width > width, offsetX > 0 {
                offsetX = 0
                offsetY += rowHeight + spacing
                rowHeight = 0
            }
            rowHeight = max(rowHeight, size.height)
            offsetX += size.width + spacing
        }
        return CGSize(width: width, height: offsetY + rowHeight)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var offsetX = bounds.minX
        var offsetY = bounds.minY
        var rowHeight: CGFloat = 0
        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if offsetX + size.width > bounds.maxX, offsetX > bounds.minX {
                offsetX = bounds.minX
                offsetY += rowHeight + spacing
                rowHeight = 0
            }
            subview.place(at: CGPoint(x: offsetX, y: offsetY), proposal: ProposedViewSize(size))
            rowHeight = max(rowHeight, size.height)
            offsetX += size.width + spacing
        }
    }
}