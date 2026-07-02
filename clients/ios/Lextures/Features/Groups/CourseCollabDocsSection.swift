import SwiftUI

/// Course workspace entry for collaborative documents (M7.4).
struct CourseCollabDocsSection: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var docs: [CollabDoc] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = true
    @State private var openDoc: CollabDocRoute?

    private var courseDocs: [CollabDoc] {
        GroupsLogic.courseCollabDocs(docs)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            if !NetworkMonitor.shared.isOnline {
                OfflineBanner()
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }
            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }

            if loading && courseDocs.isEmpty {
                LMSSkeletonList(count: 2)
            } else if courseDocs.isEmpty {
                LMSEmptyState(
                    systemImage: "doc.text",
                    title: L.text("mobile.collabDocs.emptyTitle"),
                    message: L.text("mobile.collabDocs.emptyMessage")
                )
            } else {
                ForEach(courseDocs) { doc in
                    Button {
                        openDoc = CollabDocRoute(docId: doc.id, title: doc.title)
                    } label: {
                        docRow(doc)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .navigationDestination(item: $openDoc) { route in
            CollabDocView(course: course, docId: route.docId, title: route.title)
        }
        .task { await load() }
        .refreshable { await load(force: true) }
    }

    @ViewBuilder
    private func docRow(_ doc: CollabDoc) -> some View {
        LMSCard {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(doc.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(doc.docType == .whiteboard
                        ? L.text("mobile.collabDocs.typeWhiteboard")
                        : L.text("mobile.collabDocs.typeRichText"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !docs.isEmpty { loading = false }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.collabDocs(course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCollabDocs(courseCode: course.courseCode, accessToken: token)
            }
            docs = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.collabDocs.loadError")
        }
    }
}