import PhotosUI
import SwiftUI
import UIKit
import UniformTypeIdentifiers

/// Student assignment detail: instructions, compose, submit, status, feedback link (M5.1).
struct AssignmentDetailView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary
    let item: CourseStructureItem

    private var courseCode: String { course.courseCode }
    private var resubmissionEnabled: Bool { course.resubmissionWorkflowEnabled == true }
    private var draftKey: String { AssignmentLogic.draftStorageKey(courseCode: courseCode, itemId: item.id) }

    @State private var detail: ModuleItemDetail?
    @State private var mySubmission: AssignmentSubmission?
    @State private var myGrade: SubmissionGrade?
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var draftText = ""
    @State private var urlText = ""
    @State private var submitting = false
    @State private var submitSuccess: String?
    @State private var pendingAttachment: PendingAttachment?
    @State private var uploader = AttachmentUploader()
    @State private var feedbackRoute: GradeFeedbackRoute?
    @State private var receivedReviewsRoute: PeerReviewsReceivedRoute?
    @State private var previewTarget: FilePreviewTarget?
    @State private var showCamera = false
    @State private var showFileImporter = false
    @State private var photoPickerItem: PhotosPickerItem?

    private var status: AssignmentSubmissionStatus {
        AssignmentLogic.status(submission: mySubmission, grade: myGrade, detail: detail)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let submitSuccess {
                        LMSCard(accent: LexturesTheme.brandTeal) {
                            Label(submitSuccess, systemImage: "checkmark.circle.fill")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.primary)
                        }
                    }

                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 40)
                    } else {
                        header
                        if let markdown = detail?.markdown,
                           !markdown.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            instructionsCard(markdown)
                        }
                        statusCard
                        if canShowComposer {
                            composerCard
                        }
                        if status == .graded || (myGrade?.posted == true) {
                            feedbackLink
                        }
                        if shell.platformFeatures.ffPeerReview, mySubmission != nil {
                            peerReviewReceivedLink
                        }
                        detailsCard
                    }
                }
                .padding(16)
            }
            .refreshable { await load() }
        }
        .navigationTitle(detail?.title ?? item.title)
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(item: $feedbackRoute) { route in
            GradeFeedbackView(course: course, column: route.column)
        }
        .navigationDestination(item: $receivedReviewsRoute) { route in
            ReviewsReceivedView(
                courseCode: route.courseCode,
                assignmentId: route.assignmentId,
                assignmentTitle: route.assignmentTitle
            )
        }
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
        .sheet(isPresented: $showCamera) {
            CameraCaptureView { image in
                attachImage(image)
            }
        }
        .fileImporter(
            isPresented: $showFileImporter,
            allowedContentTypes: [.pdf, .image, .plainText, .data],
            allowsMultipleSelection: false
        ) { result in
            handleFileImport(result)
        }
        .onChange(of: photoPickerItem) { _, item in
            Task { await loadPhotoPickerItem(item) }
        }
        .task {
            draftText = AssignmentDraftStore.load(key: draftKey)
            await load()
        }
        .onChange(of: draftText) { _, text in
            AssignmentDraftStore.save(key: draftKey, text: text)
        }
    }

    private var canShowComposer: Bool {
        AssignmentLogic.canSubmit(
            detail: detail,
            submission: mySubmission,
            resubmissionWorkflowEnabled: resubmissionEnabled
        ) || pendingAttachment != nil
    }

    // MARK: Header

    private var header: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                chip(text: L.text("mobile.assignment.label"), icon: "doc.text.fill", tint: LexturesTheme.accent(for: colorScheme))
                if let due = LMSDates.parse(detail?.dueAt ?? item.dueAt) {
                    let late = due < Date() && mySubmission == nil
                    chip(
                        text: L.format("mobile.assignment.due", due.formatted(date: .abbreviated, time: .shortened)),
                        icon: "clock.fill",
                        tint: late ? LexturesTheme.coral : LexturesTheme.coral
                    )
                }
                if let points = detail?.pointsWorth ?? item.pointsWorth.map({ Int($0) }) {
                    chip(text: L.format("mobile.assignment.points", points), icon: "star.fill", tint: LexturesTheme.amber)
                }
            }
            Text(detail?.title ?? item.title)
                .font(LexturesTheme.displayFont(24))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
    }

    private func chip(text: String, icon: String, tint: Color) -> some View {
        Label(text, systemImage: icon)
            .font(.caption.weight(.semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 9)
            .padding(.vertical, 4)
            .background(tint.opacity(0.13))
            .clipShape(Capsule())
    }

    // MARK: Instructions

    private func instructionsCard(_ markdown: String) -> some View {
        LMSCard {
            readerToolbar(markdown: markdown)
            MarkdownTextView(markdown: markdown)
                .lexturesReadableText()
        }
    }

    @ViewBuilder
    private func readerToolbar(markdown: String) -> some View {
        let caps = shell.platformFeatures.immersiveReader
        if caps.toolbarEnabled {
            ReaderToolbar(
                text: markdown,
                courseCode: courseCode,
                readAloudEnabled: caps.readAloudEnabled,
                translationEnabled: caps.translationEnabled,
                preferencesEnabled: caps.preferencesEnabled
            )
        } else {
            ReadAloudButton(text: markdown)
        }
    }

    // MARK: Status

    private var statusCard: some View {
        LMSCard(accent: statusAccent) {
            Text(L.text("mobile.assignment.yourWork"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            switch status {
            case .notStarted:
                Label(L.text("mobile.assignment.notSubmitted"), systemImage: "tray")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            case .submitted, .late:
                submittedSummary(late: status == .late)
            case .revisionRequested:
                revisionBanner
            case .graded:
                submittedSummary(late: AssignmentLogic.isLate(detail: detail, submittedAt: mySubmission?.submittedAt))
                gradedSummary
            }

            if let submission = mySubmission, AssignmentLogic.hasAttachment(submission) {
                attachmentPreviewRow(submission)
            }
        }
    }

    private var statusAccent: Color {
        switch status {
        case .revisionRequested: return LexturesTheme.coral
        case .graded: return LexturesTheme.brandTeal
        default: return LexturesTheme.primary
        }
    }

    @ViewBuilder
    private func submittedSummary(late: Bool) -> some View {
        if let submission = mySubmission {
            HStack(spacing: 10) {
                Image(systemName: late ? "exclamationmark.circle.fill" : "checkmark.circle.fill")
                    .foregroundStyle(late ? LexturesTheme.coral : LexturesTheme.primary)
                VStack(alignment: .leading, spacing: 2) {
                    Text(
                        late
                            ? L.text("mobile.assignment.submittedLate")
                            : L.format("mobile.assignment.submittedAt", LMSDates.shortDateTime(submission.submittedAt))
                    )
                    .font(.subheadline.weight(.medium))
                    if let version = submission.versionNumber, version > 1 {
                        Text(L.format("mobile.assignment.version", version))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
            }
        }
    }

    private var revisionBanner: some View {
        VStack(alignment: .leading, spacing: 6) {
            Label(L.text("mobile.assignment.revisionRequested"), systemImage: "arrow.uturn.backward.circle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.coral)
            if let feedback = mySubmission?.revisionFeedback, !feedback.isEmpty {
                Text(feedback)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            if let revisionDue = LMSDates.parse(mySubmission?.revisionDueAt) {
                Text(L.format("mobile.assignment.revisionDue", revisionDue.formatted(date: .abbreviated, time: .shortened)))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.coral)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(LexturesTheme.coral.opacity(0.08))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    @ViewBuilder
    private var gradedSummary: some View {
        if let grade = myGrade, grade.posted == true, let earned = grade.pointsEarned {
            Divider()
            HStack {
                Text(L.text("mobile.assignment.grade"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Spacer()
                Text(gradeText(earned: earned, max: grade.maxPoints))
                    .font(LexturesTheme.displayFont(18, weight: .bold))
                    .foregroundStyle(LexturesTheme.primary)
            }
        }
    }

    private func attachmentPreviewRow(_ submission: AssignmentSubmission) -> some View {
        Button {
            if let path = submission.attachmentContentPath,
               let name = submission.attachmentFilename {
                previewTarget = FilePreviewTarget.submissionContentPath(
                    courseCode: courseCode,
                    contentPath: path,
                    fileName: name,
                    mimeType: submission.attachmentMimeType
                )
            }
        } label: {
            Label(submission.attachmentFilename ?? L.text("mobile.assignment.attachment"), systemImage: "paperclip")
                .font(.caption)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
        }
        .padding(.top, 4)
    }

    // MARK: Composer

    private var composerCard: some View {
        LMSCard {
            Text(L.text("mobile.assignment.submitWork"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if let reasonKey = AssignmentLogic.submitDisabledReasonKey(
                detail: detail,
                submission: mySubmission,
                resubmissionWorkflowEnabled: resubmissionEnabled
            ), !canSubmitNow {
                Text(L.dynamicText(reasonKey))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
            }

            if AssignmentLogic.canSubmitText(
                detail: detail,
                submission: mySubmission,
                resubmissionWorkflowEnabled: resubmissionEnabled
            ) {
                TextEditor(text: $draftText)
                    .frame(minHeight: 120)
                    .overlay(alignment: .topLeading) {
                        if draftText.isEmpty {
                            Text(L.text("mobile.assignment.textPlaceholder"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .padding(.top, 8)
                                .padding(.leading, 4)
                                .allowsHitTesting(false)
                        }
                    }
                    .accessibilityLabel(L.text("mobile.assignment.textPlaceholder"))
            }

            if detail?.submissionAllowUrl == true {
                TextField(L.text("mobile.assignment.urlPlaceholder"), text: $urlText)
                    .textInputAutocapitalization(.never)
                    .keyboardType(.URL)
                    .autocorrectionDisabled()
            }

            if AssignmentLogic.canSubmitFile(
                detail: detail,
                submission: mySubmission,
                resubmissionWorkflowEnabled: resubmissionEnabled
            ) {
                attachmentPickers
                uploadProgress
                if let pending = pendingAttachment {
                    Label(pending.fileName, systemImage: "doc.fill")
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }

            if !NetworkMonitor.shared.isOnline {
                Label(L.text("mobile.assignment.offlineHint"), systemImage: "icloud.and.arrow.up")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.amber)
            }

            Button(submitButtonTitle) {
                Task { await submit() }
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .disabled(!canSubmitNow || submitting || isUploading)
        }
    }

    private var attachmentPickers: some View {
        HStack(spacing: 10) {
            PhotosPicker(selection: $photoPickerItem, matching: .any(of: [.images, .videos])) {
                Label(L.text("mobile.assignment.pickPhoto"), systemImage: "photo.on.rectangle")
                    .font(.caption.weight(.semibold))
            }
            Button {
                showCamera = true
            } label: {
                Label(L.text("mobile.assignment.camera"), systemImage: "camera")
                    .font(.caption.weight(.semibold))
            }
            Button {
                showFileImporter = true
            } label: {
                Label(L.text("mobile.assignment.pickFile"), systemImage: "folder")
                    .font(.caption.weight(.semibold))
            }
        }
        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
        .padding(.vertical, 4)
    }

    @ViewBuilder
    private var uploadProgress: some View {
        switch uploader.phase {
        case .idle:
            EmptyView()
        case .uploading(let progress):
            ProgressView(value: progress)
                .accessibilityLabel(L.text("mobile.assignment.uploading"))
        case .failed(let message):
            VStack(alignment: .leading, spacing: 6) {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.coral)
                Button(L.text("mobile.common.retry")) {
                    retryUpload()
                }
                .font(.caption.weight(.semibold))
            }
        case .done:
            Label(L.text("mobile.assignment.uploadComplete"), systemImage: "checkmark.circle")
                .font(.caption)
                .foregroundStyle(LexturesTheme.brandTeal)
        }
    }

    private var feedbackLink: some View {
        Button {
            feedbackRoute = GradeFeedbackRoute(column: AssignmentLogic.gradeColumn(item: item, detail: detail))
        } label: {
            LMSCard(accent: LexturesTheme.brandTeal) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(L.text("mobile.assignment.viewFeedback"))
                            .font(.subheadline.weight(.semibold))
                        Text(L.text("mobile.assignment.viewFeedbackHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }

    private var peerReviewReceivedLink: some View {
        Button {
            receivedReviewsRoute = PeerReviewsReceivedRoute(
                courseCode: courseCode,
                assignmentId: item.id,
                assignmentTitle: detail?.title ?? item.title
            )
        } label: {
            LMSCard {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(L.text("mobile.peerReview.receivedLink"))
                            .font(.subheadline.weight(.semibold))
                        Text(L.text("mobile.peerReview.receivedLinkHint"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer()
                    Image(systemName: "chevron.right")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
        .buttonStyle(.plain)
    }

    private var detailsCard: some View {
        let rows = ItemDetailRows.rows(for: item, detail: detail, pointsValue: detail?.pointsWorth ?? item.pointsWorth.map { Int($0) })
        return Group {
            if !rows.isEmpty {
                LMSCard {
                    Text(L.text("mobile.assignment.details"))
                        .font(LexturesTheme.displayFont(18))
                    Divider().padding(.vertical, 2)
                    ForEach(rows, id: \.0) { label, value in
                        HStack {
                            Text(label)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Spacer()
                            Text(value)
                                .font(.subheadline.weight(.semibold))
                                .multilineTextAlignment(.trailing)
                        }
                        .padding(.vertical, 3)
                    }
                    let typeKeys = AssignmentLogic.submissionTypeLabelKeys(detail: detail)
                    if !typeKeys.isEmpty {
                        HStack {
                            Text(L.text("mobile.assignment.allowedTypes"))
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Spacer()
                            Text(typeKeys.map { L.dynamicText($0) }.joined(separator: ", "))
                                .font(.subheadline.weight(.semibold))
                                .multilineTextAlignment(.trailing)
                        }
                        .padding(.vertical, 3)
                    }
                }
            }
        }
    }

    // MARK: Submit logic

    private var canSubmitNow: Bool {
        if isUploading { return false }
        if pendingAttachment != nil { return true }
        let text = composedText
        if !text.isEmpty {
            return AssignmentLogic.canSubmit(
                detail: detail,
                submission: mySubmission,
                resubmissionWorkflowEnabled: resubmissionEnabled
            )
        }
        return false
    }

    private var composedText: String {
        let body = draftText.trimmingCharacters(in: .whitespacesAndNewlines)
        let url = urlText.trimmingCharacters(in: .whitespacesAndNewlines)
        if detail?.submissionAllowUrl == true, !url.isEmpty {
            if body.isEmpty { return url }
            return "\(body)\n\n\(url)"
        }
        return body
    }

    private var submitButtonTitle: String {
        if submitting { return L.text("mobile.assignment.submitting") }
        if mySubmission != nil { return L.text("mobile.assignment.resubmit") }
        return L.text("mobile.assignment.submit")
    }

    private var isUploading: Bool {
        if case .uploading = uploader.phase { return true }
        return false
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            detail = try await LMSAPI.fetchItemDetail(courseCode: courseCode, item: item, accessToken: token)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.assignment.loadError")
        }
        mySubmission = try? await LMSAPI.fetchMySubmission(courseCode: courseCode, itemId: item.id, accessToken: token)
        if let submission = mySubmission {
            myGrade = try? await LMSAPI.fetchSubmissionGrade(
                courseCode: courseCode,
                itemId: item.id,
                submissionId: submission.id,
                accessToken: token
            )
        }
    }

    private func submit() async {
        guard let token = session.accessToken else { return }
        submitting = true
        submitSuccess = nil
        errorMessage = nil
        defer { submitting = false }

        if let attachment = pendingAttachment {
            uploader.upload(
                courseCode: courseCode,
                itemId: item.id,
                fileData: attachment.data,
                fileName: attachment.fileName,
                mimeType: attachment.mimeType,
                accessToken: token
            ) { submission in
                Task { @MainActor in
                    pendingAttachment = nil
                    finishSubmit(submission: submission)
                }
            }
            return
        }

        let text = composedText
        guard !text.isEmpty else { return }

        if !NetworkMonitor.shared.isOnline {
            do {
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: "/api/v1/courses/\(courseCode)/assignments/\(item.id)/submissions/text",
                    body: SubmitAssignmentTextRequest(text: text),
                    label: L.text("mobile.assignment.saveAnswer"),
                    accessToken: token,
                    preferQueue: true
                )
                submitSuccess = L.text("mobile.assignment.queuedOffline")
            } catch {
                errorMessage = (error as? LocalizedError)?.errorDescription
            }
            return
        }

        do {
            let submission = try await LMSAPI.submitAssignmentText(
                courseCode: courseCode,
                itemId: item.id,
                text: text,
                accessToken: token
            )
            AssignmentDraftStore.clear(key: draftKey)
            draftText = ""
            urlText = ""
            finishSubmit(submission: submission)
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.assignment.submitError")
        }
    }

    private func finishSubmit(submission: AssignmentSubmission) {
        mySubmission = submission
        submitSuccess = L.format(
            "mobile.assignment.submitSuccess",
            submission.versionNumber ?? 1,
            LMSDates.relative(submission.submittedAt)
        )
        Task { await load() }
    }

    private func retryUpload() {
        guard let attachment = pendingAttachment, let token = session.accessToken else { return }
        uploader.upload(
            courseCode: courseCode,
            itemId: item.id,
            fileData: attachment.data,
            fileName: attachment.fileName,
            mimeType: attachment.mimeType,
            accessToken: token
        ) { submission in
            Task { @MainActor in
                pendingAttachment = nil
                finishSubmit(submission: submission)
            }
        }
    }

    private func attachImage(_ image: UIImage) {
        guard let data = image.jpegData(compressionQuality: 0.82) else { return }
        guard AssignmentLogic.isAllowedFileSize(Int64(data.count)) else {
            errorMessage = L.text("mobile.assignment.fileTooLarge")
            return
        }
        pendingAttachment = PendingAttachment(data: data, fileName: "photo.jpg", mimeType: "image/jpeg")
    }

    private func handleFileImport(_ result: Result<[URL], Error>) {
        guard case .success(let urls) = result, let url = urls.first else { return }
        guard url.startAccessingSecurityScopedResource() else { return }
        defer { url.stopAccessingSecurityScopedResource() }
        guard let data = try? Data(contentsOf: url) else { return }
        guard AssignmentLogic.isAllowedFileSize(Int64(data.count)) else {
            errorMessage = L.text("mobile.assignment.fileTooLarge")
            return
        }
        let mime = UTType(filenameExtension: url.pathExtension)?.preferredMIMEType ?? "application/octet-stream"
        guard AssignmentLogic.isAllowedMimeType(mime) else {
            errorMessage = L.text("mobile.assignment.fileTypeNotAllowed")
            return
        }
        pendingAttachment = PendingAttachment(data: data, fileName: url.lastPathComponent, mimeType: mime)
    }

    private func loadPhotoPickerItem(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        guard let data = try? await item.loadTransferable(type: Data.self) else { return }
        guard AssignmentLogic.isAllowedFileSize(Int64(data.count)) else {
            await MainActor.run { errorMessage = L.text("mobile.assignment.fileTooLarge") }
            return
        }
        let mime = item.supportedContentTypes.first?.preferredMIMEType ?? "image/jpeg"
        let name = item.supportedContentTypes.first?.preferredFilenameExtension.map { "photo.\($0)" } ?? "photo.jpg"
        await MainActor.run {
            pendingAttachment = PendingAttachment(data: data, fileName: name, mimeType: mime)
            photoPickerItem = nil
        }
    }

    private func gradeText(earned: Double, max: Double?) -> String {
        if let max { return "\(earned.formatted()) / \(max.formatted())" }
        return earned.formatted()
    }
}

private struct PendingAttachment {
    var data: Data
    var fileName: String
    var mimeType: String
}

/// UIKit camera wrapper for capturing assignment photos.
private struct CameraCaptureView: UIViewControllerRepresentable {
    var onCapture: (UIImage) -> Void
    @Environment(\.dismiss) private var dismiss

    func makeUIViewController(context: Context) -> UIImagePickerController {
        let picker = UIImagePickerController()
        picker.sourceType = .camera
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: UIImagePickerController, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(onCapture: onCapture, dismiss: dismiss) }

    final class Coordinator: NSObject, UIImagePickerControllerDelegate, UINavigationControllerDelegate {
        let onCapture: (UIImage) -> Void
        let dismiss: DismissAction

        init(onCapture: @escaping (UIImage) -> Void, dismiss: DismissAction) {
            self.onCapture = onCapture
            self.dismiss = dismiss
        }

        func imagePickerController(
            _ picker: UIImagePickerController,
            didFinishPickingMediaWithInfo info: [UIImagePickerController.InfoKey: Any]
        ) {
            if let image = info[.originalImage] as? UIImage {
                onCapture(image)
            }
            dismiss()
        }

        func imagePickerControllerDidCancel(_ picker: UIImagePickerController) {
            dismiss()
        }
    }
}
