import AVFoundation
import PhotosUI
import SwiftUI
import UniformTypeIdentifiers

/// Bottom-sheet composer for board posts (VC.M2).
struct BoardComposerView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courseCode: String
    let boardId: String
    var onCreated: (BoardPost) -> Void

    @State private var contentType: BoardContentType = .text
    @State private var title = ""
    @State private var bodyText = ""
    @State private var linkUrl = ""
    @State private var altText = ""
    @State private var photoItem: PhotosPickerItem?
    @State private var pickedFileName = ""
    @State private var pickedMime = "application/octet-stream"
    @State private var pickedData: Data?
    @State private var showCamera = false
    @State private var showDocumentPicker = false
    @State private var showMicExplainer = false
    @State private var showCameraExplainer = false
    @State private var isRecording = false
    @State private var recorder: AVAudioRecorder?
    @State private var recordURL: URL?
    @State private var uploadProgress: Double?
    @State private var submitting = false
    @State private var errorMessage: String?
    @State private var validationHint: String?

    private let composeTypes: [BoardContentType] = [.text, .image, .link, .file, .audio]

    private func label(for type: BoardContentType) -> String {
        switch type {
        case .text: return L.text("mobile.boards.post.type.text")
        case .image: return L.text("mobile.boards.post.type.image")
        case .file: return L.text("mobile.boards.post.type.file")
        case .link: return L.text("mobile.boards.post.type.link")
        case .video: return L.text("mobile.boards.post.type.video")
        case .audio: return L.text("mobile.boards.post.type.audio")
        case .drawing: return L.text("mobile.boards.post.type.drawing")
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker(L.text("mobile.boards.compose.typeSwitcher"), selection: $contentType) {
                        ForEach(composeTypes, id: \.self) { type in
                            Text(label(for: type)).tag(type)
                        }
                    }
                    .pickerStyle(.segmented)

                    Text(L.text("mobile.boards.compose.drawingDisabledHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                Section {
                    TextField(L.text("mobile.boards.compose.titleLabel"), text: $title)
                    switch contentType {
                    case .text:
                        TextField(L.text("mobile.boards.compose.bodyLabel"), text: $bodyText, axis: .vertical)
                            .lineLimit(3 ... 8)
                    case .link, .video:
                        TextField(L.text("mobile.boards.compose.linkLabel"), text: $linkUrl)
                            .textInputAutocapitalization(.never)
                            .keyboardType(.URL)
                    case .image:
                        imagePickerSection
                        TextField(L.text("mobile.boards.compose.altLabel"), text: $altText, axis: .vertical)
                            .lineLimit(2 ... 4)
                        if altText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            Text(L.text("mobile.boards.compose.altHint"))
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }
                    case .file:
                        filePickerSection
                    case .audio:
                        audioSection
                    case .drawing:
                        EmptyView()
                    }
                }

                if let uploadProgress {
                    Section {
                        ProgressView(value: uploadProgress)
                        Text(L.text("mobile.boards.compose.uploading"))
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }

                if let validationHint {
                    Section {
                        Text(validationHint)
                            .font(.subheadline)
                            .foregroundStyle(.orange)
                    }
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage)
                            .font(.subheadline)
                            .foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle(L.text("mobile.boards.compose.navTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) {
                        stopRecordingIfNeeded()
                        dismiss()
                    }
                    .disabled(submitting)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.boards.compose.submit")) {
                        Task { await submit() }
                    }
                    .disabled(submitting)
                }
            }
            .onChange(of: photoItem) { _, item in
                Task { await loadPhoto(item) }
            }
            .sheet(isPresented: $showCamera) {
                BoardCameraCaptureView { image in
                    if let data = image.jpegData(compressionQuality: 0.88) {
                        pickedData = data
                        pickedFileName = "photo-\(Int(Date().timeIntervalSince1970)).jpg"
                        pickedMime = "image/jpeg"
                    }
                }
            }
            .sheet(isPresented: $showDocumentPicker) {
                BoardDocumentPicker { url in
                    loadDocument(url)
                }
            }
            .alert(L.text("mobile.boards.compose.micPermissionTitle"), isPresented: $showMicExplainer) {
                Button(L.text("mobile.boards.compose.continue")) {
                    Task { await beginRecording() }
                }
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
            } message: {
                Text(L.text("mobile.boards.compose.micPermissionMessage"))
            }
            .alert(L.text("mobile.boards.compose.cameraPermissionTitle"), isPresented: $showCameraExplainer) {
                Button(L.text("mobile.boards.compose.continue")) { showCamera = true }
                Button(L.text("mobile.common.cancel"), role: .cancel) {}
            } message: {
                Text(L.text("mobile.boards.compose.cameraPermissionMessage"))
            }
        }
    }

    @ViewBuilder
    private var imagePickerSection: some View {
        if let name = pickedFileName.nilIfEmpty {
            Text(name)
                .font(.subheadline)
        }
        PhotosPicker(selection: $photoItem, matching: .images) {
            Label(L.text("mobile.boards.compose.pickPhoto"), systemImage: "photo.on.rectangle")
        }
        Button {
            showCameraExplainer = true
        } label: {
            Label(L.text("mobile.boards.compose.takePhoto"), systemImage: "camera")
        }
    }

    @ViewBuilder
    private var filePickerSection: some View {
        if let name = pickedFileName.nilIfEmpty {
            Text(name)
                .font(.subheadline)
        }
        Button {
            showDocumentPicker = true
        } label: {
            Label(L.text("mobile.boards.compose.pickFile"), systemImage: "doc")
        }
    }

    @ViewBuilder
    private var audioSection: some View {
        if let name = pickedFileName.nilIfEmpty {
            Text(name)
                .font(.subheadline)
        }
        if isRecording {
            Button(role: .destructive) {
                stopRecording()
            } label: {
                Label(L.text("mobile.boards.compose.stopRecord"), systemImage: "stop.circle.fill")
            }
        } else {
            Button {
                showMicExplainer = true
            } label: {
                Label(L.text("mobile.boards.compose.recordAudio"), systemImage: "mic.circle.fill")
            }
        }
    }

    private func loadPhoto(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        do {
            if let data = try await item.loadTransferable(type: Data.self) {
                pickedData = data
                pickedFileName = "photo-\(Int(Date().timeIntervalSince1970)).jpg"
                pickedMime = "image/jpeg"
            }
        } catch {
            errorMessage = L.text("mobile.boards.compose.loadFileError")
        }
    }

    private func loadDocument(_ url: URL) {
        let scoped = url.startAccessingSecurityScopedResource()
        defer { if scoped { url.stopAccessingSecurityScopedResource() } }
        do {
            pickedData = try Data(contentsOf: url)
            pickedFileName = url.lastPathComponent
            pickedMime = UTType(filenameExtension: url.pathExtension)?.preferredMIMEType
                ?? "application/octet-stream"
        } catch {
            errorMessage = L.text("mobile.boards.compose.loadFileError")
        }
    }

    private func beginRecording() async {
        let session = AVAudioSession.sharedInstance()
        do {
            try session.setCategory(.playAndRecord, mode: .default, options: [.defaultToSpeaker])
            try session.setActive(true)
            let granted = await withCheckedContinuation { (cont: CheckedContinuation<Bool, Never>) in
                AVAudioSession.sharedInstance().requestRecordPermission { cont.resume(returning: $0) }
            }
            guard granted else {
                errorMessage = L.text("mobile.boards.compose.micDenied")
                return
            }
            let url = FileManager.default.temporaryDirectory
                .appendingPathComponent("board-audio-\(UUID().uuidString).m4a")
            let settings: [String: Any] = [
                AVFormatIDKey: Int(kAudioFormatMPEG4AAC),
                AVSampleRateKey: 44100,
                AVNumberOfChannelsKey: 1,
                AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue,
            ]
            let recorder = try AVAudioRecorder(url: url, settings: settings)
            recorder.record()
            self.recorder = recorder
            self.recordURL = url
            self.isRecording = true
            self.pickedFileName = url.lastPathComponent
            self.pickedMime = "audio/mp4"
        } catch {
            errorMessage = L.text("mobile.boards.compose.recordError")
        }
    }

    private func stopRecording() {
        recorder?.stop()
        isRecording = false
        if let url = recordURL, let data = try? Data(contentsOf: url) {
            pickedData = data
            pickedFileName = url.lastPathComponent
            pickedMime = "audio/mp4"
        }
        recorder = nil
    }

    private func stopRecordingIfNeeded() {
        if isRecording { stopRecording() }
    }

    private func submit() async {
        stopRecordingIfNeeded()
        validationHint = nil
        errorMessage = nil

        let hasFile = pickedData != nil
        let hasAudio = contentType == .audio && pickedData != nil
        let validation = BoardsLogic.validateCompose(
            contentType: contentType,
            text: bodyText,
            linkUrl: linkUrl,
            hasFile: hasFile,
            altText: altText,
            hasAudio: hasAudio
        )
        switch validation {
        case .ok:
            break
        case .missingText:
            validationHint = L.text("mobile.boards.compose.bodyRequired")
            return
        case .missingLink:
            validationHint = L.text("mobile.boards.compose.linkRequired")
            return
        case .missingFile:
            validationHint = L.text("mobile.boards.compose.fileRequired")
            return
        case .missingAltText:
            validationHint = L.text("mobile.boards.compose.altRequired")
            return
        case .missingAudio:
            validationHint = L.text("mobile.boards.compose.audioRequired")
            return
        }

        guard let token = session.accessToken else { return }
        submitting = true
        defer {
            submitting = false
            uploadProgress = nil
        }

        do {
            var attachmentId: String?
            var postType = contentType.rawValue
            var body: BoardPostBody?
            var link: String?
            let titleValue = title.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty

            switch contentType {
            case .text:
                body = BoardsLogic.makeTextBody(bodyText)
            case .link:
                link = linkUrl.trimmingCharacters(in: .whitespacesAndNewlines)
                if BoardsLogic.videoEmbedFromUrl(link ?? "") != nil {
                    postType = BoardContentType.video.rawValue
                }
            case .image, .file, .audio:
                guard let data = pickedData else { return }
                uploadProgress = 0.05
                let att = try await LMSAPI.uploadBoardAttachment(
                    courseCode: courseCode,
                    boardId: boardId,
                    fileName: pickedFileName.isEmpty ? "upload.bin" : pickedFileName,
                    mimeType: pickedMime,
                    fileData: data,
                    altText: contentType == .image ? altText.trimmingCharacters(in: .whitespacesAndNewlines) : nil,
                    contentType: contentType.rawValue,
                    accessToken: token,
                    onProgress: { uploadProgress = $0 }
                )
                attachmentId = att.id
            case .video, .drawing:
                break
            }

            let created = try await LMSAPI.createBoardPost(
                courseCode: courseCode,
                boardId: boardId,
                contentType: postType,
                title: titleValue,
                body: body,
                linkUrl: link,
                attachmentId: attachmentId,
                accessToken: token
            )
            onCreated(created)
            dismiss()
        } catch let error as APIError {
            if case let .httpStatus(_, message) = error {
                if message?.contains("Storage limit") == true {
                    errorMessage = L.text("mobile.boards.compose.quotaExceeded")
                } else if BoardsLogic.isFilterBlockMessage(message) {
                    errorMessage = L.text("mobile.boards.moderation.filterBlocked")
                } else if BoardsLogic.isLockOrFreezeMessage(message) {
                    errorMessage = L.text("mobile.boards.sync.lockedNotice")
                } else {
                    errorMessage = error.errorDescription ?? L.text("mobile.boards.compose.postError")
                }
            } else {
                errorMessage = error.errorDescription ?? L.text("mobile.boards.compose.postError")
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.boards.compose.postError")
        }
    }
}

private extension String {
    var nilIfEmpty: String? {
        let t = trimmingCharacters(in: .whitespacesAndNewlines)
        return t.isEmpty ? nil : t
    }
}

private struct BoardCameraCaptureView: UIViewControllerRepresentable {
    var onCapture: (UIImage) -> Void
    @Environment(\.dismiss) private var dismiss

    func makeUIViewController(context: Context) -> UIImagePickerController {
        let picker = UIImagePickerController()
        picker.sourceType = UIImagePickerController.isSourceTypeAvailable(.camera) ? .camera : .photoLibrary
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

private struct BoardDocumentPicker: UIViewControllerRepresentable {
    var onPick: (URL) -> Void
    @Environment(\.dismiss) private var dismiss

    func makeUIViewController(context: Context) -> UIDocumentPickerViewController {
        let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.item], asCopy: true)
        picker.delegate = context.coordinator
        return picker
    }

    func updateUIViewController(_ uiViewController: UIDocumentPickerViewController, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(onPick: onPick, dismiss: dismiss) }

    final class Coordinator: NSObject, UIDocumentPickerDelegate {
        let onPick: (URL) -> Void
        let dismiss: DismissAction

        init(onPick: @escaping (URL) -> Void, dismiss: DismissAction) {
            self.onPick = onPick
            self.dismiss = dismiss
        }

        func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
            if let url = urls.first { onPick(url) }
            dismiss()
        }

        func documentPickerWasCancelled(_ controller: UIDocumentPickerViewController) {
            dismiss()
        }
    }
}
