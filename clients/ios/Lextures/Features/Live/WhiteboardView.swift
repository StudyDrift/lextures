import SwiftUI

struct WhiteboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss
    let course: CourseSummary
    let board: CourseWhiteboard

    @State private var loadedBoard: CourseWhiteboard?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var scale: CGFloat = 1
    @State private var offset: CGSize = .zero

    private var elements: [WhiteboardElement] {
        loadedBoard?.canvasData ?? board.canvasData ?? []
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
                if loading {
                    ProgressView()
                } else if elements.isEmpty {
                    LMSEmptyState(
                        systemImage: "scribble.variable",
                        title: L.text("mobile.live.whiteboard.emptyCanvas"),
                        message: L.text("mobile.live.whiteboard.emptyCanvasHint")
                    )
                } else {
                    whiteboardCanvas
                }
            }
            .navigationTitle(loadedBoard?.title ?? board.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.live.close")) { dismiss() }
                }
            }
            .safeAreaInset(edge: .bottom) {
                Text(L.text("mobile.live.whiteboard.readOnlyNotice"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.ultraThinMaterial)
            }
            .task { await load() }
        }
    }

    private var whiteboardCanvas: some View {
        GeometryReader { proxy in
            Canvas { context, size in
                var transformed = context
                transformed.translateBy(x: offset.width, y: offset.height)
                transformed.scaleBy(x: scale, y: scale)
                WhiteboardRenderer.draw(
                    elements: elements,
                    in: transformed,
                    size: size,
                    isDark: colorScheme == .dark
                )
            }
            .gesture(
                MagnificationGesture()
                    .onChanged { value in scale = max(0.5, min(value, 4)) }
            )
            .simultaneousGesture(
                DragGesture()
                    .onChanged { value in offset = value.translation }
            )
            .frame(width: proxy.size.width, height: max(proxy.size.height, 320))
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .padding(12)
        }
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
    }
}