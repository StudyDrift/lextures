import SwiftUI

struct CoursesListView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @State private var courses: [CourseSummary] = []
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var loadedOnce = false
    @State private var searchText = ""

    private var filteredCourses: [CourseSummary] {
        let q = searchText.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return courses }
        return courses.filter {
            $0.displayTitle.lowercased().contains(q)
                || $0.courseCode.lowercased().contains(q)
                || $0.description.lowercased().contains(q)
        }
    }

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        if loading && courses.isEmpty {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 40)
                        } else if filteredCourses.isEmpty {
                            LMSEmptyState(
                                systemImage: "book",
                                title: searchText.isEmpty ? "No courses yet" : "No matching courses",
                                message: searchText.isEmpty
                                    ? "Courses you enroll in will show up here."
                                    : "Try different keywords, or clear search."
                            )
                        } else {
                            ForEach(filteredCourses) { course in
                                NavigationLink(value: course) {
                                    CourseRowCard(course: course)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(16)
                }
                .refreshable { await load(force: true) }
            }
            .navigationTitle("Courses")
            .navigationBarTitleDisplayMode(.inline)
            .searchable(text: $searchText, prompt: "Search courses")
            .navigationDestination(for: CourseSummary.self) { course in
                CourseDetailView(course: course)
            }
            .task { await load() }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if loadedOnce && !force { return }
        loading = true
        errorMessage = nil
        defer {
            loading = false
            loadedOnce = true
        }
        do {
            courses = try await LMSAPI.fetchCourses(accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load courses."
        }
    }
}

struct CourseRowCard: View {
    @Environment(\.colorScheme) private var colorScheme
    let course: CourseSummary

    var body: some View {
        LMSCard {
            HStack(alignment: .top, spacing: 12) {
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(LexturesTheme.primary.opacity(0.12))
                    .frame(width: 44, height: 44)
                    .overlay(
                        Image(systemName: "book")
                            .foregroundStyle(LexturesTheme.primary)
                    )

                VStack(alignment: .leading, spacing: 4) {
                    Text(course.displayTitle)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(course.courseCode)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if !course.description.isEmpty {
                        Text(course.description)
                            .font(.caption)
                            .lineLimit(2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                Spacer(minLength: 0)

                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .padding(.top, 14)
            }
        }
    }
}
