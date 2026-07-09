import SwiftUI

struct ArchivedCoursesAdminRoute: Hashable {}

/// Global archived courses admin: list, restore, and permanent delete (M14.10).
struct ArchivedCoursesAdminView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var courses: [ArchivedCourseRow] = []
    @State private var searchText = ""
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var statusMessage: String?
    @State private var busyCode: String?
    @State private var pendingRestore: ArchivedCourseRow?
    @State private var deleteTarget: ArchivedCourseRow?
    @State private var deletePhrase = ""

    private var features: MobilePlatformFeatures { shell.platformFeatures }
    private var filteredRows: [ArchivedCourseRow] {
        ArchivedCoursesAdminLogic.filterRows(courses, query: searchText)
    }

    var body: some View {
        Group {
            if !ArchivedCoursesAdminLogic.canView(features: features, permissions: shell.permissions) {
                accessDenied
            } else {
                content
            }
        }
        .navigationTitle(L.text("mobile.admin.archivedCourses.title"))
        .navigationBarTitleDisplayMode(.inline)
        .searchable(
            text: $searchText,
            prompt: Text(L.text("mobile.admin.archivedCourses.search"))
        )
        .refreshable { await load() }
        .task { await load() }
        .confirmationDialog(
            L.text("mobile.admin.archivedCourses.restoreConfirm"),
            isPresented: Binding(
                get: { pendingRestore != nil },
                set: { if !$0 { pendingRestore = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.admin.archivedCourses.restore")) {
                if let row = pendingRestore {
                    Task { await restore(row) }
                }
            }
        }
        .sheet(item: $deleteTarget) { row in
            deleteSheet(for: row)
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.archivedCourses.accessDeniedTitle"),
            message: L.text("mobile.admin.archivedCourses.accessDeniedMessage")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.archivedCourses.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    if let statusMessage {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }

                    if loading && courses.isEmpty {
                        LMSSkeletonList(count: 3)
                    } else if courses.isEmpty {
                        LMSEmptyState(
                            systemImage: "archivebox",
                            title: L.text("mobile.admin.archivedCourses.emptyTitle"),
                            message: L.text("mobile.admin.archivedCourses.emptyMessage")
                        )
                    } else if filteredRows.isEmpty {
                        LMSEmptyState(
                            systemImage: "magnifyingglass",
                            title: L.text("mobile.admin.archivedCourses.emptyTitle"),
                            message: L.text("mobile.admin.archivedCourses.emptySearch")
                        )
                    } else {
                        ForEach(filteredRows) { row in
                            courseCard(row)
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    private func courseCard(_ row: ArchivedCourseRow) -> some View {
        let busy = busyCode == row.courseCode
        return LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(row.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    ? L.text("mobile.emDash")
                    : row.title)
                    .font(.headline)
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                Text("\(L.text("mobile.admin.archivedCourses.courseCode")): \(row.courseCode)")
                    .font(.caption.monospaced())
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Text("\(L.text("mobile.admin.archivedCourses.archivedBy")): \(ArchivedCoursesAdminLogic.archivedByLabel(row))")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                Text("\(L.text("mobile.admin.archivedCourses.archivedAt")): \(ArchivedCoursesAdminLogic.formatArchivedAt(row.archivedAt))")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                HStack(spacing: 12) {
                    Button {
                        pendingRestore = row
                    } label: {
                        Label(
                            busy && deleteTarget == nil
                                ? L.text("mobile.admin.archivedCourses.restoring")
                                : L.text("mobile.admin.archivedCourses.restore"),
                            systemImage: "arrow.uturn.backward"
                        )
                    }
                    .buttonStyle(.bordered)
                    .disabled(busy)

                    Button(role: .destructive) {
                        deletePhrase = ""
                        deleteTarget = row
                    } label: {
                        Label(L.text("mobile.admin.archivedCourses.delete"), systemImage: "trash")
                    }
                    .buttonStyle(.bordered)
                    .disabled(busy)
                }
                .padding(.top, 4)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func deleteSheet(for row: ArchivedCourseRow) -> some View {
        NavigationStack {
            Form {
                Section {
                    Text(L.text("mobile.admin.archivedCourses.deleteMessage"))
                    Text(row.title)
                        .font(.headline)
                    Text(row.courseCode)
                        .font(.caption.monospaced())
                }
                Section {
                    Text(L.text("mobile.admin.archivedCourses.deleteWarning"))
                        .font(.caption)
                        .foregroundStyle(.red)
                }
                Section(
                    L.format(
                        "mobile.admin.archivedCourses.deleteConfirmPhraseLabel",
                        ArchivedCoursesAdminLogic.deleteConfirmPhrase(for: row)
                    )
                ) {
                    TextField(
                        ArchivedCoursesAdminLogic.deleteConfirmPhrase(for: row),
                        text: $deletePhrase
                    )
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .font(.body.monospaced())
                }
            }
            .navigationTitle(L.text("mobile.admin.archivedCourses.deleteTitle"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.cancel")) {
                        deleteTarget = nil
                    }
                    .disabled(busyCode == row.courseCode)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L.text("mobile.admin.archivedCourses.deletePermanently"), role: .destructive) {
                        deleteTarget = nil
                        Task { await deletePermanently(row) }
                    }
                    .disabled(
                        busyCode == row.courseCode
                            || !ArchivedCoursesAdminLogic.deleteConfirmMatches(typed: deletePhrase, row: row)
                    )
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            courses = try await LMSAPI.fetchArchivedCourses(accessToken: token)
        } catch {
            errorMessage = ArchivedCoursesAdminLogic.userFacingError(error)
        }
    }

    private func restore(_ row: ArchivedCourseRow) async {
        guard let token = session.accessToken else { return }
        pendingRestore = nil
        busyCode = row.courseCode
        errorMessage = nil
        statusMessage = nil
        defer { busyCode = nil }

        do {
            try await LMSAPI.restoreArchivedCourse(courseCode: row.courseCode, accessToken: token)
            courses = ArchivedCoursesAdminLogic.rowsAfterRestore(courses, courseCode: row.courseCode)
            statusMessage = L.text("mobile.admin.archivedCourses.restoreSuccess")
        } catch {
            errorMessage = ArchivedCoursesAdminLogic.userFacingError(error)
        }
    }

    private func deletePermanently(_ row: ArchivedCourseRow) async {
        guard let token = session.accessToken else { return }
        busyCode = row.courseCode
        errorMessage = nil
        statusMessage = nil
        defer { busyCode = nil }

        do {
            try await LMSAPI.deleteArchivedCoursePermanently(courseCode: row.courseCode, accessToken: token)
            courses = ArchivedCoursesAdminLogic.rowsAfterDelete(courses, courseCode: row.courseCode)
            statusMessage = L.text("mobile.admin.archivedCourses.deleteSuccess")
        } catch {
            errorMessage = ArchivedCoursesAdminLogic.userFacingError(error)
        }
    }
}