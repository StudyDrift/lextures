import SwiftUI

/// Course workspace Library section listing e-reserve items (M3.6).
struct CourseLibraryView: View {
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let items: [CourseStructureItem]
    var onSelectItem: (CourseStructureItem) -> Void

    private var libraryItems: [CourseStructureItem] {
        LibraryResourceLogic.libraryItems(from: items)
    }

    var body: some View {
        if libraryItems.isEmpty {
            LMSEmptyState(
                systemImage: "books.vertical.fill",
                title: L.text("mobile.library.courseEmptyTitle"),
                message: L.text("mobile.library.courseEmptyMessage")
            )
        } else {
            VStack(spacing: 10) {
                ForEach(libraryItems, id: \.id) { item in
                    Button {
                        onSelectItem(item)
                    } label: {
                        LMSCard {
                            HStack(spacing: 12) {
                                Image(systemName: "book.closed.fill")
                                    .font(.title3)
                                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(item.title)
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                        .multilineTextAlignment(.leading)
                                    Text(L.text("mobile.library.type.generic"))
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                                Spacer(minLength: 0)
                                Image(systemName: "chevron.right")
                                    .font(.caption.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }
}