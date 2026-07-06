import SwiftUI

/// Read-only per-course grades for a linked child (M10.1).
struct ParentGradesDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let studentId: String
    let childName: String

    @State private var courses: [ParentCourseGradesRow] = []
    @State private var loading = true
    @State private var errorMessage: String?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, courses.isEmpty {
                LMSEmptyState(systemImage: "chart.bar", title: L.text("mobile.parent.section.grades"), message: errorMessage)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        Text(L.format("mobile.parent.readOnly", childName))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if courses.isEmpty {
                            Text(L.text("mobile.parent.grades.empty"))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        } else {
                            ForEach(courses) { course in
                                LMSCard {
                                    Text(course.title)
                                        .font(.headline)
                                    Text(course.courseCode)
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    if course.grades.isEmpty {
                                        Text(L.text("mobile.parent.grades.noScores"))
                                            .font(.subheadline)
                                            .padding(.top, 4)
                                    } else {
                                        ForEach(course.grades.sorted(by: { $0.key < $1.key }), id: \.key) { itemId, score in
                                            HStack {
                                                Text(itemId.prefix(8) + "…")
                                                    .font(.caption.monospaced())
                                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                                Spacer()
                                                Text(score)
                                                    .font(.subheadline.weight(.medium).monospacedDigit())
                                            }
                                            .padding(.vertical, 4)
                                        }
                                    }
                                }
                            }
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.parent.section.grades"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            courses = try await LMSAPI.fetchParentStudentGrades(studentId: studentId, accessToken: token)
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
