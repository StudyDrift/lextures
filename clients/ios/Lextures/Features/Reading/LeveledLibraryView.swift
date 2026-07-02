import SwiftUI

struct LeveledLibraryView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let orgId: String
    let onLogBook: (LibraryBook) -> Void

    @State private var gradeBand = ""
    @State private var books: [LibraryBook] = []
    @State private var loading = false
    @State private var errorMessage: String?
    @State private var selectedBook: LibraryBook?
    @State private var previewURL: URL?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            VStack(alignment: .leading, spacing: 12) {
                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                }

                LMSSegmentedChips(
                    options: ReadingLogic.gradeBands,
                    selection: $gradeBand,
                    label: { band in
                        band.isEmpty ? L.text("mobile.reading.allLevels") : band
                    }
                )
                .onChange(of: gradeBand) { _, _ in Task { await load(force: true) } }

                if loading && books.isEmpty {
                    LMSSkeletonList(count: 4)
                } else if books.isEmpty {
                    LMSEmptyState(
                        systemImage: "books.vertical",
                        title: L.text("mobile.reading.libraryEmptyTitle"),
                        message: L.text("mobile.reading.libraryEmptyMessage")
                    )
                } else {
                    ScrollView {
                        LazyVStack(spacing: 10) {
                            ForEach(books) { book in
                                Button { selectedBook = book } label: {
                                    bookRow(book)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }
            .padding(16)
        }
        .navigationTitle(L.text("mobile.reading.libraryTitle"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load(force: true) }
        .sheet(item: $selectedBook) { book in
            NavigationStack {
                bookDetail(book)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button(L.text("mobile.ia.close")) { selectedBook = nil }
                        }
                    }
            }
        }
        .sheet(item: $previewURL) { url in
            NavigationStack {
                WebItemView(title: L.text("mobile.reading.previewTitle"), urlString: url.absoluteString)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button(L.text("mobile.ia.close")) { previewURL = nil }
                        }
                    }
            }
        }
    }

    @ViewBuilder
    private func bookRow(_ book: LibraryBook) -> some View {
        LMSCard {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(book.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                    if let subtitle = ReadingLogic.bookSubtitle(book) {
                        Text(subtitle)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    @ViewBuilder
    private func bookDetail(_ book: LibraryBook) -> some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    Text(book.title)
                        .font(.title3.weight(.semibold))
                    if let author = book.author, !author.isEmpty {
                        Text(author).font(.subheadline)
                    }
                    if let lexile = ReadingLogic.formatLexile(book.lexileLevel) {
                        Text(lexile).font(.caption)
                    }
                    if let band = book.fpBand ?? book.gradeBand {
                        Text(band).font(.caption)
                    }
                    if let summary = book.summary, !summary.isEmpty {
                        Text(summary)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    if let cover = book.coverUrl?.trimmingCharacters(in: .whitespacesAndNewlines),
                       !cover.isEmpty,
                       let url = URL(string: cover) {
                        Button(L.text("mobile.reading.previewOpen")) {
                            previewURL = url
                        }
                        .buttonStyle(.bordered)
                    }
                    Button(L.text("mobile.reading.logThisBook")) {
                        selectedBook = nil
                        onLogBook(book)
                    }
                    .buttonStyle(.borderedProminent)
                }
                .padding(16)
            }
        }
        .navigationTitle(book.title)
        .navigationBarTitleDisplayMode(.inline)
    }

    private func load(force: Bool = false) async {
        guard let token = session.accessToken else { return }
        if !force && !books.isEmpty { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            let filter = LibraryBooksFilter(
                lexileMin: nil,
                lexileMax: nil,
                gradeBand: gradeBand.isEmpty ? nil : gradeBand
            )
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.libraryBooks(orgId: orgId, gradeBand: gradeBand),
                accessToken: token
            ) {
                try await LMSAPI.fetchLibraryBooks(orgId: orgId, filter: filter, accessToken: token)
            }
            books = result.value.sorted { $0.title.localizedCaseInsensitiveCompare($1.title) == .orderedAscending }
        } catch {
            errorMessage = L.text("mobile.reading.error.load")
        }
    }
}

extension URL: @retroactive Identifiable {
    public var id: String { absoluteString }
}