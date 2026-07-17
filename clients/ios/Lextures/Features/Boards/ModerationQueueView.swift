import SwiftUI

/// Manager moderation queue: pending / reported / flagged (VC.M7).
struct ModerationQueueView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.dismiss) private var dismiss
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String
    let boardId: String
    var onChanged: () -> Void

    private enum Tab: String, CaseIterable, Identifiable {
        case pending, reports, flagged
        var id: String { rawValue }
    }

    @State private var tab: Tab = .pending
    @State private var queue: BoardModerationQueue?
    @State private var loading = true
    @State private var busyId: String?
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("", selection: $tab) {
                    Text("\(L.text("mobile.boards.moderation.tabPending")) (\(queue?.pendingPosts.count ?? 0))")
                        .tag(Tab.pending)
                    Text("\(L.text("mobile.boards.moderation.tabReports")) (\(queue?.userReports.count ?? 0))")
                        .tag(Tab.reports)
                    Text("\(L.text("mobile.boards.moderation.tabFlagged")) (\(queue?.flaggedReports.count ?? 0))")
                        .tag(Tab.flagged)
                }
                .pickerStyle(.segmented)
                .padding(16)
                .accessibilityLabel(L.text("mobile.boards.moderation.title"))

                if let errorMessage {
                    LMSErrorBanner(message: errorMessage)
                        .padding(.horizontal, 16)
                }

                if loading && queue == nil {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollView {
                        LazyVStack(alignment: .leading, spacing: 12) {
                            switch tab {
                            case .pending:
                                pendingList
                            case .reports:
                                reportList(queue?.userReports ?? [])
                            case .flagged:
                                reportList(queue?.flaggedReports ?? [])
                            }
                        }
                        .padding(16)
                    }
                }
            }
            .background(LexturesTheme.sceneBackground(for: colorScheme))
            .navigationTitle(L.text("mobile.boards.moderation.title"))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L.text("mobile.common.close")) { dismiss() }
                }
            }
            .task { await load() }
        }
    }

    @ViewBuilder
    private var pendingList: some View {
        let posts = queue?.pendingPosts ?? []
        if posts.isEmpty {
            Text(L.text("mobile.boards.moderation.emptyPending"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            ForEach(posts) { post in
                LMSCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(post.title.isEmpty ? L.text("mobile.boards.moderation.untitled") : post.title)
                            .font(.subheadline.weight(.semibold))
                        let preview = BoardsLogic.bodyPlainText(post)
                        if !preview.isEmpty {
                            Text(preview)
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                .lineLimit(3)
                        }
                        HStack(spacing: 8) {
                            Button(L.text("mobile.boards.moderation.approve")) {
                                Task { await runPostAction(post.id, approve: true) }
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(busyId == post.id)
                            .accessibilityLabel(L.text("mobile.boards.moderation.approve"))

                            Button(L.text("mobile.boards.moderation.reject")) {
                                Task { await runPostAction(post.id, approve: false) }
                            }
                            .buttonStyle(.bordered)
                            .disabled(busyId == post.id)
                            .accessibilityLabel(L.text("mobile.boards.moderation.reject"))
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private func reportList(_ reports: [BoardReport]) -> some View {
        if reports.isEmpty {
            Text(L.text("mobile.boards.moderation.emptyReports"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        } else {
            ForEach(reports) { report in
                LMSCard {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(kindLabel(report.kind))
                            .font(.caption.weight(.semibold))
                        if !report.reason.isEmpty {
                            Text(report.reason)
                                .font(.subheadline)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        HStack(spacing: 8) {
                            Button(L.text("mobile.boards.moderation.dismiss")) {
                                Task { await runReportAction(report.id, action: "dismiss") }
                            }
                            .buttonStyle(.bordered)
                            .disabled(busyId == report.id)
                            .accessibilityLabel(L.text("mobile.boards.moderation.dismiss"))

                            Button(L.text("mobile.boards.moderation.hide")) {
                                Task { await runReportAction(report.id, action: "hide") }
                            }
                            .buttonStyle(.bordered)
                            .disabled(busyId == report.id)
                            .accessibilityLabel(L.text("mobile.boards.moderation.hide"))

                            Button(L.text("mobile.boards.moderation.remove"), role: .destructive) {
                                Task { await runReportAction(report.id, action: "remove") }
                            }
                            .buttonStyle(.bordered)
                            .disabled(busyId == report.id)
                            .accessibilityLabel(L.text("mobile.boards.moderation.remove"))
                        }
                    }
                }
            }
        }
    }

    private func kindLabel(_ kind: String) -> String {
        switch kind.lowercased() {
        case "filter": return L.text("mobile.boards.moderation.kind.filter")
        case "av_blocked": return L.text("mobile.boards.moderation.kind.av_blocked")
        default: return L.text("mobile.boards.moderation.kind.user")
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = true
        errorMessage = nil
        defer { loading = false }
        do {
            queue = try await LMSAPI.fetchBoardModerationQueue(
                courseCode: courseCode,
                boardId: boardId,
                accessToken: token
            )
        } catch {
            errorMessage = L.text("mobile.boards.moderation.loadError")
        }
    }

    private func runPostAction(_ postId: String, approve: Bool) async {
        guard let token = session.accessToken else { return }
        busyId = postId
        defer { busyId = nil }
        do {
            if approve {
                _ = try await LMSAPI.approveBoardPost(
                    courseCode: courseCode, boardId: boardId, postId: postId, accessToken: token
                )
            } else {
                _ = try await LMSAPI.rejectBoardPost(
                    courseCode: courseCode, boardId: boardId, postId: postId, accessToken: token
                )
            }
            await load()
            onChanged()
        } catch {
            errorMessage = L.text("mobile.boards.moderation.actionError")
        }
    }

    private func runReportAction(_ reportId: String, action: String) async {
        guard let token = session.accessToken else { return }
        busyId = reportId
        defer { busyId = nil }
        do {
            _ = try await LMSAPI.resolveBoardReport(
                courseCode: courseCode,
                boardId: boardId,
                reportId: reportId,
                action: action,
                accessToken: token
            )
            await load()
            onChanged()
        } catch {
            errorMessage = L.text("mobile.boards.moderation.actionError")
        }
    }
}
