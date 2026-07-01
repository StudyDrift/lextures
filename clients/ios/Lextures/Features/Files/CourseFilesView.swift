import SwiftUI

/// Browse a course's file manager folders and files (M3.2).
struct CourseFilesView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var folderId: String?
    @State private var breadcrumbs: [CourseFileBreadcrumb] = []
    @State private var folders: [CourseFileFolder] = []
    @State private var files: [CourseFileItem] = []
    @State private var cacheLabel: String?
    @State private var errorMessage: String?
    @State private var loading = false
    @State private var previewTarget: FilePreviewTarget?
    @State private var showClearConfirm = false
    @State private var savedKeys: Set<String> = []
    @State private var filesSocket = CourseFilesSocket()

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            breadcrumbBar
            cacheBar

            if let errorMessage {
                LMSErrorBanner(message: errorMessage)
            }
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }

            if loading && folders.isEmpty && files.isEmpty {
                LMSSkeletonList(count: 4)
            } else if folders.isEmpty && files.isEmpty && errorMessage == nil {
                LMSEmptyState(
                    systemImage: "folder",
                    title: L.text("mobile.files.emptyFolder"),
                    message: L.text("mobile.files.emptyFolderHint")
                )
            } else {
                fileList
            }
        }
        .navigationDestination(item: $previewTarget) { target in
            FilePreviewView(target: target)
        }
        .confirmationDialog(
            L.text("mobile.files.clearCacheTitle"),
            isPresented: $showClearConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.files.clearCacheConfirm"), role: .destructive) {
                Task { await clearCourseDownloads() }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text(L.text("mobile.files.clearCacheMessage"))
        }
        .task(id: folderId) { await load() }
        .refreshable { await load() }
        .task {
            filesSocket.connect(courseCode: course.courseCode, accessToken: { session.accessToken })
        }
        .onDisappear { filesSocket.disconnect() }
        .onChange(of: filesSocket.revision) { _, _ in
            Task { await load() }
        }
    }

    @ViewBuilder
    private var breadcrumbBar: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 4) {
                breadcrumbChip(title: L.text("mobile.files.root"), isActive: folderId == nil) {
                    folderId = nil
                }
                ForEach(breadcrumbs) { crumb in
                    Image(systemName: "chevron.right")
                        .font(.caption2)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    breadcrumbChip(title: crumb.name, isActive: folderId == crumb.id) {
                        folderId = crumb.id
                    }
                }
            }
        }
    }

    private func breadcrumbChip(title: String, isActive: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(title)
                .font(.caption.weight(isActive ? .semibold : .regular))
                .foregroundStyle(isActive ? LexturesTheme.accent(for: colorScheme) : LexturesTheme.textSecondary(for: colorScheme))
                .lineLimit(1)
        }
        .disabled(isActive)
        .accessibilityLabel(title)
    }

    @ViewBuilder
    private var cacheBar: some View {
        HStack {
            Text(L.text("mobile.files.cacheSize"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Spacer()
            Text(ByteCountFormatter.string(fromByteCount: Int64(offline.storageBytes), countStyle: .file))
                .font(.caption.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            if offline.storageBytes > 0 {
                Button(L.text("mobile.files.clearCache")) { showClearConfirm = true }
                    .font(.caption.weight(.semibold))
            }
        }
    }

    private var fileList: some View {
        LazyVStack(spacing: 0) {
            ForEach(folders) { folder in
                Button {
                    folderId = folder.id
                } label: {
                    fileRow(
                        name: folder.name,
                        icon: "folder.fill",
                        subtitle: L.text("mobile.files.folder"),
                        saved: false
                    )
                }
                .buttonStyle(.plain)
                Divider().padding(.leading, 44)
            }
            ForEach(files) { file in
                Button {
                    previewTarget = FilePreviewTarget.from(file: file, courseCode: course.courseCode)
                } label: {
                    fileRow(
                        name: file.title,
                        icon: CourseFileLogic.listIcon(isFolder: false, fileName: file.title, mimeType: file.mimeType),
                        subtitle: rowSubtitle(for: file),
                        saved: savedKeys.contains(file.id)
                    )
                }
                .buttonStyle(.plain)
                Divider().padding(.leading, 44)
            }
        }
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .task(id: files.map(\.id).joined()) {
            await refreshSavedKeys()
        }
    }

    private func refreshSavedKeys() async {
        var keys: Set<String> = []
        for file in files {
            let target = FilePreviewTarget.from(file: file, courseCode: course.courseCode)
            if await FileDownloadManager.isDownloaded(target: target, offline: offline) {
                keys.insert(file.id)
            }
        }
        savedKeys = keys
    }

    private func fileRow(name: String, icon: String, subtitle: String, saved: Bool) -> some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .font(.body)
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 28)
            VStack(alignment: .leading, spacing: 2) {
                Text(name)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
            Spacer(minLength: 8)
            if saved {
                Label(L.text("mobile.files.saved"), systemImage: "arrow.down.circle.fill")
                    .font(.caption2)
                    .foregroundStyle(.green)
                    .labelStyle(.iconOnly)
                    .accessibilityLabel(L.text("mobile.files.saved"))
            }
            Image(systemName: "chevron.right")
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .contentShape(Rectangle())
    }

    private func rowSubtitle(for file: CourseFileItem) -> String {
        let size = CourseFileLogic.formatBytes(file.byteSize)
        if let date = LMSDates.parse(file.updatedAt) {
            return "\(size) · \(date.formatted(date: .abbreviated, time: .omitted))"
        }
        return size
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let cacheKey = OfflineCacheKey.courseFiles(courseCode: course.courseCode, folderId: folderId)
            let result = try await offline.cachedFetch(key: cacheKey, accessToken: token) {
                if let folderId {
                    return try await LMSAPI.fetchCourseFilesFolder(
                        courseCode: course.courseCode,
                        folderId: folderId,
                        accessToken: token
                    )
                }
                return try await LMSAPI.fetchCourseFilesRoot(courseCode: course.courseCode, accessToken: token)
            }
            folders = result.value.folders.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
            files = result.value.files.sorted { $0.title.localizedCaseInsensitiveCompare($1.title) == .orderedAscending }
            breadcrumbs = result.value.breadcrumbs ?? []
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? L.text("mobile.files.loadError")
        }
    }

    private func clearCourseDownloads() async {
        await offline.clearStorage()
    }
}
