import SwiftUI

/// Public course landing page with reviews and self-enroll (M9.1).
struct CourseLandingView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let slug: String

    @State private var course: PublicCatalogCourse?
    @State private var reviews: CourseReviewsListResponse?
    @State private var enrolledCourses: [CourseSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var enrolling = false
    @State private var enrollError: String?
    @State private var showPurchase = false
    @State private var showReviewComposer = false
    @State private var reviewEligibility: ReviewEligibility?

    private var isEnrolled: Bool {
        guard let course else { return false }
        return CatalogLogic.isEnrolled(courseCode: course.courseCode, in: enrolledCourses)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, course == nil {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: L.text("mobile.catalog.landingErrorTitle"),
                    message: errorMessage
                )
            } else if let course {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        header(course)
                        aboutSection(course)
                        if let reviews, shell.platformFeatures.ffCourseReviews {
                            reviewsSection(reviews)
                        }
                        enrollSection(course)
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(course?.title ?? L.text("mobile.catalog.landingTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .sheet(isPresented: $showPurchase) {
            if let course {
                PurchaseFlowSheet(
                    courseId: course.id,
                    courseCode: course.courseCode,
                    title: course.title,
                    priceCents: course.priceCents,
                    currency: "USD"
                )
            }
        }
        .sheet(isPresented: $showReviewComposer) {
            if let course {
                ReviewComposer(
                    courseCode: course.courseCode,
                    courseTitle: course.title,
                    hasReview: reviewEligibility?.hasReview == true,
                    canEdit: reviewEligibility?.canEdit == true,
                    onSubmitted: { Task { await load() } }
                )
            }
        }
    }

    @ViewBuilder
    private func header(_ course: PublicCatalogCourse) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                CourseHeroImage(urlString: course.heroImageUrl, fallbackKey: course.courseCode, height: 160)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

                HStack(spacing: 8) {
                    if let category = course.category, !category.isEmpty {
                        Text(category)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if let level = course.difficultyLevel, !level.isEmpty {
                        Text(level.capitalized)
                            .font(.caption.weight(.semibold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(Capsule().fill(LexturesTheme.cardBackground(for: colorScheme)))
                    }
                }

                if let instructor = course.instructorName, !instructor.isEmpty {
                    Text(L.format("mobile.catalog.instructor", instructor))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                HStack(spacing: 12) {
                    Text(CatalogLogic.ratingLabel(average: course.averageRating, count: course.ratingCount))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    Text(L.format("mobile.catalog.enrolledCount", course.enrollmentCount))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func aboutSection(_ course: PublicCatalogCourse) -> some View {
        LMSSectionHeader(title: L.text("mobile.catalog.aboutTitle"), systemImage: "text.alignleft")
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                let paragraphs = CatalogLogic.previewParagraphs(from: course.description)
                if paragraphs.isEmpty {
                    Text(L.text("mobile.catalog.noDescription"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    ForEach(Array(paragraphs.enumerated()), id: \.offset) { _, paragraph in
                        Text(paragraph)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func reviewsSection(_ reviews: CourseReviewsListResponse) -> some View {
        LMSSectionHeader(title: L.text("mobile.catalog.reviewsTitle"), systemImage: "star.fill")
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(CatalogLogic.ratingLabel(
                    average: reviews.summary.averageRating,
                    count: reviews.summary.ratingCount
                ))
                .font(.subheadline.weight(.semibold))

                if let reviewEligibility,
                   CourseReviewLogic.shouldShowComposer(reviewEligibility) {
                    Button(L.text("mobile.reviews.writeCta")) {
                        showReviewComposer = true
                    }
                    .font(.caption.weight(.semibold))
                } else if let reviewEligibility, !reviewEligibility.eligible {
                    Text(CourseReviewLogic.progressHint(reviewEligibility.progressPercent))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                ForEach(reviews.reviews ?? []) { review in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(review.reviewerDisplayName)
                                .font(.caption.weight(.semibold))
                            Spacer()
                            Text(L.format("mobile.catalog.reviewStars", review.rating))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if let text = review.reviewText, !text.isEmpty {
                            Text(text)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private func enrollSection(_ course: PublicCatalogCourse) -> some View {
        if let enrollError {
            LMSErrorBanner(message: enrollError)
        }

        LMSCard {
            HStack {
                Text(CatalogLogic.formatPrice(cents: course.priceCents))
                    .font(.title3.weight(.bold))
                Spacer()
                actionButton(course)
            }
        }
    }

    @ViewBuilder
    private func actionButton(_ course: PublicCatalogCourse) -> some View {
        if isEnrolled {
            Button(L.text("mobile.catalog.continue")) {
                continueCourse(course)
            }
            .buttonStyle(.borderedProminent)
            .tint(LexturesTheme.primary)
        } else if CatalogLogic.isPaid(priceCents: course.priceCents) {
            if BillingLogic.billingEnabled(shell.platformFeatures) {
                Button(L.text("mobile.billing.purchase")) {
                    showPurchase = true
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
            } else {
                Button(L.text("mobile.catalog.openOnWeb")) {
                    openURL(AppConfiguration.webURL(path: CatalogLogic.catalogWebPath(slug: slug)))
                }
                .buttonStyle(.borderedProminent)
                .tint(LexturesTheme.primary)
            }
        } else {
            Button {
                Task { await enroll(course) }
            } label: {
                Text(enrolling ? L.text("mobile.catalog.enrolling") : L.text("mobile.catalog.enrollFree"))
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
            async let detailTask = LMSAPI.fetchPublicCatalogCourseDetail(
                slug: slug,
                accessToken: session.accessToken
            )
            async let reviewsTask = LMSAPI.fetchPublicCatalogCourseReviews(
                slug: slug,
                accessToken: session.accessToken
            )
            if let token = session.accessToken {
                enrolledCourses = try await LMSAPI.fetchCourses(accessToken: token)
            }
            guard let loaded = try await detailTask else {
                errorMessage = L.text("mobile.catalog.landingNotFound")
                return
            }
            course = loaded
            reviews = try await reviewsTask
            if let token = session.accessToken,
               shell.platformFeatures.ffCourseReviews,
               CatalogLogic.isEnrolled(courseCode: loaded.courseCode, in: enrolledCourses) {
                reviewEligibility = try? await LMSAPI.fetchReviewEligibility(
                    courseCode: loaded.courseCode,
                    accessToken: token
                )
            } else {
                reviewEligibility = nil
            }
        } catch {
            errorMessage = L.text("mobile.catalog.landingError")
        }
    }

    private func enroll(_ course: PublicCatalogCourse) async {
        guard let token = session.accessToken else { return }
        enrolling = true
        enrollError = nil
        defer { enrolling = false }
        do {
            _ = try await LMSAPI.selfEnrollInCourse(courseCode: course.courseCode, accessToken: token)
            let refreshed = try await LMSAPI.fetchCourse(courseCode: course.courseCode, accessToken: token)
            enrolledCourses.append(refreshed)
            continueCourse(course, summary: refreshed)
        } catch let error as APIError {
            switch error {
            case .httpStatus(402, _):
                enrollError = L.text("mobile.catalog.paidRequired")
            case .httpStatus(403, _):
                enrollError = L.text("mobile.catalog.enrollForbidden")
            default:
                enrollError = L.text("mobile.catalog.enrollError")
            }
        } catch {
            enrollError = L.text("mobile.catalog.enrollError")
        }
    }

    private func continueCourse(_ course: PublicCatalogCourse, summary: CourseSummary? = nil) {
        let target = summary ?? CatalogLogic.enrolledCourse(courseCode: course.courseCode, in: enrolledCourses)
        guard let target else { return }
        shell.activeCourse = target
        shell.activeCourseRoot = .profile
        shell.activeCourseSection = .modules
        shell.select(.courses)
    }
}