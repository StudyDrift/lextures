import SwiftUI

/// Course discussions section embedded in course workspace (M7.1).
struct CourseDiscussionsSection: View {
    let course: CourseSummary
    var initialThreadId: String?

    var body: some View {
        DiscussionsListView(course: course, initialThreadId: initialThreadId)
    }
}
