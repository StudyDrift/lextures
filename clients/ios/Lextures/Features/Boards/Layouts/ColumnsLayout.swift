import SwiftUI

/// Horizontally swipeable section lanes (VC.M3 FR-1 / FR-3 / FR-4).
struct ColumnsLayoutView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.layoutDirection) private var layoutDirection

    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void
    var onCreateSection: ((String) -> Void)?
    var onDeleteSection: ((String) -> Void)?

    @State private var newSectionTitle = ""
    @State private var showAddSection = false

    private var orderedSections: [BoardSection] {
        BoardsLogic.sortedSections(sections)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if canManage {
                HStack {
                    Spacer()
                    Button {
                        newSectionTitle = ""
                        showAddSection = true
                    } label: {
                        Label(L.text("mobile.boards.section.add"), systemImage: "plus")
                            .font(.subheadline.weight(.semibold))
                    }
                }
            }

            if orderedSections.isEmpty && posts.isEmpty {
                BoardPostsEmptyPlaceholder()
            } else if orderedSections.isEmpty {
                unsortedLane
            } else {
                TabView {
                    ForEach(orderedSections) { section in
                        sectionLane(section)
                            .padding(.horizontal, 4)
                    }
                    unsortedLane
                        .padding(.horizontal, 4)
                }
                .tabViewStyle(.page(indexDisplayMode: .automatic))
                .frame(minHeight: 420)
                .environment(\.layoutDirection, layoutDirection)
            }
        }
        .alert(L.text("mobile.boards.section.add"), isPresented: $showAddSection) {
            TextField(L.text("mobile.boards.section.titlePlaceholder"), text: $newSectionTitle)
            Button(L.text("mobile.common.save")) {
                let title = newSectionTitle.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !title.isEmpty else { return }
                onCreateSection?(title)
            }
            .disabled(newSectionTitle.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            Button(L.text("mobile.common.cancel"), role: .cancel) {}
        }
    }

    private var unsortedLane: some View {
        lane(
            title: L.text("mobile.boards.section.unsorted"),
            sectionId: nil,
            section: nil,
            lanePosts: BoardsLogic.postsInSection(posts, sectionId: nil)
        )
    }

    private func sectionLane(_ section: BoardSection) -> some View {
        lane(
            title: section.title,
            sectionId: section.id,
            section: section,
            lanePosts: BoardsLogic.postsInSection(posts, sectionId: section.id)
        )
    }

    private func lane(
        title: String,
        sectionId: String?,
        section: BoardSection?,
        lanePosts: [BoardPost]
    ) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(title)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer()
                if canManage, let section {
                    Button(role: .destructive) {
                        onDeleteSection?(section.id)
                    } label: {
                        Image(systemName: "trash")
                    }
                    .accessibilityLabel(L.text("mobile.boards.section.delete"))
                }
            }
            if lanePosts.isEmpty {
                Text(L.text("mobile.boards.section.dropHere"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .frame(maxWidth: .infinity, minHeight: 120)
                    .overlay(
                        RoundedRectangle(cornerRadius: 12)
                            .strokeBorder(style: StrokeStyle(lineWidth: 1, dash: [6]))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.4))
                    )
            } else {
                ScrollView {
                    LazyVStack(spacing: 10) {
                        ForEach(lanePosts) { post in
                            BoardPostCardSlot(
                                post: post,
                                siblings: lanePosts,
                                sections: orderedSections,
                                board: board,
                                canManage: canManage,
                                currentUserId: currentUserId,
                                onEdit: onEdit,
                                onDelete: onDelete,
                                onArrange: { p, input in
                                    var next = input
                                    if next.sectionId == nil, sectionId != nil {
                                        // Keep section when only reordering inside the lane.
                                    }
                                    onArrange(p, next)
                                }
                            )
                            .id(post.id)
                        }
                    }
                }
            }
        }
        .padding(12)
        .background(
            RoundedRectangle(cornerRadius: 14)
                .fill(LexturesTheme.cardBackground(for: colorScheme))
        )
    }
}
