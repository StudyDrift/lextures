import SwiftUI

@MainActor
@Observable
final class ReadingModel {
    var entries: [ReadingLogEntry] = []
    var loginStreakDays = 0
    var bookClubCourses: [CourseSummary] = []
    var orgId: String?
    var errorMessage: String?
    var loading = false
    var saving = false
    var showLogSheet = false
    var logDraft = LogReadingDraft()
    var logError: String?

    func load(accessToken: String?, offline: OfflineService) async {
        guard let accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }

        do {
            async let entriesTask = offline.cachedFetch(
                key: OfflineCacheKey.readingLog(),
                accessToken: accessToken
            ) { try await LMSAPI.fetchReadingLogEntries(accessToken: accessToken) }
            async let coursesTask = LMSAPI.fetchCourses(accessToken: accessToken)
            async let statsTask = LMSAPI.fetchStudyStats(accessToken: accessToken)

            entries = try await entriesTask.value
            let courses = try await coursesTask
            orgId = ReadingLogic.resolveOrgId(from: courses)
            bookClubCourses = ReadingLogic.bookClubCourses(from: courses)
            if let stats = try? await statsTask {
                loginStreakDays = stats.loginStreakDays
            }
        } catch {
            errorMessage = L.text("mobile.reading.error.load")
        }
    }

    func saveEntry(accessToken: String?, offline: OfflineService, draft: LogReadingDraft) async {
        guard let accessToken else { return }
        guard ReadingLogic.logEntryValid(
            bookTitle: draft.bookTitle,
            bookId: draft.bookId,
            logDate: draft.logDate
        ) else {
            logError = L.text("mobile.reading.error.validation")
            return
        }

        saving = true
        logError = nil
        defer { saving = false }

        let title = draft.bookTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        let reflection = draft.reflection.trimmingCharacters(in: .whitespacesAndNewlines)
        let pages = Int(draft.pagesRead.trimmingCharacters(in: .whitespacesAndNewlines))
        let body = PostReadingLogBody(
            bookId: draft.bookId,
            bookTitle: title.isEmpty ? nil : title,
            logDate: draft.logDate,
            pagesRead: pages,
            reflection: reflection.isEmpty ? nil : reflection
        )

        do {
            if NetworkMonitor.shared.isOnline {
                _ = try await LMSAPI.createReadingLogEntry(body: body, accessToken: accessToken)
            } else {
                _ = try await offline.enqueueMutation(
                    method: "POST",
                    path: "/api/v1/me/reading-log",
                    body: body,
                    label: L.text("mobile.reading.logSave"),
                    accessToken: accessToken
                )
            }
            showLogSheet = false
            logDraft = LogReadingDraft()
            await load(accessToken: accessToken, offline: offline)
        } catch {
            logError = L.text("mobile.reading.error.save")
        }
    }

    func beginLog(for book: LibraryBook? = nil) {
        if let book {
            logDraft = LogReadingDraft(
                bookId: book.id,
                bookTitle: book.title,
                logDate: ReadingLogic.todayISO()
            )
        } else {
            logDraft = LogReadingDraft(logDate: ReadingLogic.todayISO())
        }
        logError = nil
        showLogSheet = true
    }
}

struct ReadingDashboardView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let onOpenBookClub: (CourseSummary) -> Void

    @State private var model = ReadingModel()

    private var weeklyPages: Int { ReadingLogic.weeklyPages(from: model.entries) }
    private var readingStreak: Int { ReadingLogic.readingStreakDays(from: model.entries) }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if !NetworkMonitor.shared.isOnline {
                        OfflineBanner()
                    }
                    if let errorMessage = model.errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }

                    statsRow

                    Button {
                        model.beginLog()
                    } label: {
                        Label(L.text("mobile.reading.logAction"), systemImage: "plus.circle.fill")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .accessibilityLabel(L.text("mobile.reading.logAction"))

                    if model.orgId != nil {
                        NavigationLink {
                            if let orgId = model.orgId {
                                LeveledLibraryView(orgId: orgId) { book in
                                    model.beginLog(for: book)
                                }
                            }
                        } label: {
                            LMSCard {
                                HStack {
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text(L.text("mobile.reading.libraryTitle"))
                                            .font(.subheadline.weight(.semibold))
                                        Text(L.text("mobile.reading.libraryHint"))
                                            .font(.caption)
                                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    }
                                    Spacer()
                                    Image(systemName: "chevron.right")
                                        .font(.caption)
                                }
                            }
                        }
                        .buttonStyle(.plain)
                    }

                    bookClubSection
                    historySection
                }
                .padding(16)
            }
        }
        .navigationTitle(L.text("mobile.reading.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await model.load(accessToken: session.accessToken, offline: offline) }
        .refreshable { await model.load(accessToken: session.accessToken, offline: offline) }
        .sheet(isPresented: $model.showLogSheet) {
            LogReadingSheet(
                initial: model.logDraft,
                saving: model.saving,
                errorMessage: model.logError
            ) { draft in
                Task {
                    await model.saveEntry(accessToken: session.accessToken, offline: offline, draft: draft)
                }
            }
        }
    }

    @ViewBuilder
    private var statsRow: some View {
        HStack(spacing: 12) {
            statCard(title: L.text("mobile.reading.weeklyPages"), value: "\(weeklyPages)")
            statCard(
                title: L.text("mobile.reading.readingStreak"),
                value: L.plural("mobile.reading.streakDays", count: readingStreak)
            )
            if model.loginStreakDays > 0 {
                statCard(
                    title: L.text("mobile.reading.loginStreak"),
                    value: L.plural("mobile.reading.streakDays", count: model.loginStreakDays)
                )
            }
        }
    }

    @ViewBuilder
    private func statCard(title: String, value: String) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(value)
                    .font(.title2.weight(.bold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    @ViewBuilder
    private var bookClubSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.reading.bookClubTitle"))
                .font(.headline)
            Text(L.text("mobile.reading.bookClubHint"))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

            if model.bookClubCourses.isEmpty {
                Text(L.text("mobile.reading.bookClubEmpty"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                ForEach(model.bookClubCourses) { course in
                    Button {
                        onOpenBookClub(course)
                    } label: {
                        LMSCard {
                            HStack {
                                Text(course.displayTitle)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                Spacer()
                                Image(systemName: "person.3")
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    @ViewBuilder
    private var historySection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(L.text("mobile.reading.historyTitle"))
                .font(.headline)

            if model.loading && model.entries.isEmpty {
                LMSSkeletonList(count: 3)
            } else if model.entries.isEmpty {
                LMSEmptyState(
                    systemImage: "book",
                    title: L.text("mobile.reading.historyEmptyTitle"),
                    message: L.text("mobile.reading.historyEmptyMessage")
                )
            } else {
                ForEach(model.entries) { entry in
                    LMSCard {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(entry.bookTitle ?? L.text("mobile.reading.unknownBook"))
                                .font(.subheadline.weight(.semibold))
                            HStack {
                                Text(entry.logDate)
                                if let pages = entry.pagesRead {
                                    Text(L.format("mobile.reading.pagesCount", pages))
                                }
                            }
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            if let reflection = entry.reflection, !reflection.isEmpty {
                                Text(reflection)
                                    .font(.caption)
                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                    .lineLimit(2)
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
            }
        }
    }
}