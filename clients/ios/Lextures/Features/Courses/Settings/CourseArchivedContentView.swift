import SwiftUI

/// Archived module items list with restore (M13.12).
struct CourseArchivedContentView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let course: CourseSummary

    @State private var permissions: [String] = []
    @State private var permissionsLoaded = false
    @State private var structureItems: [CourseStructureItem] = []
    @State private var loading = true
    @State private var loadError: String?
    @State private var cacheLabel: String?
    @State private var actionError: String?
    @State private var actionSuccess: String?
    @State private var restoringId: String?
    @State private var pendingRestore: CourseArchivedContentLogic.ArchivedContentRow?

    private var isOnline: Bool { NetworkMonitor.shared.isOnline }

    private var canView: Bool {
        CourseArchivedContentLogic.canViewArchivedContent(
            courseCode: course.courseCode,
            permissions: permissions
        )
    }

    private var rows: [CourseArchivedContentLogic.ArchivedContentRow] {
        CourseArchivedContentLogic.archivedRows(from: structureItems)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                if !permissionsLoaded || loading {
                    ProgressView(L.text("mobile.courseSettings.loading"))
                } else if !canView {
                    accessDenied
                } else {
                    if !isOnline {
                        OfflineBanner()
                    }
                    if let cacheLabel {
                        StalenessChip(label: cacheLabel)
                    }
                    if let loadError {
                        LMSErrorBanner(message: loadError)
                    }
                    if let actionError {
                        LMSErrorBanner(message: actionError)
                    }
                    if let actionSuccess {
                        LMSCard(accent: LexturesTheme.brandTeal) {
                            Label(actionSuccess, systemImage: "checkmark.circle.fill")
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.primary)
                        }
                    }

                    Text(L.text("mobile.courseSettings.archivedContent.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if rows.isEmpty {
                        LMSEmptyState(
                            systemImage: "archivebox",
                            title: L.text("mobile.courseSettings.archivedContent.emptyTitle"),
                            message: L.text("mobile.courseSettings.archivedContent.emptyMessage")
                        )
                    } else {
                        ForEach(rows) { row in
                            archivedRowCard(row)
                        }
                    }
                }
            }
            .padding(16)
        }
        .task(id: course.courseCode) {
            await loadPermissions()
            await reload()
        }
        .confirmationDialog(
            L.text("mobile.courseSettings.archivedContent.restoreConfirmTitle"),
            isPresented: Binding(
                get: { pendingRestore != nil },
                set: { if !$0 { pendingRestore = nil } }
            ),
            titleVisibility: .visible
        ) {
            if let row = pendingRestore {
                Button(L.text("mobile.courseSettings.archivedContent.restore")) {
                    Task { await restore(row) }
                }
                Button(L.text("mobile.courseSettings.archivedContent.cancel"), role: .cancel) {
                    pendingRestore = nil
                }
            }
        } message: {
            if let row = pendingRestore {
                Text(L.format("mobile.courseSettings.archivedContent.restoreConfirmMessage", row.title))
            }
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.courseSettings.accessDeniedTitle"),
            message: L.text("mobile.courseSettings.archivedContent.accessDeniedMessage")
        )
    }

    private func archivedRowCard(_ row: CourseArchivedContentLogic.ArchivedContentRow) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 10) {
                Text(row.title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                LabeledContent(L.text("mobile.courseSettings.archivedContent.type")) {
                    Text(L.text(String.LocalizationValue(row.kindLabelKey)))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .font(.caption)

                LabeledContent(L.text("mobile.courseSettings.archivedContent.module")) {
                    Text(row.moduleTitle)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .font(.caption)

                LabeledContent(L.text("mobile.courseSettings.archivedContent.archivedAt")) {
                    Text(CourseArchivedContentLogic.formatArchivedAt(row.archivedAt))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
                .font(.caption)

                Button {
                    pendingRestore = row
                } label: {
                    Label(
                        restoringId == row.id
                            ? L.text("mobile.courseSettings.archivedContent.restoring")
                            : L.text("mobile.courseSettings.archivedContent.restore"),
                        systemImage: "arrow.uturn.backward"
                    )
                }
                .buttonStyle(.bordered)
                .disabled(restoringId != nil)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func loadPermissions() async {
        defer { permissionsLoaded = true }
        guard let token = session.accessToken else { return }
        permissions = (try? await LMSAPI.fetchMyPermissions(accessToken: token)) ?? []
    }

    private func reload() async {
        guard let token = session.accessToken else { return }
        loading = true
        loadError = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: CourseArchivedContentLogic.cacheKeyArchivedStructure(courseCode: course.courseCode),
                accessToken: token
            ) {
                try await LMSAPI.fetchCourseArchivedStructure(
                    courseCode: course.courseCode,
                    accessToken: token
                )
            }
            structureItems = result.value
            if let cached = result.cached, cached.isStale(isOnline: isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadError = CourseArchivedContentLogic.userFacingError(error)
        }
    }

    private func restore(_ row: CourseArchivedContentLogic.ArchivedContentRow) async {
        guard let token = session.accessToken else { return }
        pendingRestore = nil
        restoringId = row.id
        actionError = nil
        actionSuccess = nil
        defer { restoringId = nil }

        do {
            _ = try await offline.enqueueMutation(
                method: "PATCH",
                path: "/api/v1/courses/\(LMSAPI.encodePath(course.courseCode))/structure/items/\(LMSAPI.encodePath(row.id))",
                body: CourseStructureItemPatch(archived: false),
                label: L.text("mobile.courseSettings.archivedContent.restoreLabel"),
                accessToken: token,
                idempotencyKey: "course-archived-restore:\(course.courseCode):\(row.id)"
            )
            structureItems = CourseArchivedContentLogic.itemsAfterRestore(
                items: structureItems,
                removedId: row.id
            )
            if isOnline {
                actionSuccess = L.text("mobile.courseSettings.archivedContent.restoreSuccess")
            } else {
                actionSuccess = L.text("mobile.courseSettings.archivedContent.restoreQueued")
            }
        } catch {
            actionError = CourseArchivedContentLogic.userFacingError(error)
        }
    }
}
