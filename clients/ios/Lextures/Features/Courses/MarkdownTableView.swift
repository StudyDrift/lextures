import SwiftUI

/// Inline GFM table with horizontal scroll and expand affordance (CT.M1 FR-5 / FR-6).
/// `suppressExpand` hides the full-screen escape hatch during lockdown quizzes (CT.M2 FR-13).
struct MarkdownTableView: View {
    @Environment(\.colorScheme) private var colorScheme
    let align: [MarkdownTableAlign]
    let header: [String]
    let rows: [[String]]
    var suppressExpand = false
    @State private var showFullScreen = false

    private let minColumnWidth: CGFloat = 96

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            ScrollView(.horizontal, showsIndicators: true) {
                tableGrid(minWidth: minColumnWidth)
            }
            if MarkdownRenderStyle.allowsAffordances(suppressAffordances: suppressExpand) {
                Button {
                    showFullScreen = true
                } label: {
                    Label(L.text("mobile.markdown.table.expand"), systemImage: "arrow.up.left.and.arrow.down.right")
                        .font(.caption.weight(.semibold))
                }
                .buttonStyle(.plain)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .accessibilityHint(L.text("mobile.markdown.table.expandHint"))
            }
        }
        .fullScreenCover(isPresented: $showFullScreen) {
            MarkdownTableFullScreenView(align: align, header: header, rows: rows)
        }
    }

    @ViewBuilder
    private func tableGrid(minWidth: CGFloat) -> some View {
        Grid(alignment: .leading, horizontalSpacing: 0, verticalSpacing: 0) {
            GridRow {
                ForEach(Array(header.enumerated()), id: \.offset) { index, cell in
                    cellView(cell, headerColumn: header[index], rowHeader: nil, isHeader: true, align: alignAt(index))
                        .frame(minWidth: minWidth, alignment: alignment(for: alignAt(index)))
                }
            }
            ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                GridRow {
                    ForEach(Array(row.enumerated()), id: \.offset) { index, cell in
                        let columnHeader = index < header.count ? header[index] : ""
                        let rowHeader = row.first
                        cellView(cell, headerColumn: columnHeader, rowHeader: rowHeader, isHeader: false, align: alignAt(index))
                            .frame(minWidth: minWidth, alignment: alignment(for: alignAt(index)))
                    }
                }
            }
        }
        .overlay {
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .stroke(LexturesTheme.textSecondary(for: colorScheme).opacity(0.35), lineWidth: 1)
        }
        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        .accessibilityElement(children: .contain)
    }

    private func cellView(
        _ text: String,
        headerColumn: String,
        rowHeader: String?,
        isHeader: Bool,
        align: MarkdownTableAlign
    ) -> some View {
        Text(inline(text))
            .font(isHeader ? .caption.weight(.semibold) : .caption)
            .multilineTextAlignment(textAlignment(for: align))
            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
            .frame(maxWidth: .infinity, alignment: alignment(for: align))
            .background(isHeader ? LexturesTheme.sceneBackground(for: colorScheme) : Color.clear)
            .accessibilityLabel(cellAccessibilityLabel(column: headerColumn, row: rowHeader, value: text, isHeader: isHeader))
    }

    private func cellAccessibilityLabel(column: String, row: String?, value: String, isHeader: Bool) -> String {
        if isHeader { return value }
        if let row, !row.isEmpty, row != value {
            return "\(column), \(row): \(value)"
        }
        return "\(column): \(value)"
    }

    private func alignAt(_ index: Int) -> MarkdownTableAlign {
        index < align.count ? align[index] : .default
    }

    private func alignment(for align: MarkdownTableAlign) -> Alignment {
        switch align {
        case .center: return .center
        case .right: return .trailing
        case .left, .default: return .leading
        }
    }

    private func textAlignment(for align: MarkdownTableAlign) -> TextAlignment {
        switch align {
        case .center: return .center
        case .right: return .trailing
        case .left, .default: return .leading
        }
    }

    private func inline(_ text: String) -> AttributedString {
        (try? AttributedString(
            markdown: text,
            options: .init(interpretedSyntax: .inlineOnlyPreservingWhitespace)
        )) ?? AttributedString(text)
    }
}

/// Full-screen table viewer with pinned header (CT.M1 FR-6).
struct MarkdownTableFullScreenView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    let align: [MarkdownTableAlign]
    let header: [String]
    let rows: [[String]]

    private let minColumnWidth: CGFloat = 120

    var body: some View {
        NavigationStack {
            ScrollView([.horizontal, .vertical], showsIndicators: true) {
                Grid(alignment: .leading, horizontalSpacing: 0, verticalSpacing: 0) {
                    GridRow {
                        ForEach(Array(header.enumerated()), id: \.offset) { _, cell in
                            Text(cell)
                                .font(.subheadline.weight(.semibold))
                                .padding(12)
                                .frame(minWidth: minColumnWidth, alignment: .leading)
                                .background(LexturesTheme.sceneBackground(for: colorScheme))
                                .accessibilityAddTraits(.isHeader)
                        }
                    }
                    ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                        GridRow {
                            ForEach(Array(row.enumerated()), id: \.offset) { index, cell in
                                let column = index < header.count ? header[index] : ""
                                Text(cell)
                                    .font(.subheadline)
                                    .padding(12)
                                    .frame(minWidth: minColumnWidth, alignment: .leading)
                                    .accessibilityLabel("\(column): \(cell)")
                            }
                        }
                    }
                }
                .padding()
            }
            .navigationTitle(L.text("mobile.markdown.table.expand"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.markdown.table.close")) { dismiss() }
                }
            }
        }
    }
}
