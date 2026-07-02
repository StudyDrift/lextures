import SwiftUI

/// Collaborative document viewer with embedded web editor and web fallback (M7.4).
struct CollabDocView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.openURL) private var openURL

    let course: CourseSummary
    let docId: String
    let title: String

    @State private var doc: CollabDoc?
    @State private var loading = true
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 0) {
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
                    .padding(16)
            }

            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let doc, doc.docType == .whiteboard {
                whiteboardFallback(doc)
            } else {
                AuthenticatedWebView(
                    urlString: GroupsLogic.collabDocWebPath(courseCode: course.courseCode, docId: docId),
                    accessToken: session.accessToken,
                    onError: { errorMessage = L.text("mobile.collabDocs.loadError") }
                )
            }

            HStack {
                if doc?.docType != .whiteboard {
                    Text(L.text("mobile.collabDocs.editOnWebHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer()
                Button(L.text("mobile.collabDocs.openOnWeb")) {
                    openURL(AppConfiguration.webURL(
                        path: GroupsLogic.collabDocWebPath(courseCode: course.courseCode, docId: docId)
                    ))
                }
                .font(.subheadline.weight(.semibold))
            }
            .padding(12)
            .background(LexturesTheme.sceneBackground(for: colorScheme))
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    @ViewBuilder
    private func whiteboardFallback(_ doc: CollabDoc) -> some View {
        LMSEmptyState(
            systemImage: "scribble.variable",
            title: L.text("mobile.collabDocs.whiteboardTitle"),
            message: L.format("mobile.collabDocs.whiteboardMessage", doc.title)
        )
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            doc = try await LMSAPI.fetchCollabDoc(
                courseCode: course.courseCode,
                docId: docId,
                accessToken: token
            )
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.collabDocs.loadError")
        }
    }
}