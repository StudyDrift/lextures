import SwiftUI

/// Date-ordered axis with Undated tray (VC.M3 FR-6).
struct TimelineLayoutView: View {
    @Environment(\.colorScheme) private var colorScheme

    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    private var dated: [BoardPost] { BoardsLogic.datedPosts(posts) }
    private var undated: [BoardPost] { BoardsLogic.undatedPosts(posts) }

    var body: some View {
        if posts.isEmpty {
            BoardPostsEmptyPlaceholder()
        } else {
            VStack(alignment: .leading, spacing: 16) {
                if !dated.isEmpty {
                    VStack(alignment: .leading, spacing: 0) {
                        ForEach(dated) { post in
                            HStack(alignment: .top, spacing: 12) {
                                VStack(spacing: 0) {
                                    Circle()
                                        .fill(Color.accentColor)
                                        .frame(width: 10, height: 10)
                                    Rectangle()
                                        .fill(Color.accentColor.opacity(0.35))
                                        .frame(width: 2)
                                        .frame(maxHeight: .infinity)
                                }
                                VStack(alignment: .leading, spacing: 6) {
                                    Text(formatDate(post.eventDate))
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    BoardPostCardSlot(
                                        post: post,
                                        siblings: dated,
                                        sections: sections,
                                        board: board,
                                        canManage: canManage,
                                        currentUserId: currentUserId,
                                        showTimeline: true,
                                        onEdit: onEdit,
                                        onDelete: onDelete,
                                        onArrange: onArrange
                                    )
                                }
                            }
                            .fixedSize(horizontal: false, vertical: true)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.boards.layout.timeline"))
                }

                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.boards.layout.undatedTray"))
                        .font(.subheadline.weight(.semibold))
                    if undated.isEmpty {
                        Text(L.text("mobile.boards.layout.undatedEmpty"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    } else {
                        ForEach(undated) { post in
                            BoardPostCardSlot(
                                post: post,
                                siblings: undated,
                                sections: sections,
                                board: board,
                                canManage: canManage,
                                currentUserId: currentUserId,
                                showTimeline: true,
                                onEdit: onEdit,
                                onDelete: onDelete,
                                onArrange: onArrange
                            )
                        }
                    }
                }
                .padding(12)
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .strokeBorder(style: StrokeStyle(lineWidth: 1, dash: [6]))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.4))
                )
            }
        }
    }

    private func formatDate(_ raw: String?) -> String {
        guard let raw, !raw.isEmpty else { return "" }
        if let date = ISO8601DateFormatter().date(from: raw) ?? DateFormatting.parse(raw) {
            return date.formatted(date: .abbreviated, time: .omitted)
        }
        return raw
    }
}
