import SwiftUI

/// Graceful empty state for course workspace slots awaiting destination stories.
struct CourseDestinationPlaceholder: View {
    @Environment(\.colorScheme) private var colorScheme
    let section: CourseWorkspaceSection

    var body: some View {
        LMSEmptyState(
            systemImage: icon,
            title: section.label,
            message: L.text("mobile.ia.placeholder.message")
        )
        .frame(maxWidth: .infinity)
        .padding(.vertical, 24)
    }

    private var icon: String {
        switch section {
        case .discussions: return "bubble.left.and.bubble.right"
        case .feed: return "text.bubble"
        case .live: return "video"
        case .people: return "person.3"
        case .evaluations: return "star"
        case .library: return "books.vertical"
        case .groups: return "person.3"
        case .collabDocs: return "doc.text"
        case .boards: return "rectangle.3.group"
        case .liveQuizzes: return "gamecontroller"
        default: return "sparkles"
        }
    }
}