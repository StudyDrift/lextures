import SwiftUI

struct DashboardCoursesCarousel: View {
    let courses: [CourseSummary]
    let loading: Bool
    let courseItemCounts: [String: (modules: Int, items: Int)]
    let colorScheme: ColorScheme

    var body: some View {
        LMSSectionHeader(title: L.text("mobile.dashboard.section.yourCourses"), systemImage: "book.fill")
        if courses.isEmpty && !loading {
            LMSEmptyState(
                systemImage: "book",
                title: L.text("mobile.dashboard.empty.courses.title"),
                message: L.text("mobile.dashboard.empty.courses.message")
            )
        } else {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: 12) {
                    ForEach(courses) { course in
                        NavigationLink(value: course) {
                            dashboardCourseCarouselCard(
                                course,
                                itemCounts: courseItemCounts[course.courseCode],
                                colorScheme: colorScheme
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.vertical, 2)
                .padding(.horizontal, 2)
            }
            .scrollClipDisabled()
            .padding(.bottom, 4)
        }
    }
}

private func dashboardCourseCarouselCard(
    _ course: CourseSummary,
    itemCounts: (modules: Int, items: Int)?,
    colorScheme: ColorScheme
) -> some View {
    VStack(alignment: .leading, spacing: 0) {
        ZStack(alignment: .topTrailing) {
            CourseHeroImage(
                urlString: course.heroImageUrl,
                fallbackKey: course.courseCode,
                height: 84
            )
            Image(systemName: "book.fill")
                .font(.title3)
                .foregroundStyle(.white.opacity(0.5))
                .padding(12)
        }
        VStack(alignment: .leading, spacing: 4) {
            Text(course.displayTitle)
                .font(LexturesTheme.displayFont(15))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .lineLimit(2, reservesSpace: true)
                .multilineTextAlignment(.leading)
            Text(dashboardCourseSubtitle(course, itemCounts: itemCounts))
                .font(.caption2.weight(.medium))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(12)
    }
    .frame(width: 190, alignment: .leading)
    .background(LexturesTheme.cardBackground(for: colorScheme))
    .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
    .overlay(
        RoundedRectangle(cornerRadius: 18, style: .continuous)
            .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45), lineWidth: 1)
    )
    .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 12, y: 5)
}

private func dashboardCourseSubtitle(_ course: CourseSummary, itemCounts: (modules: Int, items: Int)?) -> String {
    if let counts = itemCounts, counts.items > 0 {
        return "\(counts.modules) module\(counts.modules == 1 ? "" : "s") · \(counts.items) item\(counts.items == 1 ? "" : "s")"
    }
    return course.courseCode.uppercased()
}
