import SwiftUI

/// Two-column masonry-style wall (VC.M3 FR-1).
struct WallLayoutView: View {
    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    private var columns: [GridItem] {
        [GridItem(.flexible(), spacing: 12), GridItem(.flexible(), spacing: 12)]
    }

    var body: some View {
        if posts.isEmpty {
            BoardPostsEmptyPlaceholder()
        } else {
            LazyVGrid(columns: columns, spacing: 12) {
                ForEach(posts) { post in
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
                    .id(post.id)
                }
            }
        }
    }
}

/// Single vertical feed (VC.M3 FR-1).
struct StreamLayoutView: View {
    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    var body: some View {
        if posts.isEmpty {
            BoardPostsEmptyPlaceholder()
        } else {
            LazyVStack(spacing: 12) {
                ForEach(posts) { post in
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
                    .id(post.id)
                }
            }
        }
    }
}

/// Uniform grid (VC.M3 FR-1).
struct GridLayoutView: View {
    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    private var columns: [GridItem] {
        [GridItem(.flexible(), spacing: 12), GridItem(.flexible(), spacing: 12)]
    }

    var body: some View {
        if posts.isEmpty {
            BoardPostsEmptyPlaceholder()
        } else {
            LazyVGrid(columns: columns, spacing: 12) {
                ForEach(posts) { post in
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
                    .id(post.id)
                    .frame(minHeight: 140, alignment: .top)
                }
            }
        }
    }
}

struct BoardPostsEmptyPlaceholder: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            VStack(spacing: 12) {
                Image(systemName: "rectangle.3.group")
                    .font(.system(size: 36))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.text("mobile.boards.postsEmptyTitle"))
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(L.text("mobile.boards.postsEmptyMessage"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 32)
        }
    }
}
