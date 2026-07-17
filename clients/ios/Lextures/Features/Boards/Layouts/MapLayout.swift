import MapKit
import SwiftUI

/// Native MapKit pins with clustering + list fallback (VC.M3 FR-7).
struct MapLayoutView: View {
    @Environment(\.colorScheme) private var colorScheme

    let posts: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    @State private var selectedId: String?
    @State private var position: MapCameraPosition = .automatic
    @State private var showList = false

    private var pinned: [BoardPost] { BoardsLogic.pinnedPosts(posts) }
    private var unpinned: [BoardPost] { BoardsLogic.unpinnedPosts(posts) }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                Text(L.text("mobile.boards.layout.map"))
                    .font(.subheadline.weight(.semibold))
                Spacer()
                Button(showList
                       ? L.text("mobile.boards.layout.mapShowMap")
                       : L.text("mobile.boards.layout.mapListFallback")) {
                    showList.toggle()
                }
                .font(.caption.weight(.semibold))
            }

            if showList {
                pinList
            } else {
                mapSurface
                    .frame(height: 280)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
                    .accessibilityLabel(L.text("mobile.boards.layout.map"))
            }

            if let selected = selectedId.flatMap({ id in posts.first(where: { $0.id == id }) }) {
                BoardPostCardSlot(
                    post: selected,
                    siblings: posts,
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    showMap: true,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            }

            if !unpinned.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.boards.layout.unpinnedTray"))
                        .font(.subheadline.weight(.semibold))
                    ForEach(unpinned) { post in
                        BoardPostCardSlot(
                            post: post,
                            siblings: unpinned,
                            sections: sections,
                            board: board,
                            canManage: canManage,
                            currentUserId: currentUserId,
                            showMap: true,
                            onEdit: onEdit,
                            onDelete: onDelete,
                            onArrange: onArrange
                        )
                    }
                }
                .padding(12)
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .strokeBorder(style: StrokeStyle(lineWidth: 1, dash: [6]))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.4))
                )
            }

            if posts.isEmpty {
                BoardPostsEmptyPlaceholder()
            }
        }
    }

    @ViewBuilder
    private var mapSurface: some View {
        if pinned.isEmpty {
            ZStack {
                RoundedRectangle(cornerRadius: 12)
                    .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.08))
                Text(L.text("mobile.boards.layout.mapEmpty"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        } else {
            Map(position: $position, selection: $selectedId) {
                ForEach(pinned) { post in
                    if let lat = post.lat, let lng = post.lng {
                        Marker(
                            post.title.isEmpty ? L.text("mobile.boards.layout.mapPin") : post.title,
                            coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lng)
                        )
                        .tag(post.id)
                    }
                }
            }
            .mapStyle(.standard(elevation: .realistic))
            .mapControls {
                MapCompass()
                MapPitchToggle()
            }
        }
    }

    private var pinList: some View {
        VStack(alignment: .leading, spacing: 8) {
            ForEach(pinned) { post in
                Button {
                    selectedId = post.id
                } label: {
                    HStack {
                        Image(systemName: "mappin.circle.fill")
                        VStack(alignment: .leading, spacing: 2) {
                            Text(post.title.isEmpty ? L.text("mobile.boards.layout.mapPin") : post.title)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            if let lat = post.lat, let lng = post.lng {
                                Text(String(format: "%.4f, %.4f", lat, lng))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        Spacer()
                    }
                    .padding(10)
                    .background(
                        RoundedRectangle(cornerRadius: 10)
                            .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.08))
                    )
                }
                .buttonStyle(.plain)
                .accessibilityLabel(post.title.isEmpty ? L.text("mobile.boards.layout.mapPin") : post.title)
            }
            if pinned.isEmpty {
                Text(L.text("mobile.boards.layout.mapEmpty"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }
}
