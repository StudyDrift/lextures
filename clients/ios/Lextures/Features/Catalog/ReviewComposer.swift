import SwiftUI

/// Star rating + review text composer for enrolled courses (M9.3).
struct ReviewComposer: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let courseTitle: String
    var initialRating: Int = 0
    var initialText: String = ""
    var hasReview: Bool = false
    var canEdit: Bool = true
    var onSubmitted: () -> Void = {}

    @State private var rating: Int
    @State private var reviewText: String
    @State private var submitting = false
    @State private var errorMessage: String?
    @State private var thanks = false

    init(
        courseCode: String,
        courseTitle: String,
        initialRating: Int = 0,
        initialText: String = "",
        hasReview: Bool = false,
        canEdit: Bool = true,
        onSubmitted: @escaping () -> Void = {}
    ) {
        self.courseCode = courseCode
        self.courseTitle = courseTitle
        self.initialRating = initialRating
        self.initialText = initialText
        self.hasReview = hasReview
        self.canEdit = canEdit
        self.onSubmitted = onSubmitted
        _rating = State(initialValue: initialRating)
        _reviewText = State(initialValue: initialText)
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if thanks {
                        LMSCard {
                            Text(L.text("mobile.reviews.thanks"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                    } else {
                        Text(courseTitle)
                            .font(.headline)
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }

                        LMSCard {
                            VStack(alignment: .leading, spacing: 12) {
                                Text(L.text("mobile.reviews.ratingLabel"))
                                    .font(.subheadline.weight(.semibold))
                                HStack(spacing: 8) {
                                    ForEach(1 ... 5, id: \.self) { star in
                                        Button {
                                            rating = star
                                        } label: {
                                            Image(systemName: star <= rating ? "star.fill" : "star")
                                                .font(.title2)
                                                .foregroundStyle(star <= rating ? .yellow : LexturesTheme.textSecondary(for: colorScheme))
                                        }
                                        .buttonStyle(.plain)
                                        .accessibilityLabel(CourseReviewLogic.starLabel(rating: star))
                                    }
                                }
                                Text(CourseReviewLogic.starLabel(rating: rating))
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                                Text(L.text("mobile.reviews.textLabel"))
                                    .font(.subheadline.weight(.semibold))
                                TextField(L.text("mobile.reviews.textPlaceholder"), text: $reviewText, axis: .vertical)
                                    .lineLimit(4 ... 8)
                                    .textFieldStyle(.roundedBorder)
                                Text(L.format(
                                    "mobile.reviews.charCount",
                                    reviewText.count,
                                    CourseReviewLogic.maxReviewTextLength
                                ))
                                .font(.caption2)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }

                        Button {
                            Task { await submit() }
                        } label: {
                            Text(submitting ? L.text("mobile.reviews.submitting") : L.text("mobile.reviews.submit"))
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(LexturesTheme.primary)
                        .disabled(submitting || session.accessToken == nil)
                    }
                }
                .padding(16)
            }
            .navigationTitle(CourseReviewLogic.composerTitle(hasReview: hasReview, canEdit: canEdit))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.discussions.cancel")) { dismiss() }
                }
            }
        }
    }

    private func submit() async {
        if let ratingError = CourseReviewLogic.validateRating(rating) {
            errorMessage = ratingError
            return
        }
        if let textError = CourseReviewLogic.validateReviewText(reviewText) {
            errorMessage = textError
            return
        }
        guard let token = session.accessToken else { return }
        submitting = true
        errorMessage = nil
        defer { submitting = false }
        do {
            _ = try await LMSAPI.submitCourseReview(
                courseCode: courseCode,
                rating: rating,
                reviewText: reviewText,
                accessToken: token
            )
            thanks = true
            onSubmitted()
        } catch let error as APIError {
            switch error {
            case .httpStatus(_, let message):
                errorMessage = message ?? L.text("mobile.reviews.submitError")
            default:
                errorMessage = L.text("mobile.reviews.submitError")
            }
        } catch {
            errorMessage = L.text("mobile.reviews.submitError")
        }
    }
}