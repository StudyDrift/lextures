import SwiftUI

struct CalendarSubscribeSheet: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.dismiss) private var dismiss

    let courses: [CourseSummary]

    @State private var loading = true
    @State private var errorMessage: String?
    @State private var tokenInfo: CalendarTokenInfo?
    @State private var createdToken: CalendarTokenCreated?
    @State private var copiedMessage: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 14) {
                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 24)
                    }
                    if let errorMessage {
                        LMSErrorBanner(message: errorMessage)
                    }
                    Text(L.text("mobile.planner.subscribe.message"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let feedURL = resolvedPersonalFeedURL {
                        feedCard(title: L.text("mobile.planner.subscribe.allCourses"), url: feedURL)
                    }

                    if let feeds = tokenInfo?.courseFeeds, !feeds.isEmpty {
                        ForEach(feeds, id: \.courseCode) { feed in
                            if let url = resolvedFeedURL(feed.feedUrl) {
                                feedCard(title: feed.title, url: url)
                            }
                        }
                    }

                    Button {
                        Task { await regenerateToken() }
                    } label: {
                        Label(L.text("mobile.planner.subscribe.generate"), systemImage: "arrow.clockwise")
                    }
                    .buttonStyle(.borderedProminent)

                    if let copiedMessage {
                        Text(copiedMessage)
                            .font(.caption)
                            .foregroundStyle(LexturesTheme.brandTeal)
                    }
                }
                .padding(16)
            }
            .navigationTitle(L.text("mobile.planner.subscribe.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.onboarding.back")) { dismiss() }
                }
            }
            .task { await load() }
        }
    }

    private var resolvedPersonalFeedURL: String? {
        if let created = createdToken?.feedUrl { return created }
        if let token = createdToken?.token, let template = tokenInfo?.personalFeedUrl {
            return template.replacingOccurrences(of: "<token>", with: token)
        }
        return nil
    }

    private func resolvedFeedURL(_ template: String) -> String? {
        guard let token = createdToken?.token else { return nil }
        return template.replacingOccurrences(of: "<token>", with: token)
    }

    @ViewBuilder
    private func feedCard(title: String, url: String) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 8) {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                Text(url)
                    .font(.caption2)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .textSelection(.enabled)
                HStack {
                    Button(L.text("mobile.planner.subscribe.copy")) {
                        UIPasteboard.general.string = url
                        copiedMessage = L.text("mobile.planner.subscribe.copied")
                    }
                    .buttonStyle(.bordered)
                    if let subscribeURL = URL(string: "webcal://\(url.replacingOccurrences(of: "https://", with: "").replacingOccurrences(of: "http://", with: ""))") {
                        Link(L.text("mobile.planner.subscribe.open"), destination: subscribeURL)
                            .buttonStyle(.borderedProminent)
                    }
                }
            }
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            tokenInfo = try await LMSAPI.fetchCalendarTokenInfo(accessToken: token)
            if tokenInfo?.hasToken != true {
                createdToken = try await LMSAPI.createCalendarToken(accessToken: token)
                tokenInfo = try await LMSAPI.fetchCalendarTokenInfo(accessToken: token)
            }
        } catch {
            errorMessage = L.text("mobile.planner.subscribe.error")
        }
    }

    private func regenerateToken() async {
        guard let token = session.accessToken else { return }
        do {
            createdToken = try await LMSAPI.createCalendarToken(accessToken: token)
            tokenInfo = try await LMSAPI.fetchCalendarTokenInfo(accessToken: token)
            copiedMessage = nil
        } catch {
            errorMessage = L.text("mobile.planner.subscribe.error")
        }
    }
}
