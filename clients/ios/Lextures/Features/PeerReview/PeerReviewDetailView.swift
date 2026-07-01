import SwiftUI

/// Read a peer submission, score the rubric, comment, and submit (M5.2).
struct PeerReviewDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let allocationId: String
    var onSubmitted: () -> Void = {}

    @State private var detail: PeerReviewAllocationDetail?
    @State private var rubricScores: [String: Double] = [:]
    @State private var scoreText = ""
    @State private var comments = ""
    @State private var loading = true
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var successMessage: String?
    @State private var previewTarget: FilePreviewTarget?

    private var rubric: RubricDefinition? { detail?.rubric }
    private var hasRubric: Bool { rubric.map { !$0.criteria.isEmpty } ?? false }
    private var isSubmitted: Bool { detail.map { PeerReviewLogic.isComplete($0.allocation) } ?? false }

    private var scoreValue: Double? {
        Double(scoreText.replacingOccurrences(of: ",", with: "."))
    }

    private var canSubmit: Bool {
        guard detail != nil, !saving else { return false }
        if hasRubric, let rubric {
            return PeerReviewLogic.rubricScoresComplete(rubric, scores: rubricScores)
        }
        if let scoreValue { return scoreValue >= 0 }
        return !comments.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading && detail == nil {
                ProgressView()
            } else if let detail {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        if let errorMessage {
                            LMSErrorBanner(message: errorMessage)
                        }
                        if let successMessage {
                            LMSCard(accent: LexturesTheme.brandTeal) {
                                Label(successMessage, systemImage: "checkmark.circle.fill")
                                    .font(.subheadline.weight(.semibold))
                            }
                        }

                        headerCard(detail)
                        submissionCard(detail.submission)

                        if hasRubric, let rubric {
                            RubricScorerView(rubric: rubric, scores: $rubricScores, disabled: isSubmitted && saving)
                        } else {
                            scoreCard
                        }

                        commentsCard
                        submitBar
                    }
                    .padding(16)
                }
            } else {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: L.text("mobile.peerReview.detailErrorTitle"),
                    message: errorMessage ?? L.text("mobile.peerReview.loadError")
                )
                .padding(24)
            }
        }
        .navigationTitle(detail?.assignmentTitle ?? L.text("mobile.peerReview.detailTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
        .task(id: allocationId) { await load() }
    }

    private func headerCard(_ detail: PeerReviewAllocationDetail) -> some View {
        LMSCard {
            Text(PeerReviewLogic.targetLabel(detail.allocation))
                .font(LexturesTheme.displayFont(20))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(detail.allocation.courseCode)
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.dynamicText(PeerReviewLogic.statusLabelKey(detail.allocation.status)))
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
        }
    }

    private func submissionCard(_ submission: AssignmentSubmission) -> some View {
        LMSCard {
            Text(L.text("mobile.peerReview.theirWork"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if let body = submission.bodyText?.trimmingCharacters(in: .whitespacesAndNewlines), !body.isEmpty {
                Text(body)
                    .font(.body)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .textSelection(.enabled)
            }

            if AssignmentLogic.hasAttachment(submission) {
                Button {
                    if let path = submission.attachmentContentPath,
                       let name = submission.attachmentFilename,
                       let courseCode = detail?.allocation.courseCode {
                        previewTarget = FilePreviewTarget.submissionContentPath(
                            courseCode: courseCode,
                            contentPath: path,
                            fileName: name,
                            mimeType: submission.attachmentMimeType
                        )
                    }
                } label: {
                    Label(
                        submission.attachmentFilename ?? L.text("mobile.assignment.attachment"),
                        systemImage: "paperclip"
                    )
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                }
                .padding(.top, 4)
            }

            if submission.bodyText?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty != false,
               !AssignmentLogic.hasAttachment(submission) {
                Text(L.text("mobile.peerReview.noSubmissionContent"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private var scoreCard: some View {
        LMSCard {
            Text(L.text("mobile.peerReview.score"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            TextField(L.text("mobile.peerReview.scorePlaceholder"), text: $scoreText)
                .keyboardType(.decimalPad)
                .padding(12)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
                .disabled(isSubmitted && !saving)
        }
    }

    private var commentsCard: some View {
        LMSCard {
            Text(L.text("mobile.peerReview.comments"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            TextField(L.text("mobile.peerReview.commentsPlaceholder"), text: $comments, axis: .vertical)
                .lineLimit(4 ... 8)
                .padding(12)
                .background(LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(LexturesTheme.fieldBorder(for: colorScheme), lineWidth: 1)
                )
                .disabled(isSubmitted && !saving)
        }
    }

    private var submitBar: some View {
        Button {
            Task { await submit() }
        } label: {
            if saving {
                ProgressView().frame(maxWidth: .infinity)
            } else {
                Text(isSubmitted ? L.text("mobile.peerReview.updateReview") : L.text("mobile.peerReview.submitReview"))
            }
        }
        .buttonStyle(AuthPrimaryButtonStyle())
        .disabled(!canSubmit || saving)
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
            detail = try await LMSAPI.fetchPeerReviewAllocation(allocationId: allocationId, accessToken: token)
            if let detail { applyDraft(from: detail) }
        } catch {
            errorMessage = L.text("mobile.peerReview.loadError")
        }
    }

    private func applyDraft(from detail: PeerReviewAllocationDetail) {
        if let review = detail.review {
            if let score = review.score {
                scoreText = score.formatted()
            }
            rubricScores = review.rubricScores ?? [:]
            comments = review.comments ?? ""
        }
    }

    private func submit() async {
        guard let token = session.accessToken, let detail else { return }
        saving = true
        errorMessage = nil
        successMessage = nil
        defer { saving = false }

        let trimmedComments = comments.trimmingCharacters(in: .whitespacesAndNewlines)
        var body = PeerReviewSubmitRequest(
            score: nil,
            rubricScores: nil,
            comments: trimmedComments.isEmpty ? nil : trimmedComments
        )
        if hasRubric, let rubric {
            body.rubricScores = rubricScores
            body.score = PeerReviewLogic.rubricTotal(rubric, scores: rubricScores)
        } else if let scoreValue {
            body.score = scoreValue
        }

        do {
            if NetworkMonitor.shared.isOnline {
                _ = try await LMSAPI.submitPeerReview(
                    allocationId: allocationId,
                    body: body,
                    accessToken: token
                )
            } else {
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: "/api/v1/peer-review/allocations/\(allocationId)",
                    body: body,
                    label: L.text("mobile.peerReview.queueLabel"),
                    accessToken: token,
                    preferQueue: true
                )
            }
            successMessage = L.text("mobile.peerReview.submitSuccess")
            onSubmitted()
            await load()
        } catch {
            errorMessage = L.text("mobile.peerReview.submitError")
        }
    }
}
