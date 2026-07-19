import SwiftUI

struct WhiteboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    let course: CourseSummary
    let board: CourseWhiteboard
    var canEdit: Bool = false
    var onDeleted: (() -> Void)? = nil

    @State private var loadedBoard: CourseWhiteboard?
    @State private var elements: [WhiteboardElement] = []
    @State private var history = WhiteboardLogic.History()
    @State private var tool: WhiteboardTool = .pen
    @State private var color: String = WhiteboardLogic.colors[0]
    @State private var strokeWidth: Double = WhiteboardLogic.strokeWidths[1]
    @State private var eraserSize: Double = WhiteboardLogic.eraserSizes[1]
    @State private var draftElement: WhiteboardElement?
    @State private var dragStart: CGPoint?
    @State private var selectedIdx: Int?
    @State private var selectDragStart: CGPoint?
    @State private var selectOrigElement: WhiteboardElement?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var saveState: SaveState = .idle
    @State private var scale: CGFloat = 1
    @State private var offset: CGSize = .zero
    @State private var autosaveTask: Task<Void, Never>?

    private enum SaveState: Equatable {
        case idle
        case saving
        case saved
        case failed
    }

    private var title: String {
        loadedBoard?.title ?? board.title
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                if loading {
                    ProgressView()
                } else if elements.isEmpty && !canEdit {
                    LMSEmptyState(
                        systemImage: "scribble.variable",
                        title: L.text("mobile.live.whiteboard.emptyCanvas"),
                        message: L.text("mobile.live.whiteboard.emptyCanvasHint")
                    )
                } else {
                    whiteboardCanvas
                }
            }
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.live.close")) { dismiss() }
                }
                if canEdit {
                    ToolbarItem(placement: .topBarTrailing) {
                        Menu {
                            Button(L.text("mobile.whiteboard.save")) {
                                Task { await save() }
                            }
                            Button(L.text("mobile.whiteboard.delete"), role: .destructive) {
                                Task { await deleteBoard() }
                            }
                        } label: {
                            Image(systemName: "ellipsis.circle")
                        }
                        .accessibilityLabel(L.text("mobile.whiteboard.actions"))
                    }
                }
            }
            .safeAreaInset(edge: .bottom) {
                if canEdit {
                    editorChrome
                } else {
                    Text(L.text("mobile.live.whiteboard.readOnlyNotice"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                        .background(.ultraThinMaterial)
                }
            }
            .task { await load() }
            .onDisappear { autosaveTask?.cancel() }
        }
    }

    private var editorChrome: some View {
        VStack(spacing: 6) {
            if let errorMessage {
                Text(errorMessage)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            HStack {
                Text(saveLabel)
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Spacer()
                if saveState == .failed {
                    Button(L.text("mobile.whiteboard.retry")) {
                        Task { await save() }
                    }
                    .font(.caption2)
                }
            }
            .padding(.horizontal, 14)

            toolbar
        }
        .padding(.vertical, 8)
        .background(.ultraThinMaterial)
    }

    private var saveLabel: String {
        switch saveState {
        case .idle: return L.text("mobile.whiteboard.status.idle")
        case .saving: return L.text("mobile.whiteboard.status.saving")
        case .saved: return L.text("mobile.whiteboard.status.saved")
        case .failed: return L.text("mobile.whiteboard.status.failed")
        }
    }

    private var whiteboardCanvas: some View {
        GeometryReader { proxy in
            let canvas = Canvas { context, size in
                var transformed = context
                transformed.translateBy(x: offset.width, y: offset.height)
                transformed.scaleBy(x: scale, y: scale)
                WhiteboardRenderer.draw(
                    elements: elements,
                    in: transformed,
                    size: size,
                    isDark: colorScheme == .dark
                )
                if let draftElement {
                    WhiteboardRenderer.draw(
                        elements: [draftElement],
                        in: transformed,
                        size: size,
                        isDark: colorScheme == .dark
                    )
                }
            }
            Group {
                if canEdit {
                    canvas
                        .gesture(
                            DragGesture(minimumDistance: 0)
                                .onChanged { value in
                                    handleDrag(to: canvasPoint(from: value.location))
                                }
                                .onEnded { _ in
                                    commitDraft()
                                    scheduleAutosave()
                                }
                        )
                } else {
                    canvas
                        .gesture(
                            DragGesture().onChanged { value in offset = value.translation }
                        )
                }
            }
            .simultaneousGesture(
                MagnificationGesture()
                    .onChanged { value in scale = max(0.5, min(value, 4)) }
            )
            .frame(width: proxy.size.width, height: max(proxy.size.height, 320))
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .padding(12)
            .accessibilityLabel(L.text("mobile.whiteboard.canvas"))
        }
    }

    private func canvasPoint(from location: CGPoint) -> CGPoint {
        CGPoint(
            x: (location.x - offset.width) / scale,
            y: (location.y - offset.height) / scale
        )
    }

    private func handleDrag(to point: CGPoint) {
        switch tool {
        case .select:
            if selectDragStart == nil {
                selectedIdx = WhiteboardLogic.pickElement(elements, at: point)
                if let selectedIdx {
                    selectDragStart = point
                    selectOrigElement = elements[selectedIdx]
                    history.push(elements)
                }
                return
            }
            guard let selectedIdx, let start = selectDragStart, let orig = selectOrigElement else { return }
            let dx = Double(point.x - start.x)
            let dy = Double(point.y - start.y)
            var next = elements
            next[selectedIdx] = WhiteboardLogic.translate(orig, dx: dx, dy: dy)
            elements = next
            WhiteboardObservability.record("whiteboard_edited", attributes: ["tool": "select"])
        case .eraser:
            let next = WhiteboardLogic.erase(from: elements, at: point, radius: eraserSize)
            if next.count != elements.count {
                history.push(elements)
                elements = next
                WhiteboardObservability.record("whiteboard_edited", attributes: ["tool": "eraser"])
            }
        case .pen:
            if var draft = draftElement, draft.type == "stroke", var pts = draft.pts {
                pts.append([Double(point.x), Double(point.y)])
                draft.pts = pts
                draftElement = draft
            } else {
                dragStart = point
                draftElement = WhiteboardLogic.stroke(color: color, width: strokeWidth, pts: [point])
            }
        case .line, .rect, .circle, .triangle:
            let start = dragStart ?? point
            if dragStart == nil { dragStart = point }
            switch tool {
            case .line:
                draftElement = WhiteboardLogic.line(color: color, width: strokeWidth, from: start, to: point)
            case .rect:
                draftElement = WhiteboardLogic.rect(color: color, width: strokeWidth, from: start, to: point)
            case .circle:
                draftElement = WhiteboardLogic.circle(color: color, width: strokeWidth, from: start, to: point)
            case .triangle:
                draftElement = WhiteboardLogic.triangle(color: color, width: strokeWidth, from: start, to: point)
            default:
                break
            }
        }
    }

    private func commitDraft() {
        defer {
            dragStart = nil
            selectDragStart = nil
            selectOrigElement = nil
        }
        guard tool != .select, tool != .eraser, let draftElement else { return }
        history.push(elements)
        elements.append(draftElement)
        self.draftElement = nil
        WhiteboardObservability.record("whiteboard_edited", attributes: ["tool": tool.rawValue])
    }

    private var toolbar: some View {
        VStack(spacing: 8) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 6) {
                    ForEach(WhiteboardTool.allCases, id: \.rawValue) { item in
                        toolButton(item)
                    }
                    Button {
                        if let previous = history.undo(current: elements) {
                            elements = previous
                            WhiteboardObservability.record("whiteboard_undo")
                            scheduleAutosave()
                        }
                    } label: {
                        Image(systemName: "arrow.uturn.backward")
                            .frame(width: 44, height: 44)
                    }
                    .disabled(!history.canUndo)
                    .accessibilityLabel(L.text("mobile.whiteboard.undo"))

                    Button {
                        if let next = history.redo(current: elements) {
                            elements = next
                            scheduleAutosave()
                        }
                    } label: {
                        Image(systemName: "arrow.uturn.forward")
                            .frame(width: 44, height: 44)
                    }
                    .disabled(!history.canRedo)
                    .accessibilityLabel(L.text("mobile.whiteboard.redo"))
                }
                .padding(.horizontal, 14)
            }

            HStack(spacing: 8) {
                ForEach(WhiteboardLogic.colors, id: \.self) { hex in
                    Button { color = hex } label: {
                        Circle()
                            .fill(colorFromHex(hex))
                            .frame(width: 28, height: 28)
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
                    .accessibilityLabel(L.format("mobile.whiteboard.color", hex))
                }
                Spacer()
                ForEach(tool == .eraser ? WhiteboardLogic.eraserSizes : WhiteboardLogic.strokeWidths, id: \.self) { width in
                    Button {
                        if tool == .eraser { eraserSize = width } else { strokeWidth = width }
                    } label: {
                        Circle()
                            .fill(LexturesTheme.textPrimary(for: colorScheme))
                            .frame(width: 6 + width * 1.5, height: 6 + width * 1.5)
                            .frame(width: 44, height: 44)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(L.format("mobile.whiteboard.strokeWidth", "\(Int(width))"))
                }
            }
            .padding(.horizontal, 14)
        }
    }

    private func toolButton(_ item: WhiteboardTool) -> some View {
        Button { tool = item } label: {
            Image(systemName: toolIcon(item))
                .font(.subheadline)
                .foregroundStyle(tool == item ? .white : LexturesTheme.textPrimary(for: colorScheme))
                .frame(width: 44, height: 44)
                .background(
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(tool == item ? LexturesTheme.accent(for: colorScheme) : .clear)
                )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text(String.LocalizationValue(item.accessibilityKey)))
    }

    private func toolIcon(_ item: WhiteboardTool) -> String {
        switch item {
        case .select: return "arrow.up.left.and.arrow.down.right"
        case .pen: return "scribble"
        case .line: return "line.diagonal"
        case .rect: return "rectangle"
        case .circle: return "circle"
        case .triangle: return "triangle"
        case .eraser: return "eraser"
        }
    }

    private func colorFromHex(_ hex: String) -> Color {
        var raw = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        if raw.hasPrefix("#") { raw.removeFirst() }
        guard raw.count == 6, let value = UInt64(raw, radix: 16) else { return .primary }
        return Color(
            red: Double((value >> 16) & 0xFF) / 255,
            green: Double((value >> 8) & 0xFF) / 255,
            blue: Double(value & 0xFF) / 255
        )
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            loadedBoard = try await LMSAPI.fetchCourseWhiteboard(
                courseCode: course.courseCode,
                boardId: board.id,
                accessToken: token
            )
        } catch {
            loadedBoard = board
            errorMessage = L.text("mobile.live.whiteboard.error.load")
        }
        elements = loadedBoard?.canvasData ?? board.canvasData ?? []
    }

    private func scheduleAutosave() {
        guard canEdit else { return }
        autosaveTask?.cancel()
        autosaveTask = Task {
            try? await Task.sleep(nanoseconds: WhiteboardLogic.autosaveDelayMs * 1_000_000)
            guard !Task.isCancelled else { return }
            await save()
        }
    }

    private func save() async {
        guard canEdit, let token = session.accessToken else { return }
        saveState = .saving
        errorMessage = nil
        do {
            let saved = try await LMSAPI.updateCourseWhiteboard(
                courseCode: course.courseCode,
                boardId: board.id,
                title: title,
                canvasData: elements,
                accessToken: token
            )
            loadedBoard = saved
            elements = saved.canvasData ?? elements
            saveState = .saved
        } catch {
            saveState = .failed
            errorMessage = L.text("mobile.whiteboard.error.save")
        }
    }

    private func deleteBoard() async {
        guard canEdit, let token = session.accessToken else { return }
        do {
            try await LMSAPI.deleteCourseWhiteboard(
                courseCode: course.courseCode,
                boardId: board.id,
                accessToken: token
            )
            onDeleted?()
            dismiss()
        } catch {
            errorMessage = L.text("mobile.whiteboard.error.delete")
        }
    }
}
