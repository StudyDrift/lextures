import AVKit
import SwiftUI
import WebKit

/// Type-specific board post card (VC.M2 / VC.M3 / VC.M5).
struct BoardPostCard: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL
    @Environment(\.boardEngagement) private var engagement

    let post: BoardPost
    var canEdit: Bool = false
    var canArrange: Bool = false
    var canManageBoard: Bool = false
    var currentUserId: String?
    var reactionMode: BoardReactionMode = .none
    var canInteract: Bool = true
    var assignmentLinked: Bool = false
    var sections: [BoardSection] = []
    var siblings: [BoardPost] = []
    var showTimelineArrange: Bool = false
    var showMapArrange: Bool = false
    var onEdit: (() -> Void)?
    var onDelete: (() -> Void)?
    var onArrange: ((ArrangeBoardPostInput) -> Void)?

    @State private var showFullImage = false
    @State private var showDeleteConfirm = false
    @State private var showComments = false
    @State private var showGradeSheet = false
    @State private var showReport = false

    private var knownType: BoardContentType? {
        BoardContentType(rawValue: post.contentType.lowercased())
    }

    private var canGrade: Bool {
        canManageBoard && reactionMode == .grade
    }

    private var safetyState: BoardPostSafetyState {
        BoardsLogic.postSafetyState(post)
    }

    var body: some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                if safetyState == .removed {
                    Text(L.text("mobile.boards.moderation.removedPlaceholder"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .accessibilityLabel(L.text("mobile.boards.moderation.removedPlaceholder"))
                } else {
                    header
                    if safetyState == .pendingApproval {
                        Text(L.text("mobile.boards.moderation.pendingBadge"))
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.orange)
                            .accessibilityLabel(L.text("mobile.boards.moderation.pendingBadge"))
                    }
                    content
                    engagementFooter
                }
            }
        }
        .accessibilityElement(children: .contain)
        .fullScreenCover(isPresented: $showFullImage) {
            if let url = BoardsLogic.attachmentMediaURL(post.attachment) {
                BoardImageViewer(
                    url: url,
                    altText: post.attachment?.altText ?? post.title,
                    onClose: { showFullImage = false }
                )
            }
        }
        .confirmationDialog(
            L.text("mobile.boards.post.deleteConfirm"),
            isPresented: $showDeleteConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.boards.post.delete"), role: .destructive) {
                onDelete?()
            }
            Button(L.text("mobile.common.cancel"), role: .cancel) {}
        }
        .sheet(isPresented: $showComments) {
            if let engagement {
                CommentSheet(
                    courseCode: engagement.courseCode,
                    boardId: post.boardId,
                    postId: post.id,
                    canInteract: canInteract,
                    canManageBoard: canManageBoard,
                    currentUserId: currentUserId,
                    onCountChange: { delta in
                        var next = post
                        next.commentCount = max(0, (post.commentCount ?? 0) + delta)
                        engagement.onPostUpdate(next)
                    }
                )
                .presentationDetents([.medium, .large])
            }
        }
        .sheet(isPresented: $showGradeSheet) {
            if let engagement {
                GradeSheet(
                    courseCode: engagement.courseCode,
                    boardId: post.boardId,
                    post: post,
                    assignmentLinked: assignmentLinked,
                    onPostUpdate: engagement.onPostUpdate,
                    onAnnounce: engagement.onAnnounce
                )
                .presentationDetents([.medium])
            }
        }
        .sheet(isPresented: $showReport) {
            if let engagement {
                ReportDialog(
                    courseCode: engagement.courseCode,
                    boardId: post.boardId,
                    postId: post.id
                )
                .presentationDetents([.medium])
            }
        }
    }

    @ViewBuilder
    private var engagementFooter: some View {
        if let engagement {
            HStack(spacing: 8) {
                if reactionMode != .none {
                    ReactionControl(
                        courseCode: engagement.courseCode,
                        boardId: post.boardId,
                        post: post,
                        reactionMode: reactionMode,
                        canInteract: canInteract,
                        canGrade: canGrade,
                        assignmentLinked: assignmentLinked,
                        onPostUpdate: engagement.onPostUpdate,
                        onAnnounce: engagement.onAnnounce,
                        onOpenGradeSheet: { showGradeSheet = true }
                    )
                }
                Spacer(minLength: 0)
                Button {
                    showComments = true
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "bubble.left")
                        let n = post.commentCount ?? 0
                        if n > 0 {
                            Text("\(n)")
                                .font(.caption.weight(.medium))
                                .monospacedDigit()
                        } else {
                            Text(L.text("mobile.boards.comment.toggle"))
                                .font(.caption.weight(.medium))
                        }
                    }
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .frame(minHeight: 36)
                    .padding(.horizontal, 8)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(L.text("mobile.boards.comment.toggle"))
            }
        }
    }

    private var header: some View {
        HStack(alignment: .top, spacing: 8) {
            VStack(alignment: .leading, spacing: 2) {
                if !post.title.isEmpty {
                    Text(post.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                }
                if let author = BoardsLogic.attributionLabel(for: post) {
                    Text(author)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Text(typeLabel)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 0)
            if canArrange, let onArrange {
                CardArrangeMenu(
                    post: post,
                    sections: sections,
                    siblings: siblings.isEmpty ? [post] : siblings,
                    showTimeline: showTimelineArrange,
                    showMap: showMapArrange,
                    onMoveToSection: { onArrange(ArrangeBoardPostInput(sectionId: $0)) },
                    onReorder: { onArrange(ArrangeBoardPostInput(sortIndex: $0)) },
                    onSetEventDate: showTimelineArrange
                        ? { onArrange(ArrangeBoardPostInput(eventDate: $0 ?? "")) }
                        : nil,
                    onSetCoords: showMapArrange
                        ? { lat, lng in onArrange(ArrangeBoardPostInput(lat: lat, lng: lng)) }
                        : nil
                )
            }
            if canEdit || canManageBoard || engagement != nil {
                Menu {
                    if canEdit, knownType == .text || knownType == .link {
                        Button(L.text("mobile.boards.post.edit")) { onEdit?() }
                    }
                    if engagement != nil {
                        Button(L.text("mobile.boards.report.action")) { showReport = true }
                    }
                    if canManageBoard {
                        if let onHide = engagement?.onHidePost {
                            Button(L.text("mobile.boards.moderation.hide")) { onHide(post) }
                        }
                        if let onRemove = engagement?.onRemovePost {
                            Button(L.text("mobile.boards.moderation.remove"), role: .destructive) {
                                onRemove(post)
                            }
                        }
                    }
                    if canEdit {
                        Button(L.text("mobile.boards.post.delete"), role: .destructive) {
                            showDeleteConfirm = true
                        }
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .accessibilityLabel(L.text("mobile.boards.post.actions"))
            }
        }
    }

    private var typeLabel: String {
        switch post.contentType.lowercased() {
        case "text": return L.text("mobile.boards.post.type.text")
        case "image": return L.text("mobile.boards.post.type.image")
        case "file": return L.text("mobile.boards.post.type.file")
        case "link": return L.text("mobile.boards.post.type.link")
        case "video": return L.text("mobile.boards.post.type.video")
        case "audio": return L.text("mobile.boards.post.type.audio")
        case "drawing": return L.text("mobile.boards.post.type.drawing")
        default: return L.text("mobile.boards.post.type.unsupported")
        }
    }

    @ViewBuilder
    private var content: some View {
        switch knownType {
        case .text:
            let plain = BoardsLogic.bodyPlainText(post)
            if !plain.isEmpty {
                MarkdownTextView(markdown: plain)
            }
        case .image:
            mediaAttachment(kind: .image)
        case .file:
            mediaAttachment(kind: .file)
        case .audio:
            mediaAttachment(kind: .audio)
        case .video:
            if let link = post.linkUrl, let embed = BoardsLogic.videoEmbedFromUrl(link),
               let embedURL = BoardsLogic.embedURL(for: embed) {
                BoardEmbedWebView(url: embedURL)
                    .frame(height: 200)
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .accessibilityLabel(L.text("mobile.boards.post.videoEmbed"))
            } else {
                mediaAttachment(kind: .video)
                if let link = post.linkUrl {
                    linkPreviewOrPlain(link)
                }
            }
        case .link:
            if let link = post.linkUrl {
                if let embed = BoardsLogic.videoEmbedFromUrl(link),
                   let embedURL = BoardsLogic.embedURL(for: embed) {
                    BoardEmbedWebView(url: embedURL)
                        .frame(height: 200)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                } else {
                    linkPreviewOrPlain(link)
                }
            }
        case .drawing:
            drawingView
        case .none:
            Text(L.text("mobile.boards.post.unsupportedMessage"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
    }

    private enum MediaKind { case image, file, audio, video }

    @ViewBuilder
    private func mediaAttachment(kind: MediaKind) -> some View {
        if let att = post.attachment {
            let scan = att.scanStatus.lowercased()
            if scan == "pending" {
                HStack(spacing: 8) {
                    ProgressView()
                    Text(L.text("mobile.boards.post.scanning"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .accessibilityElement(children: .combine)
            } else if scan == "blocked" {
                Label(L.text("mobile.boards.post.blocked"), systemImage: "exclamationmark.triangle.fill")
                    .font(.subheadline)
                    .foregroundStyle(.orange)
            } else if let url = BoardsLogic.attachmentMediaURL(att) {
                switch kind {
                case .image:
                    Button {
                        showFullImage = true
                    } label: {
                        AsyncImage(url: url) { phase in
                            switch phase {
                            case .success(let image):
                                image
                                    .resizable()
                                    .scaledToFit()
                                    .frame(maxHeight: 240)
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            case .failure:
                                Image(systemName: "photo")
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            default:
                                ProgressView()
                            }
                        }
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(att.altText.isEmpty ? L.text("mobile.boards.post.imageAltFallback") : att.altText)
                case .audio:
                    BoardAVPlayer(url: url, video: false)
                case .video:
                    BoardAVPlayer(url: url, video: true)
                        .frame(height: 200)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                case .file:
                    Button {
                        openURL(url)
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "doc.fill")
                            VStack(alignment: .leading, spacing: 2) {
                                Text(att.fileName.isEmpty ? L.text("mobile.boards.post.type.file") : att.fileName)
                                    .font(.subheadline.weight(.medium))
                                if att.sizeBytes > 0 {
                                    Text(BoardsLogic.formatFileSize(att.sizeBytes))
                                        .font(.caption)
                                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                }
                            }
                            Spacer()
                            Image(systemName: "arrow.up.right")
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    @ViewBuilder
    private func linkPreviewOrPlain(_ link: String) -> some View {
        if let preview = post.linkPreview, preview.title != nil || preview.description != nil {
            Button {
                if let url = BoardsLogic.absoluteURL(link) { openURL(url) }
            } label: {
                HStack(alignment: .top, spacing: 10) {
                    if let image = preview.image, let imageURL = BoardsLogic.absoluteURL(image) {
                        AsyncImage(url: imageURL) { phase in
                            if case .success(let img) = phase {
                                img.resizable().scaledToFill()
                            } else {
                                Color.gray.opacity(0.15)
                            }
                        }
                        .frame(width: 56, height: 56)
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                    } else {
                        Image(systemName: "link")
                            .frame(width: 56, height: 56)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    VStack(alignment: .leading, spacing: 4) {
                        Text(preview.title ?? link)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            .multilineTextAlignment(.leading)
                        if let desc = preview.description, !desc.isEmpty {
                            Text(desc)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .lineLimit(2)
                        }
                    }
                    Spacer(minLength: 0)
                }
            }
            .buttonStyle(.plain)
        } else if let url = BoardsLogic.absoluteURL(link) {
            Link(link, destination: url)
                .font(.subheadline)
        }
    }

    @ViewBuilder
    private var drawingView: some View {
        let elements = BoardsLogic.parseDrawingElements(post.drawingData)
        if elements.isEmpty {
            Text(L.text("mobile.boards.post.drawingEmpty"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            Canvas { context, size in
                WhiteboardRenderer.draw(
                    elements: elements,
                    in: context,
                    size: size,
                    isDark: colorScheme == .dark
                )
            }
            .frame(height: 160)
            .clipShape(RoundedRectangle(cornerRadius: 8))
            .accessibilityLabel(L.text("mobile.boards.post.type.drawing"))
        }
    }
}

private struct BoardAVPlayer: View {
    let url: URL
    let video: Bool

    var body: some View {
        VideoPlayer(player: AVPlayer(url: url))
            .frame(maxHeight: video ? 240 : 56)
            .accessibilityLabel(video ? L.text("mobile.boards.post.type.video") : L.text("mobile.boards.post.type.audio"))
    }
}

private struct BoardEmbedWebView: UIViewRepresentable {
    let url: URL

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.allowsInlineMediaPlayback = true
        let view = WKWebView(frame: .zero, configuration: config)
        view.scrollView.isScrollEnabled = false
        view.load(URLRequest(url: url))
        return view
    }

    func updateUIView(_ uiView: WKWebView, context: Context) {}
}

private struct BoardImageViewer: View {
    let url: URL
    let altText: String
    var onClose: () -> Void

    var body: some View {
        NavigationStack {
            ZStack {
                Color.black.ignoresSafeArea()
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFit()
                            .accessibilityLabel(altText)
                    default:
                        ProgressView().tint(.white)
                    }
                }
            }
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { onClose() }
                        .foregroundStyle(.white)
                }
            }
        }
    }
}
