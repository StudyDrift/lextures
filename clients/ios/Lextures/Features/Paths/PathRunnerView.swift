import SwiftUI

struct PathRunnerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let path: PathProgress

    @State private var progress: PathProgress
    @State private var coursesByCode: [String: CourseSummary] = [:]
    @State private var openCourse: CourseSummary?
    @State private var errorMessage: String?
    @State private var loading = false

    init(path: PathProgress) {
        self.path = path
        _progress = State(initialValue: path)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    summaryCard
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    stepsSection
                }
                .padding(16)
            }
        }
        .navigationTitle(progress.pathTitle)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $openCourse) { course in
            CourseDetailView(course: course)
        }
        .refreshable { await reload() }
        .task { await reload() }
    }

    private var summaryCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                ProgressView(value: Double(progress.percent), total: 100)
                    .tint(LexturesTheme.primary)
                Text(progress.progressLabel)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if progress.justCompleted {
                    Text(L.text("mobile.paths.completedBanner"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.amber)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var stepsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            LMSSectionHeader(title: L.text("mobile.paths.steps"), systemImage: "list.number")
            ForEach(PathsLogic.sortedCourses(progress.courses)) { course in
                stepRow(course)
            }
        }
    }

    @ViewBuilder
    private func stepRow(_ course: PathCourseProgress) -> some View {
        let locked = PathsLogic.isLocked(course)
        let isNext = PathsLogic.nextCourse(in: progress)?.courseId == course.courseId

        LMSCard(accent: isNext ? LexturesTheme.primary : nil) {
            HStack(spacing: 12) {
                Image(systemName: course.isCompleted ? "checkmark.circle.fill" : (locked ? "lock.fill" : "circle"))
                    .foregroundStyle(
                        course.isCompleted ? LexturesTheme.primary :
                            locked ? LexturesTheme.textSecondary(for: colorScheme) :
                            LexturesTheme.accent(for: colorScheme)
                    )
                VStack(alignment: .leading, spacing: 3) {
                    Text(course.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(course.courseCode.uppercased())
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if isNext {
                        Text(L.text("mobile.paths.nextStep"))
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(LexturesTheme.primary)
                    } else if locked {
                        Text(L.text("mobile.paths.locked"))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer(minLength: 0)
                if !locked {
                    Button(L.text(isNext ? "mobile.paths.continue" : "mobile.paths.openCourse")) {
                        Task { await open(course) }
                    }
                    .font(.caption.weight(.semibold))
                    .buttonStyle(.borderedProminent)
                    .tint(isNext ? LexturesTheme.primary : LexturesTheme.accent(for: colorScheme))
                }
            }
        }
    }

    private func open(_ course: PathCourseProgress) async {
        if let cached = coursesByCode[course.courseCode] {
            openCourse = cached
            return
        }
        guard let token = session.accessToken else { return }
        loading = true
        defer { loading = false }
        do {
            let summary = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            coursesByCode[course.courseCode] = summary
            openCourse = summary
        } catch {
            errorMessage = L.text("mobile.paths.error.openCourse")
        }
    }

    private func reload() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            progress = try await OfflineService.shared.cachedFetch(
                key: OfflineCacheKey.pathProgress(progress.pathId),
                accessToken: token
            ) {
                try await LMSAPI.fetchPathProgress(pathId: progress.pathId, accessToken: token)
            }.value
        } catch {
            errorMessage = L.text("mobile.paths.error.load")
        }
    }
}