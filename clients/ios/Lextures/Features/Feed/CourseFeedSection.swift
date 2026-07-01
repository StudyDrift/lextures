import SwiftUI

/// Course feed section embedded in the course workspace (M7.6).
struct CourseFeedSection: View {
    let course: CourseSummary

    var body: some View {
        FeedChannelsView(course: course)
    }
}
