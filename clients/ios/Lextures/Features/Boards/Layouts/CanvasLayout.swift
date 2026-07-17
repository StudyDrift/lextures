import SwiftUI

/// Pan/pinch canvas with absolutely positioned cards (VC.M3 FR-5). Move-only in v1.
struct CanvasLayoutView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    @State private var scale: CGFloat = 1
    @State private var offset: CGSize = .zero
    @State private var panStart: CGSize = .zero
    @State private var dragOffsets: [String: CGSize] = [:]
    @State private var didHaptic = false
    @GestureState private var magnify: CGFloat = 1

    private let defaultW: CGFloat = 220
    private let defaultH: CGFloat = 160

    var body: some View {
        if posts.isEmpty {
            BoardPostsEmptyPlaceholder()
        } else {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.boards.layout.canvasHint"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                GeometryReader { geo in
                    let visible = visiblePosts(in: geo.size)
                    ZStack(alignment: .topLeading) {
                        RoundedRectangle(cornerRadius: 12)
                            .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.06))

                        ForEach(Array(visible.enumerated()), id: \.element.id) { index, post in
                            let pos = position(for: post, index: index)
                            let live = dragOffsets[post.id] ?? .zero
                            BoardPostCardSlot(
                                post: post,
                                siblings: posts,
                                sections: sections,
                                board: board,
                                canManage: canManage,
                                currentUserId: currentUserId,
                                onEdit: onEdit,
                                onDelete: onDelete,
                                onArrange: onArrange
                            )
                            .frame(width: CGFloat(pos.w ?? Double(defaultW)))
                            .position(
                                x: CGFloat(pos.x ?? 40) + live.width + offset.width,
                                y: CGFloat(pos.y ?? 40) + live.height + offset.height
                            )
                            .scaleEffect(scale * magnify)
                            .gesture(cardDrag(post: post, base: pos), including: arrangeEnabled(post) ? .gesture : .subviews)
                        }
                    }
                    .frame(width: geo.size.width, height: geo.size.height)
                    .clipped()
                    .gesture(panGesture)
                    .simultaneousGesture(magnificationGesture)
                }
                .frame(minHeight: 420)
                .clipShape(RoundedRectangle(cornerRadius: 12))
            }
        }
    }

    private func arrangeEnabled(_ post: BoardPost) -> Bool {
        BoardsLogic.canArrangePost(
            post: post,
            board: board,
            currentUserId: currentUserId,
            canManage: canManage
        )
    }

    private func position(for post: BoardPost, index: Int) -> BoardPostPosition {
        if let p = post.position { return p }
        let col = index % 3
        let row = index / 3
        return BoardPostPosition(
            x: Double(40 + col * Int(defaultW + 24)),
            y: Double(40 + row * Int(defaultH + 24)),
            w: Double(defaultW),
            h: Double(defaultH)
        )
    }

    /// Virtualize roughly off-screen cards for larger boards.
    private func visiblePosts(in size: CGSize) -> [BoardPost] {
        guard posts.count > 40 else { return posts }
        let margin: CGFloat = 400
        return posts.enumerated().compactMap { index, post in
            let pos = position(for: post, index: index)
            let x = CGFloat(pos.x ?? 0) + offset.width
            let y = CGFloat(pos.y ?? 0) + offset.height
            let w = CGFloat(pos.w ?? Double(defaultW))
            let h = CGFloat(pos.h ?? Double(defaultH))
            let rect = CGRect(x: x, y: y, width: w, height: h)
            let view = CGRect(x: -margin, y: -margin, width: size.width + margin * 2, height: size.height + margin * 2)
            return view.intersects(rect) ? post : nil
        }
    }

    private var panGesture: some Gesture {
        DragGesture()
            .onChanged { value in
                if reduceMotion { return }
                offset = CGSize(
                    width: panStart.width + value.translation.width,
                    height: panStart.height + value.translation.height
                )
            }
            .onEnded { _ in
                panStart = offset
            }
    }

    private var magnificationGesture: some Gesture {
        MagnificationGesture()
            .updating($magnify) { value, state, _ in
                if !reduceMotion { state = value }
            }
            .onEnded { value in
                if reduceMotion { return }
                scale = min(2.5, max(0.5, scale * value))
            }
    }

    private func cardDrag(post: BoardPost, base: BoardPostPosition) -> some Gesture {
        LongPressGesture(minimumDuration: 0.35)
            .sequenced(before: DragGesture())
            .onChanged { value in
                guard arrangeEnabled(post) else { return }
                switch value {
                case .second(true, let drag):
                    if let drag {
                        if !didHaptic {
                            UIImpactFeedbackGenerator(style: .light).impactOccurred()
                            didHaptic = true
                        }
                        dragOffsets[post.id] = drag.translation
                    }
                default:
                    break
                }
            }
            .onEnded { value in
                guard arrangeEnabled(post) else { return }
                defer {
                    dragOffsets[post.id] = nil
                    didHaptic = false
                }
                guard case .second(true, let drag?) = value else { return }
                let next = BoardPostPosition(
                    x: (base.x ?? 40) + Double(drag.translation.width / max(scale, 0.01)),
                    y: (base.y ?? 40) + Double(drag.translation.height / max(scale, 0.01)),
                    w: base.w ?? Double(defaultW),
                    h: base.h ?? Double(defaultH)
                )
                onArrange(post, ArrangeBoardPostInput(position: next))
            }
    }
}
