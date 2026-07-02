import SwiftUI

struct DashboardWhatsNext: Hashable {
    var course: CourseSummary
    var primary: LearnerRecommendationItem?
    var chips: [LearnerRecommendationItem]
    var degraded: Bool
}

struct DashboardStudySection: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let studentCourses: [CourseSummary]
    let onOpenReview: () -> Void
    let onOpenRecommendation: (CourseSummary, CourseStructureItem) -> Void
    let onOpenPaths: () -> Void

    @State private var myPaths: [PathProgress] = []
    @State private var whatsNext: DashboardWhatsNext?
    @State private var loadingPaths = false
    @State private var loadingRecs = false

    var body: some View {
        Group {
            if !myPaths.isEmpty {
                pathsCard
            }
            if let whatsNext {
                recommendationsCard(whatsNext)
            }
        }
        .task(id: studentCourses.map(\.id).joined(separator: ",")) {
            await loadPaths()
            await loadRecommendations()
        }
    }

    private var pathsCard: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    LMSSectionHeader(title: L.text("mobile.paths.dashboardTitle"), systemImage: "point.topleft.down.to.point.bottomright.curvepath")
                    Spacer(minLength: 0)
                    Button(L.text("mobile.paths.viewAll"), action: onOpenPaths)
                        .font(.caption.weight(.semibold))
                }
                ForEach(myPaths.prefix(3)) { path in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(path.pathTitle)
                                .font(.caption.weight(.semibold))
                            Spacer(minLength: 0)
                            Text(path.progressLabel)
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        ProgressView(value: Double(path.percent), total: 100)
                            .tint(LexturesTheme.primary)
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func recommendationsCard(_ bundle: DashboardWhatsNext) -> some View {
        LMSCard(accent: LexturesTheme.primary) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "sparkles")
                        .foregroundStyle(LexturesTheme.primary)
                    Text(bundle.course.displayTitle)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                if let primary = bundle.primary {
                    Text(primary.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(primary.reason)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    if bundle.degraded {
                        Text(L.text("mobile.paths.recommendationsDegraded"))
                            .font(.caption2)
                            .foregroundStyle(LexturesTheme.amber)
                    }
                    Button(L.text("mobile.paths.go")) {
                        if primary.itemType == "review_card" {
                            onOpenReview()
                        } else {
                            Task { await openRecommendation(course: bundle.course, item: primary) }
                        }
                        Task { await recordClick(course: bundle.course, item: primary, rank: 0) }
                    }
                    .font(.caption.weight(.semibold))
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.primary)
                    if !bundle.chips.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(Array(bundle.chips.enumerated()), id: \.element.id) { index, chip in
                                    Button(chip.title) {
                                        if chip.itemType == "review_card" {
                                            onOpenReview()
                                        } else {
                                            Task { await openRecommendation(course: bundle.course, item: chip) }
                                        }
                                        Task { await recordClick(course: bundle.course, item: chip, rank: index + 1) }
                                    }
                                    .font(.caption2.weight(.semibold))
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 6)
                                    .background(LexturesTheme.primary.opacity(0.12))
                                    .clipShape(Capsule())
                                }
                            }
                        }
                    }
                } else {
                    Text(L.format("mobile.paths.caughtUpInCourse", bundle.course.displayTitle))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadPaths() async {
        guard let token = session.accessToken else { return }
        loadingPaths = true
        defer { loadingPaths = false }
        myPaths = (try? await LMSAPI.fetchMyPaths(accessToken: token)) ?? []
    }

    private func loadRecommendations() async {
        guard let token = session.accessToken,
              let userId = NotebookStore.jwtSubject(from: token),
              let course = studentCourses.first(where: \.viewerIsStudent) ?? studentCourses.first
        else {
            whatsNext = nil
            return
        }
        loadingRecs = true
        defer { loadingRecs = false }
        do {
            let responses = try await withThrowingTaskGroup(of: LearnerRecommendationsResponse.self) { group in
                for surface in PathsLogic.recommendationSurfaces {
                    group.addTask {
                        try await LMSAPI.fetchLearnerRecommendations(
                            userId: userId,
                            courseId: course.id,
                            surface: surface,
                            accessToken: token,
                            limit: 4
                        )
                    }
                }
                var out: [LearnerRecommendationsResponse] = []
                for try await response in group { out.append(response) }
                return out
            }
            let merged = PathsLogic.mergeRecommendations(responses)
            whatsNext = DashboardWhatsNext(
                course: course,
                primary: merged.primary,
                chips: merged.chips,
                degraded: merged.degraded
            )
            if let primary = merged.primary {
                try? await LMSAPI.postRecommendationEvent(
                    body: RecommendationEventBody(
                        courseId: course.id,
                        itemId: primary.itemId,
                        surface: primary.surface,
                        eventType: "impression",
                        rank: 0
                    ),
                    accessToken: token
                )
            }
        } catch {
            whatsNext = nil
        }
    }

    private func openRecommendation(course: CourseSummary, item: LearnerRecommendationItem) async {
        guard let token = session.accessToken else { return }
        let structure = (try? await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)) ?? []
        guard let target = PathsLogic.structureItem(for: item, in: structure) else { return }
        onOpenRecommendation(course, target)
    }

    private func recordClick(course: CourseSummary, item: LearnerRecommendationItem, rank: Int) async {
        guard let token = session.accessToken else { return }
        try? await LMSAPI.postRecommendationEvent(
            body: RecommendationEventBody(
                courseId: course.id,
                itemId: item.itemId,
                surface: item.surface,
                eventType: "click",
                rank: rank
            ),
            accessToken: token
        )
    }
}