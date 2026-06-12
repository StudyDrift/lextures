import SwiftUI

/// "Overview" section of course detail: syllabus sections rendered as markdown,
/// falling back to the course description when no syllabus exists.
struct CourseSyllabusSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    @State private var syllabus: SyllabusPayload?
    @State private var errorMessage: String?
    @State private var loading = true

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if loading {
                LMSSkeletonList(count: 2)
            } else if let syllabus, syllabus.hasContent {
                if syllabus.syllabusAcceptancePending == true {
                    acceptancePendingBanner
                }
                ForEach(syllabus.sections) { section in
                    sectionCard(section)
                }
                if let updated = LMSDates.parse(syllabus.updatedAt) {
                    Text("Updated \(updated.formatted(date: .abbreviated, time: .omitted))")
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .frame(maxWidth: .infinity, alignment: .trailing)
                }
            } else {
                descriptionFallback
            }
        }
        .task { await load() }
    }

    private var acceptancePendingBanner: some View {
        Label(
            "This course asks you to review and accept the syllabus. You can accept it from the web app.",
            systemImage: "checkmark.seal"
        )
        .font(.caption)
        .foregroundStyle(LexturesTheme.amber)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(LexturesTheme.amber.opacity(0.11))
        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private func sectionCard(_ section: SyllabusSection) -> some View {
        LMSCard {
            if !section.heading.isEmpty {
                Text(section.heading)
                    .font(LexturesTheme.displayFont(18))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            MarkdownTextView(markdown: section.markdown)
        }
    }

    @ViewBuilder
    private var descriptionFallback: some View {
        if course.description.isEmpty {
            LMSEmptyState(
                systemImage: "doc.text",
                title: "No syllabus yet",
                message: "The course overview will appear here once the instructor adds it."
            )
        } else {
            LMSCard {
                Text("About this course")
                    .font(LexturesTheme.displayFont(18))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Text(course.description)
                    .font(.subheadline)
                    .lineSpacing(3)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            syllabus = try await LMSAPI.fetchSyllabus(courseCode: course.courseCode, accessToken: token)
        } catch {
            // A missing syllabus is expected for many courses — fall back quietly.
            syllabus = nil
        }
    }
}
