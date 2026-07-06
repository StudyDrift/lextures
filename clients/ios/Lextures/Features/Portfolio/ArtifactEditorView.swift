import PhotosUI
import SwiftUI
import UIKit
import UniformTypeIdentifiers

/// Add or edit a portfolio artifact: capture/upload, link, submission, reflection (M12.1).
struct ArtifactEditorView: View {
    @Environment(AuthSession.self) private var session

    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let portfolioId: String
    var existing: PortfolioArtifact?
    let onSaved: (PortfolioArtifact) -> Void

    @State private var kind: ArtifactEditorKind = .upload
    @State private var title = ""
    @State private var reflection = ""
    @State private var externalUrl = ""
    @State private var textContent = ""
    @State private var outcomeTags = ""
    @State private var isPublic = false
    @State private var saving = false
    @State private var errorMessage: String?

    @State private var pendingAttachment: PendingPortfolioAttachment?
    @State private var uploader = PortfolioArtifactUploader()
    @State private var showCamera = false
    @State private var showFileImporter = false
    @State private var photoPickerItem: PhotosPickerItem?

    @State private var enrolledCourses: [CourseSummary] = []
    @State private var submissionPickerCourse: CourseSummary?
    @State private var submissionOptions: [SubmissionPick] = []
    @State private var selectedSubmissionId: String?
    @State private var loadingSubmissions = false

    private var isEditing: Bool { existing != nil }

    init(portfolioId: String, existing: PortfolioArtifact? = nil, onSaved: @escaping (PortfolioArtifact) -> Void) {
        self.portfolioId = portfolioId
        self.existing = existing
        self.onSaved = onSaved
        if let existing {
            _title = State(initialValue: existing.title)
            _reflection = State(initialValue: existing.description)
            _externalUrl = State(initialValue: existing.externalUrl)
            _textContent = State(initialValue: existing.textContent)
            _outcomeTags = State(initialValue: existing.outcomeIds.joined(separator: ", "))
            _isPublic = State(initialValue: existing.isPublic)
            switch existing.artifactType {
            case "url": _kind = State(initialValue: .url)
            case "text_page": _kind = State(initialValue: .textPage)
            case "heading": _kind = State(initialValue: .heading)
            case "submission": _kind = State(initialValue: .submission)
            default: _kind = State(initialValue: .upload)
            }
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.coral)
                }

                if !isEditing {
                    Picker(L.text("mobile.portfolio.artifactKind"), selection: $kind) {
                        ForEach(ArtifactEditorKind.allCases) { k in
                            Text(k.label).tag(k)
                        }
                    }
                }

                TextField(L.text("mobile.portfolio.fieldTitle"), text: $title)
                    .accessibilityLabel(L.text("mobile.portfolio.fieldTitle"))

                TextField(L.text("mobile.portfolio.fieldReflection"), text: $reflection, axis: .vertical)
                    .lineLimit(3 ... 8)
                    .accessibilityLabel(L.text("mobile.portfolio.fieldReflection"))

                TextField(L.text("mobile.portfolio.fieldOutcomeTags"), text: $outcomeTags)
                    .accessibilityLabel(L.text("mobile.portfolio.fieldOutcomeTags"))

                Toggle(L.text("mobile.portfolio.artifactPublic"), isOn: $isPublic)

                kindFields
                uploadSection
                submissionSection
            }
            .navigationTitle(isEditing ? L.text("mobile.portfolio.editArtifact") : L.text("mobile.portfolio.addArtifact"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("common.cancel")) { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(isEditing ? L.text("common.save") : L.text("mobile.portfolio.add")) {
                        Task { await save() }
                    }
                    .disabled(saving || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .sheet(isPresented: $showCamera) {
                PortfolioCameraCaptureView { image in attachImage(image) }
            }
            .fileImporter(
                isPresented: $showFileImporter,
                allowedContentTypes: [.pdf, .image, .plainText, .data, .movie],
                allowsMultipleSelection: false
            ) { result in
                handleFileImport(result)
            }
            .onChange(of: photoPickerItem) { _, item in
                Task { await loadPhotoPickerItem(item) }
            }
            .onChange(of: submissionPickerCourse) { _, course in
                if course != nil { Task { await loadSubmissions(for: course!) } }
            }
            .task { await loadCourses() }
        }
    }

    @ViewBuilder
    private var kindFields: some View {
        switch kind {
        case .url:
            TextField(L.text("mobile.portfolio.fieldUrl"), text: $externalUrl)
                .textInputAutocapitalization(.never)
                .keyboardType(.URL)
        case .textPage:
            TextField(L.text("mobile.portfolio.fieldContent"), text: $textContent, axis: .vertical)
                .lineLimit(4 ... 12)
        case .heading, .upload, .submission:
            EmptyView()
        }
    }

    @ViewBuilder
    private var uploadSection: some View {
        if kind == .upload, !isEditing {
            Section(L.text("mobile.portfolio.attachFile")) {
                if let pendingAttachment {
                    Text(pendingAttachment.fileName)
                        .font(.caption)
                }
                PhotosPicker(selection: $photoPickerItem, matching: .any(of: [.images, .videos])) {
                    Label(L.text("mobile.assignment.photoLibrary"), systemImage: "photo.on.rectangle")
                }
                Button { showCamera = true } label: {
                    Label(L.text("mobile.assignment.camera"), systemImage: "camera.fill")
                }
                Button { showFileImporter = true } label: {
                    Label(L.text("mobile.assignment.chooseFile"), systemImage: "doc.fill")
                }
                uploadStatus
            }
        }
    }

    @ViewBuilder
    private var uploadStatus: some View {
        switch uploader.phase {
        case .idle:
            EmptyView()
        case .uploading(let progress):
            ProgressView(value: progress) {
                Text(L.text("mobile.portfolio.uploading"))
            }
        case .failed(let message):
            Text(message).font(.caption).foregroundStyle(LexturesTheme.coral)
        case .done:
            Label(L.text("mobile.portfolio.uploadDone"), systemImage: "checkmark.circle.fill")
                .font(.caption)
                .foregroundStyle(LexturesTheme.brandTeal)
        }
    }

    @ViewBuilder
    private var submissionSection: some View {
        if kind == .submission, !isEditing {
            Section(L.text("mobile.portfolio.fromSubmission")) {
                Picker(L.text("mobile.portfolio.pickCourse"), selection: $submissionPickerCourse) {
                    Text(L.text("mobile.portfolio.selectCourse")).tag(Optional<CourseSummary>.none)
                    ForEach(enrolledCourses, id: \.id) { course in
                        Text(course.title).tag(Optional(course))
                    }
                }
                if loadingSubmissions {
                    ProgressView()
                } else if submissionOptions.isEmpty, submissionPickerCourse != nil {
                    Text(L.text("mobile.portfolio.noSubmissions"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                } else {
                    Picker(L.text("mobile.portfolio.pickSubmission"), selection: $selectedSubmissionId) {
                        Text(L.text("mobile.portfolio.selectSubmission")).tag(Optional<String>.none)
                        ForEach(submissionOptions) { option in
                            Text(option.label).tag(Optional(option.submissionId))
                        }
                    }
                }
            }
        }
    }

    private func save() async {
        guard let token = session.accessToken else { return }
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTitle.isEmpty else { return }
        saving = true
        errorMessage = nil
        defer { saving = false }

        let outcomeIds = PortfolioLogic.parseOutcomeIds(outcomeTags)

        if let existing {
            do {
                let updated = try await LMSAPI.patchArtifact(
                    portfolioId: portfolioId,
                    artifactId: existing.id,
                    payload: PatchArtifactRequest(
                        title: trimmedTitle,
                        description: reflection.trimmingCharacters(in: .whitespacesAndNewlines),
                        textContent: kind == .textPage ? textContent : nil,
                        externalUrl: kind == .url ? externalUrl.trimmingCharacters(in: .whitespacesAndNewlines) : nil,
                        outcomeIds: outcomeIds.isEmpty ? nil : outcomeIds,
                        isPublic: isPublic
                    ),
                    accessToken: token
                )
                onSaved(updated)
                dismiss()
            } catch {
                errorMessage = L.text("mobile.portfolio.saveError")
            }
            return
        }

        switch kind {
        case .upload:
            guard let attachment = pendingAttachment else {
                errorMessage = L.text("mobile.portfolio.fileRequired")
                return
            }
            uploader.upload(
                portfolioId: portfolioId,
                fileData: attachment.data,
                fileName: attachment.fileName,
                mimeType: attachment.mimeType,
                title: trimmedTitle,
                description: reflection.trimmingCharacters(in: .whitespacesAndNewlines),
                outcomeIds: outcomeIds,
                isPublic: isPublic,
                accessToken: token
            ) { artifact in
                onSaved(artifact)
                dismiss()
            }
        default:
            do {
                let artifactType: String
                var payload = CreateArtifactRequest(
                    artifactType: "",
                    title: trimmedTitle,
                    description: reflection.trimmingCharacters(in: .whitespacesAndNewlines),
                    outcomeIds: outcomeIds.isEmpty ? nil : outcomeIds,
                    isPublic: isPublic
                )
                switch kind {
                case .url:
                    artifactType = "url"
                    payload.externalUrl = externalUrl.trimmingCharacters(in: .whitespacesAndNewlines)
                case .textPage:
                    artifactType = "text_page"
                    payload.textContent = textContent
                case .heading:
                    artifactType = "heading"
                case .submission:
                    artifactType = "submission"
                    guard let submissionId = selectedSubmissionId else {
                        errorMessage = L.text("mobile.portfolio.submissionRequired")
                        return
                    }
                    payload.sourceSubmissionId = submissionId
                case .upload:
                    return
                }
                payload.artifactType = artifactType
                let created = try await LMSAPI.createArtifact(
                    portfolioId: portfolioId,
                    payload: payload,
                    accessToken: token
                )
                onSaved(created)
                dismiss()
            } catch {
                errorMessage = L.text("mobile.portfolio.saveError")
            }
        }
    }

    private func loadCourses() async {
        guard let token = session.accessToken else { return }
        if let courses = try? await LMSAPI.fetchCourses(accessToken: token) {
            enrolledCourses = courses.filter(\.viewerIsStudent)
        }
    }

    private func loadSubmissions(for course: CourseSummary) async {
        guard let token = session.accessToken else { return }
        loadingSubmissions = true
        submissionOptions = []
        selectedSubmissionId = nil
        defer { loadingSubmissions = false }
        do {
            let structure = try await LMSAPI.fetchCourseStructure(courseCode: course.courseCode, accessToken: token)
            var picks: [SubmissionPick] = []
            for item in structure where item.kind == "assignment" {
                if let submission = try? await LMSAPI.fetchMySubmission(
                    courseCode: course.courseCode,
                    itemId: item.id,
                    accessToken: token
                ) {
                    picks.append(SubmissionPick(
                        submissionId: submission.id,
                        label: "\(item.title) · v\(submission.versionNumber ?? 1)"
                    ))
                }
            }
            submissionOptions = picks
        } catch {
            submissionOptions = []
        }
    }

    private func attachImage(_ image: UIImage) {
        guard let data = image.jpegData(compressionQuality: 0.85) else { return }
        pendingAttachment = PendingPortfolioAttachment(
            data: data,
            fileName: "photo-\(Int(Date().timeIntervalSince1970)).jpg",
            mimeType: "image/jpeg"
        )
    }

    private func handleFileImport(_ result: Result<[URL], Error>) {
        guard case .success(let urls) = result, let url = urls.first else { return }
        guard url.startAccessingSecurityScopedResource() else { return }
        defer { url.stopAccessingSecurityScopedResource() }
        guard let data = try? Data(contentsOf: url) else { return }
        let mime = UTType(filenameExtension: url.pathExtension)?.preferredMIMEType ?? "application/octet-stream"
        pendingAttachment = PendingPortfolioAttachment(data: data, fileName: url.lastPathComponent, mimeType: mime)
    }

    private func loadPhotoPickerItem(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        if let data = try? await item.loadTransferable(type: Data.self) {
            pendingAttachment = PendingPortfolioAttachment(
                data: data,
                fileName: "media-\(Int(Date().timeIntervalSince1970)).jpg",
                mimeType: "image/jpeg"
            )
        }
        photoPickerItem = nil
    }
}

private enum ArtifactEditorKind: String, CaseIterable, Identifiable {
    case upload, url, textPage, heading, submission

    var id: String { rawValue }

    var label: String {
        switch self {
        case .upload: return L.text("mobile.portfolio.kind.upload")
        case .url: return L.text("mobile.portfolio.kind.url")
        case .textPage: return L.text("mobile.portfolio.kind.textPage")
        case .heading: return L.text("mobile.portfolio.kind.heading")
        case .submission: return L.text("mobile.portfolio.kind.submission")
        }
    }
}

private struct PendingPortfolioAttachment {
    let data: Data
    let fileName: String
    let mimeType: String
}

private struct SubmissionPick: Identifiable {
    let submissionId: String
    let label: String
    var id: String { submissionId }
}

/// UIKit camera wrapper (M5.1 pattern) for portfolio photo capture.
private struct PortfolioCameraCaptureView: UIViewControllerRepresentable {
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
