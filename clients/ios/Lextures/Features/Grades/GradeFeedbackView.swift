import AVKit
import PDFKit
import SwiftUI

/// Full student feedback: rubric, comments, annotated file, a/v playback (M6.1).
struct GradeFeedbackView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let column: GradeColumn

    @State private var grade: SubmissionGrade?
    @State private var submission: AssignmentSubmission?
    @State private var annotations: [SubmissionAnnotation] = []
    @State private var feedbackMedia: [SubmissionFeedbackMedia] = []
    @State private var playbackURLs: [String: URL] = [:]
    @State private var previewTarget: FilePreviewTarget?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var feedbackMediaEnabled = true

    private var rubric: RubricDefinition? { column.rubric }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                ProgressView("Loading feedback…")
            } else if let errorMessage {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: column.title,
                    message: errorMessage
                )
                .padding(24)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        scoreHeader
                        if grade?.gradedByAi == true {
                            aiDisclosure
                        }
                        if let rubric, !(rubric.criteria.isEmpty) {
                            rubricSection(rubric)
                        }
                        if let comment = grade?.instructorComment?.trimmingCharacters(in: .whitespacesAndNewlines),
                           !comment.isEmpty {
                            commentCard("Instructor comment", body: comment)
                        }
                        if let comments = grade?.comments, !comments.isEmpty {
                            commentsSection(comments)
                        }
                        if feedbackMediaEnabled, !feedbackMedia.isEmpty {
                            feedbackMediaSection
                        }
                        if let submission {
                            submissionSection(submission)
                        }
                        if column.kind == "quiz" {
                            quizLink
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(column.title)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $previewTarget) { target in
            AnnotatedFilePreviewView(target: target, annotations: annotations)
        }
        .task { await load() }
    }

    // MARK: Sections

    private var scoreHeader: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text("Your grade")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if grade?.excused == true {
                    Text("Excused")
                        .font(LexturesTheme.displayFont(22, weight: .bold))
                } else if let earned = grade?.pointsEarned, let max = grade?.maxPoints ?? column.maxPoints {
                    Text("\(formatPoints(earned)) / \(formatPoints(max))")
                        .font(LexturesTheme.displayFont(22, weight: .bold))
                    if max > 0 {
                        Text(String(format: "%.1f%%", earned / max * 100))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    }
                } else {
                    Text("Not graded")
                        .font(LexturesTheme.displayFont(18, weight: .semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .accessibilityElement(children: .combine)
    }

    private var aiDisclosure: some View {
        LMSCard(accent: LexturesTheme.amber) {
            Label("Graded with AI assistance", systemImage: "sparkles")
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    private func rubricSection(_ rubric: RubricDefinition) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(rubric.title?.nilIfEmpty ?? "Rubric")
                    .font(LexturesTheme.displayFont(18))
                ForEach(rubric.criteria) { criterion in
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(criterion.title)
                                .font(.subheadline.weight(.semibold))
                            Spacer()
                            if let score = grade?.rubricScores?[criterion.id] {
                                Text(formatPoints(score))
                                    .font(.subheadline.weight(.bold))
                                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            } else {
                                Text("—")
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                        if let description = criterion.description?.nilIfEmpty {
                            Text(description)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        if let score = grade?.rubricScores?[criterion.id],
                           let level = matchedLevel(criterion: criterion, score: score) {
                            Text(level.label)
                                .font(.caption.weight(.medium))
                            if let note = level.description?.nilIfEmpty {
                                Text(note)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel(rubricAccessibilityLabel(criterion: criterion))
                }
            }
        }
    }

    private func commentCard(_ title: String, body: String) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                Text(title)
                    .font(LexturesTheme.displayFont(16))
                Text(body)
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
        }
    }

    private func commentsSection(_ comments: [GradeComment]) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("Feedback thread")
                    .font(LexturesTheme.displayFont(18))
                ForEach(comments, id: \.resolvedId) { comment in
                    VStack(alignment: .leading, spacing: 3) {
                        if let name = comment.displayName?.nilIfEmpty {
                            Text(name)
                                .font(.caption.weight(.semibold))
                        }
                        Text(comment.body)
                            .font(.subheadline)
                    }
                    .padding(.vertical, 2)
                }
            }
        }
    }

    private var feedbackMediaSection: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text("Audio / video feedback")
                    .font(LexturesTheme.displayFont(18))
                ForEach(feedbackMedia) { media in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(media.mediaType.capitalized)
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        if let url = playbackURLs[media.id], let token = session.accessToken {
                            if media.mediaType == "video" {
                                VideoPlayer(player: AVPlayer(url: url, headers: ["Authorization": "Bearer \(token)"]))
                                    .frame(height: 180)
                                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                            } else {
                                AudioFeedbackPlayer(url: url, accessToken: token)
                            }
                        } else {
                            ProgressView()
                                .frame(maxWidth: .infinity)
                        }
                    }
                    .accessibilityLabel("\(media.mediaType) feedback")
                }
            }
        }
    }

    private func submissionSection(_ submission: AssignmentSubmission) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text("Your submission")
                    .font(LexturesTheme.displayFont(18))
                if let filename = submission.attachmentFilename?.nilIfEmpty {
                    Button {
                        openSubmissionFile(submission, filename: filename)
                    } label: {
                        Label(
                            annotations.isEmpty ? "View submitted file" : "View file with annotations",
                            systemImage: "doc.viewfinder"
                        )
                    }
                    .buttonStyle(.bordered)
                }
                if let body = submission.bodyText?.trimmingCharacters(in: .whitespacesAndNewlines), !body.isEmpty {
                    Text(body)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private var quizLink: some View {
        LMSCard {
            Label("View quiz activity", systemImage: "questionmark.circle")
                .font(.subheadline.weight(.medium))
        }
    }

    // MARK: Load

    private func load() async {
        guard let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        errorMessage = nil
        defer { loading = false }

        if let features = try? await LMSAPI.fetchPlatformFeatures(accessToken: token) {
            feedbackMediaEnabled = features.feedbackMediaEnabled != false
        }

        guard column.kind == "assignment" else { return }

        submission = try? await LMSAPI.fetchMySubmission(
            courseCode: course.courseCode,
            itemId: column.id,
            accessToken: token
        )
        guard let submission else { return }

        do {
            grade = try await LMSAPI.fetchSubmissionGrade(
                courseCode: course.courseCode,
                itemId: column.id,
                submissionId: submission.id,
                accessToken: token
            )
            if grade?.posted == false {
                errorMessage = "This grade has not been released yet."
                grade = nil
                return
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load grade."
            return
        }

        annotations = (try? await LMSAPI.fetchSubmissionAnnotations(
            courseCode: course.courseCode,
            itemId: column.id,
            submissionId: submission.id,
            accessToken: token
        )) ?? []

        if feedbackMediaEnabled {
            feedbackMedia = (try? await LMSAPI.fetchSubmissionFeedbackMedia(
                courseCode: course.courseCode,
                itemId: column.id,
                submissionId: submission.id,
                accessToken: token
            )) ?? []
            for media in feedbackMedia {
                if let info = try? await LMSAPI.fetchFeedbackPlaybackInfo(
                    courseCode: course.courseCode,
                    itemId: column.id,
                    submissionId: submission.id,
                    mediaId: media.id,
                    accessToken: token
                ) {
                    playbackURLs[media.id] = AppConfiguration.apiURL(path: info.contentPath)
                }
            }
        }
    }

    private func openSubmissionFile(_ submission: AssignmentSubmission, filename: String) {
        if let path = submission.attachmentContentPath?.trimmingCharacters(in: .whitespacesAndNewlines),
           !path.isEmpty {
            previewTarget = FilePreviewTarget.submissionContentPath(
                courseCode: course.courseCode,
                contentPath: path,
                fileName: filename,
                mimeType: submission.attachmentMimeType
            )
        }
    }

    // MARK: Helpers

    private func matchedLevel(criterion: RubricCriterion, score: Double) -> RubricLevel? {
        criterion.levels.min(by: { abs($0.points - score) < abs($1.points - score) })
    }

    private func rubricAccessibilityLabel(criterion: RubricCriterion) -> String {
        let score = grade?.rubricScores?[criterion.id]
        return "\(criterion.title), \(score.map(formatPoints) ?? "not scored") points"
    }

    private func formatPoints(_ value: Double) -> String {
        value.truncatingRemainder(dividingBy: 1) == 0
            ? String(format: "%.0f", value)
            : String(format: "%.1f", value)
    }
}

/// File preview with optional instructor markup overlay.
struct AnnotatedFilePreviewView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let target: FilePreviewTarget
    let annotations: [SubmissionAnnotation]

    @State private var loading = true
    @State private var previewData: Data?
    @State private var errorMessage: String?

    private var previewKind: FilePreviewKind {
        CourseFileLogic.previewKind(mimeType: target.mimeType, fileName: target.displayName)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            if loading {
                ProgressView()
            } else if let errorMessage {
                Text(errorMessage).padding()
            } else {
                previewStack
            }
        }
        .navigationTitle(target.displayName)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    @ViewBuilder
    private var previewStack: some View {
        switch previewKind {
        case .image:
            if let previewData, let uiImage = UIImage(data: previewData) {
                ZStack {
                    ZoomableImagePreview(image: uiImage)
                    MarkupOverlayView(annotations: annotations)
                }
            }
        case .pdf:
            if let previewData {
                ZStack {
                    PDFKitPreview(data: previewData)
                    MarkupOverlayView(annotations: annotations)
                }
            }
        default:
            FilePreviewView(target: target)
        }
    }

    private func load() async {
        guard previewKind == .image || previewKind == .pdf else {
            loading = false
            return
        }
        guard let token = session.accessToken else {
            errorMessage = "Not signed in."
            loading = false
            return
        }
        do {
            previewData = try await FileDownloadManager.fetchData(
                courseCode: target.courseCode,
                target: target,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Could not load file."
        }
        loading = false
    }
}

private struct ZoomableImagePreview: View {
    let image: UIImage
    @State private var scale: CGFloat = 1

    var body: some View {
        ScrollView([.horizontal, .vertical]) {
            Image(uiImage: image)
                .resizable()
                .scaledToFit()
                .scaleEffect(scale)
        }
        .gesture(MagnificationGesture().onChanged { scale = max(1, $0) })
    }
}

private struct PDFKitPreview: UIViewRepresentable {
    let data: Data

    func makeUIView(context: Context) -> PDFView {
        let view = PDFView()
        view.autoScales = true
        view.displayMode = .singlePageContinuous
        view.document = PDFDocument(data: data)
        return view
    }

    func updateUIView(_ uiView: PDFView, context: Context) {}
}

private struct AudioFeedbackPlayer: View {
    let url: URL
    let accessToken: String

    var body: some View {
        VideoPlayer(player: AVPlayer(url: url, headers: ["Authorization": "Bearer \(accessToken)"]))
            .frame(height: 48)
    }
}

private extension AVPlayer {
    convenience init(url: URL, headers: [String: String]) {
        let asset = AVURLAsset(url: url, options: ["AVURLAssetHTTPHeaderFieldsKey": headers])
        self.init(playerItem: AVPlayerItem(asset: asset))
    }
}

private extension String {
    var nilIfEmpty: String? {
        isEmpty ? nil : self
    }
}
