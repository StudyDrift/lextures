import AVKit
import PDFKit
import SwiftUI
import UniformTypeIdentifiers

/// Reusable inline preview for course files, module file items, and submission attachments (M3.2).
struct FilePreviewView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let target: FilePreviewTarget

    @State private var loading = true
    @State private var errorMessage: String?
    @State private var previewData: Data?
    @State private var isSaved = false
    @State private var isDownloading = false
    @State private var shareItem: ShareableFile?

    private var previewKind: FilePreviewKind {
        CourseFileLogic.previewKind(mimeType: target.mimeType, fileName: target.displayName)
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                ProgressView(L.text("mobile.files.loading"))
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let errorMessage {
                LMSEmptyState(
                    systemImage: "exclamationmark.triangle",
                    title: target.displayName,
                    message: errorMessage
                )
                .padding(24)
            } else {
                previewContent
            }
        }
        .navigationTitle(target.displayName)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar { toolbarContent }
        .sheet(item: $shareItem) { item in
            ShareSheet(items: [item.url])
        }
        .task { await load() }
    }

    @ToolbarContentBuilder
    private var toolbarContent: some ToolbarContent {
        ToolbarItemGroup(placement: .topBarTrailing) {
            if isSaved {
                Label(L.text("mobile.files.saved"), systemImage: "arrow.down.circle.fill")
                    .foregroundStyle(.green)
                    .accessibilityLabel(L.text("mobile.files.saved"))
            }
            if isDownloading {
                ProgressView()
            } else {
                Button {
                    Task { await download() }
                } label: {
                    Label(L.text("mobile.files.download"), systemImage: "arrow.down.circle")
                }
                .accessibilityLabel(L.text("mobile.files.download"))
            }
            if previewKind == .downloadOnly || previewData != nil {
                Button {
                    Task { await openExternally() }
                } label: {
                    Label(L.text("mobile.files.openIn"), systemImage: "square.and.arrow.up")
                }
                .accessibilityLabel(L.text("mobile.files.openIn"))
            }
        }
    }

    @ViewBuilder
    private var previewContent: some View {
        switch previewKind {
        case .image:
            if let previewData, let uiImage = UIImage(data: previewData) {
                ZoomableImageView(image: uiImage)
            } else {
                unsupportedFallback
            }
        case .pdf:
            if let previewData {
                PDFKitView(data: previewData)
            } else {
                unsupportedFallback
            }
        case .audio, .video:
            if let token = session.accessToken {
                AuthedMediaPlayerView(
                    url: FileDownloadManager.contentURL(courseCode: target.courseCode, source: target.source),
                    accessToken: token
                )
            } else {
                unsupportedFallback
            }
        case .downloadOnly:
            LMSEmptyState(
                systemImage: CourseFileLogic.systemImage(for: previewKind),
                title: target.displayName,
                message: L.text("mobile.files.downloadOnlyHint")
            )
            .padding(24)
        }
    }

    private var unsupportedFallback: some View {
        LMSEmptyState(
            systemImage: "doc",
            title: target.displayName,
            message: L.text("mobile.files.previewUnavailable")
        )
        .padding(24)
    }

    private func load() async {
        loading = true
        errorMessage = nil
        defer { loading = false }

        isSaved = await FileDownloadManager.isDownloaded(target: target, offline: offline)

        if previewKind == .audio || previewKind == .video {
            loading = false
            return
        }

        if let cached = await FileDownloadManager.cachedData(target: target, offline: offline) {
            previewData = cached
            isSaved = true
            return
        }

        guard let token = session.accessToken else {
            errorMessage = L.text("mobile.files.loadError")
            return
        }

        if !NetworkMonitor.shared.isOnline {
            errorMessage = L.text("mobile.files.offlineUnavailable")
            return
        }

        do {
            previewData = try await FileDownloadManager.fetchData(
                courseCode: target.courseCode,
                target: target,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.files.loadError")
        }
    }

    private func download() async {
        guard let token = session.accessToken else { return }
        isDownloading = true
        defer { isDownloading = false }
        do {
            try await FileDownloadManager.download(target: target, accessToken: token, offline: offline)
            isSaved = true
            if previewData == nil, previewKind != .audio, previewKind != .video {
                previewData = await FileDownloadManager.cachedData(target: target, offline: offline)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.files.downloadError")
        }
    }

    private func openExternally() async {
        if let cached = await FileDownloadManager.cachedData(target: target, offline: offline),
           let url = writeTempFile(data: cached) {
            shareItem = ShareableFile(url: url)
            return
        }
        guard let token = session.accessToken else { return }
        do {
            let data = try await FileDownloadManager.fetchData(
                courseCode: target.courseCode,
                target: target,
                accessToken: token
            )
            if let url = writeTempFile(data: data) {
                shareItem = ShareableFile(url: url)
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.files.openError")
        }
    }

    private func writeTempFile(data: Data) -> URL? {
        let ext = (target.displayName as NSString).pathExtension
        let name = ext.isEmpty ? target.displayName : target.displayName
        let url = FileManager.default.temporaryDirectory.appendingPathComponent(name)
        try? data.write(to: url, options: .atomic)
        return url
    }
}

// MARK: - Subviews

private struct ZoomableImageView: View {
    let image: UIImage
    @State private var scale: CGFloat = 1

    var body: some View {
        ScrollView([.horizontal, .vertical]) {
            Image(uiImage: image)
                .resizable()
                .scaledToFit()
                .scaleEffect(scale)
                .padding()
                .accessibilityLabel("Image preview")
        }
        .gesture(
            MagnificationGesture()
                .onChanged { value in scale = max(1, value) }
        )
    }
}

private struct PDFKitView: UIViewRepresentable {
    let data: Data

    func makeUIView(context: Context) -> PDFView {
        let view = PDFView()
        view.autoScales = true
        view.displayMode = .singlePageContinuous
        view.displayDirection = .vertical
        view.document = PDFDocument(data: data)
        view.accessibilityLabel = "PDF preview"
        return view
    }

    func updateUIView(_ uiView: PDFView, context: Context) {
        if uiView.document == nil {
            uiView.document = PDFDocument(data: data)
        }
    }
}

private struct AuthedMediaPlayerView: View {
    let url: URL
    let accessToken: String

    var body: some View {
        VideoPlayer(player: AVPlayer(url: url, headers: ["Authorization": "Bearer \(accessToken)"]))
            .accessibilityLabel("Media player")
    }
}

private extension AVPlayer {
    convenience init(url: URL, headers: [String: String]) {
        let asset = AVURLAsset(url: url, options: ["AVURLAssetHTTPHeaderFieldsKey": headers])
        self.init(playerItem: AVPlayerItem(asset: asset))
    }
}

private struct ShareableFile: Identifiable {
    let id = UUID()
    let url: URL
}

private struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
