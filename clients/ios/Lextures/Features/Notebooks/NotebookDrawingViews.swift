import SwiftUI

/// Rendered whiteboard drawing in the notebook reading view; tap to edit.
struct NotebookDrawingBlockView: View {
    @Environment(\.colorScheme) private var colorScheme
    let elementsJson: String
    var onTap: () -> Void

    var body: some View {
        let elements = NotebookDrawing.parseElements(elementsJson)
        let content = NotebookDrawing.contentSize(elements)

        Button(action: onTap) {
            GeometryReader { geo in
                let scale = min(geo.size.width / content.width, 1)
                Canvas { context, _ in
                    NotebookDrawing.draw(elements, in: context, scale: scale)
                }
            }
            .aspectRatio(content.width / content.height, contentMode: .fit)
            .frame(maxWidth: .infinity)
            .overlay(alignment: .bottomTrailing) {
                HStack(spacing: 4) {
                    Image(systemName: "pencil.and.outline")
                    Text(elements.isEmpty ? "Tap to draw" : "Edit drawing")
                }
                .font(.caption2.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.85))
                .clipShape(Capsule())
                .padding(8)
            }
            .background(colorScheme == .dark ? Color(white: 0.12) : .white)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Drawing — tap to edit")
    }
}

/// Full-screen whiteboard editor: pen / line / rect / circle / triangle / eraser,
/// the web palette and stroke widths, undo and clear (parity with web `/drawing`).
struct NotebookDrawingEditorView: View {
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme
    let initialElementsJson: String
    var onSave: (String) -> Void

    private enum Tool: String, CaseIterable {
        case pen, line, rect, circle, triangle, eraser

        var icon: String {
            switch self {
            case .pen: "scribble"
            case .line: "line.diagonal"
            case .rect: "rectangle"
            case .circle: "circle"
            case .triangle: "triangle"
            case .eraser: "eraser"
            }
        }
    }

    @State private var elements: [NotebookDrawEl] = []
    @State private var undoStack: [[NotebookDrawEl]] = []
    @State private var tool: Tool = .pen
    @State private var color: String = NotebookDrawing.colors[0]
    @State private var lineWidth: Double = NotebookDrawing.strokeWidths[1]
    @State private var draftElement: NotebookDrawEl?
    @State private var dragStart: CGPoint?
    @State private var loaded = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                canvasArea
                toolbar
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .navigationTitle("Drawing")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Save") {
                        onSave(NotebookDrawing.serializeElements(elements))
                        dismiss()
                    }
                    .fontWeight(.semibold)
                }
            }
            .onAppear {
                guard !loaded else { return }
                loaded = true
                elements = NotebookDrawing.parseElements(initialElementsJson)
            }
        }
    }

    // MARK: - Canvas

    private var canvasArea: some View {
        GeometryReader { geo in
            let content = NotebookDrawing.contentSize(
                elements,
                minSize: CGSize(width: geo.size.width, height: geo.size.height)
            )
            let scale = min(geo.size.width / content.width, geo.size.height / content.height, 1)

            Canvas { context, _ in
                NotebookDrawing.draw(elements, in: context, scale: scale)
                if let draftElement {
                    NotebookDrawing.draw([draftElement], in: context, scale: scale)
                }
            }
            .background(colorScheme == .dark ? Color(white: 0.12) : .white)
            .gesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { value in
                        let point = CGPoint(x: value.location.x / scale, y: value.location.y / scale)
                        handleDrag(to: point)
                    }
                    .onEnded { _ in commitDraft() }
            )
        }
        .padding(12)
    }

    private func handleDrag(to point: CGPoint) {
        if tool == .eraser {
            let radius = max(lineWidth * 2, 12.0)
            let next = elements.filter { !NotebookDrawing.hitTest($0, point: point, radius: radius) }
            if next.count != elements.count {
                pushUndo()
                elements = next
            }
            return
        }
        let start = dragStart ?? point
        if dragStart == nil { dragStart = point }
        switch tool {
        case .pen:
            if case .stroke(let strokeColor, let strokeWidth, var pts) = draftElement {
                pts.append(point)
                draftElement = .stroke(color: strokeColor, width: strokeWidth, pts: pts)
            } else {
                draftElement = .stroke(color: color, width: lineWidth, pts: [point])
            }
        case .line:
            draftElement = .line(color: color, width: lineWidth, x1: start.x, y1: start.y, x2: point.x, y2: point.y)
        case .rect:
            draftElement = .rect(
                color: color, width: lineWidth,
                rectX: min(start.x, point.x), rectY: min(start.y, point.y),
                rectWidth: abs(point.x - start.x), rectHeight: abs(point.y - start.y)
            )
        case .circle:
            draftElement = .circle(
                color: color, width: lineWidth,
                cx: (start.x + point.x) / 2, cy: (start.y + point.y) / 2,
                rx: abs(point.x - start.x) / 2, ry: abs(point.y - start.y) / 2
            )
        case .triangle:
            draftElement = .triangle(
                color: color, width: lineWidth,
                x1: (start.x + point.x) / 2, y1: min(start.y, point.y),
                x2: start.x, y2: max(start.y, point.y),
                x3: point.x, y3: max(start.y, point.y)
            )
        case .eraser:
            break
        }
    }

    private func commitDraft() {
        dragStart = nil
        guard let draftElement else { return }
        pushUndo()
        elements.append(draftElement)
        self.draftElement = nil
    }

    private func pushUndo() {
        undoStack.append(elements)
        if undoStack.count > 50 { undoStack.removeFirst() }
    }

    // MARK: - Toolbar

    private var toolbar: some View {
        VStack(spacing: 8) {
            HStack(spacing: 6) {
                ForEach(Tool.allCases, id: \.rawValue) { item in
                    toolButton(item)
                }
                Spacer()
                Button {
                    if let last = undoStack.popLast() { elements = last }
                } label: {
                    Image(systemName: "arrow.uturn.backward")
                        .frame(width: 34, height: 34)
                }
                .disabled(undoStack.isEmpty)
                Button(role: .destructive) {
                    guard !elements.isEmpty else { return }
                    pushUndo()
                    elements = []
                } label: {
                    Image(systemName: "trash")
                        .frame(width: 34, height: 34)
                }
                .disabled(elements.isEmpty)
            }

            HStack(spacing: 8) {
                ForEach(NotebookDrawing.colors, id: \.self) { hex in
                    Button {
                        color = hex
                    } label: {
                        Circle()
                            .fill(NotebookDrawing.color(from: hex))
                            .frame(width: 24, height: 24)
                            .overlay(
                                Circle().stroke(
                                    color == hex
                                        ? LexturesTheme.accent(for: colorScheme)
                                        : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: color == hex ? 2.5 : 1
                                )
                            )
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Color \(hex)")
                }
                Spacer()
                ForEach(NotebookDrawing.strokeWidths, id: \.self) { strokeWidth in
                    Button {
                        lineWidth = strokeWidth
                    } label: {
                        Circle()
                            .fill(LexturesTheme.textPrimary(for: colorScheme))
                            .frame(width: 6 + strokeWidth * 1.5, height: 6 + strokeWidth * 1.5)
                            .frame(width: 28, height: 28)
                            .background(
                                Circle().fill(
                                    lineWidth == strokeWidth
                                        ? LexturesTheme.accent(for: colorScheme).opacity(0.18)
                                        : .clear
                                )
                            )
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Stroke width \(Int(strokeWidth))")
                }
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background(LexturesTheme.cardBackground(for: colorScheme))
    }

    private func toolButton(_ item: Tool) -> some View {
        Button {
            tool = item
        } label: {
            Image(systemName: item.icon)
                .font(.subheadline)
                .foregroundStyle(
                    tool == item ? .white : LexturesTheme.textPrimary(for: colorScheme)
                )
                .frame(width: 34, height: 34)
                .background(
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(tool == item ? LexturesTheme.accent(for: colorScheme) : .clear)
                )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(item.rawValue)
    }
}
