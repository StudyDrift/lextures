import SwiftUI

struct PathLandingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    @Environment(\.dismiss) private var dismiss

    let slug: String

    @State private var detail: LearningPathDetail?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var enrolling = false
    @State private var enrollError: String?
    @State private var enrolled = false

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, detail == nil {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: L.text("mobile.paths.landingErrorTitle"),
                    message: errorMessage
                )
            } else if let detail {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        header(detail)
                        coursesSection(detail)
                        enrollSection(detail)
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(detail?.path.title ?? L.text("mobile.paths.landingTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(isPresented: $enrolled) {
            MyPathsView()
        }
        .task { await load() }
    }

    @ViewBuilder
    private func header(_ detail: LearningPathDetail) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(L.text("mobile.paths.landingBadge"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                if !detail.path.description.isEmpty {
                    Text(detail.path.description)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Text(
                    L.format(
                        "mobile.paths.landingMeta",
                        detail.courses.count,
                        PathsLogic.formatDuration(minutes: detail.totalDurationMinutes)
                    )
                )
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func coursesSection(_ detail: LearningPathDetail) -> some View {
        LMSSectionHeader(title: L.text("mobile.paths.coursesInPath"), systemImage: "books.vertical")
        ForEach(PathsLogic.sortedCourses(detail.courses)) { course in
            LMSCard {
                VStack(alignment: .leading, spacing: 4) {
                    Text(course.title)
                        .font(.subheadline.weight(.semibold))
                    Text(course.courseCode.uppercased())
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }

    @ViewBuilder
    private func enrollSection(_ detail: LearningPathDetail) -> some View {
        if let enrollError {
            LMSErrorBanner(message: enrollError)
        }

        if PathsLogic.isPaid(bundlePriceCents: detail.path.bundlePriceCents) {
            LMSCard {
                VStack(alignment: .leading, spacing: 8) {
                    Text(L.text("mobile.paths.paidTitle"))
                        .font(.subheadline.weight(.semibold))
                    Text(L.text("mobile.paths.paidHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Button(L.text("mobile.paths.openOnWeb")) {
                        openURL(AppConfiguration.webURL(path: PathsLogic.catalogWebPath(slug: slug)))
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(LexturesTheme.primary)
                }
            }
        } else {
            Button {
                Task { await enroll(pathId: detail.path.id) }
            } label: {
                Text(enrolling ? L.text("mobile.paths.enrolling") : L.text("mobile.paths.startFree"))
                    .font(.headline)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
            .disabled(enrolling || session.accessToken == nil)
        }
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            guard let loaded = try await LMSAPI.fetchCatalogPathDetail(
                slug: slug,
                accessToken: session.accessToken
            ) else {
                errorMessage = L.text("mobile.paths.landingNotFound")
                return
            }
            detail = loaded
        } catch {
            errorMessage = L.text("mobile.paths.error.landing")
        }
    }

    private func enroll(pathId: String) async {
        guard let token = session.accessToken else { return }
        enrolling = true
        enrollError = nil
        defer { enrolling = false }
        do {
            _ = try await LMSAPI.enrollInPath(pathId: pathId, accessToken: token)
            enrolled = true
        } catch let error as APIError {
            switch error {
            case .httpStatus(402, _):
                enrollError = L.text("mobile.paths.paidRequired")
            default:
                enrollError = L.text("mobile.paths.error.enroll")
            }
        } catch {
            enrollError = L.text("mobile.paths.error.enroll")
        }
    }
}