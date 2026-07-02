import SwiftUI

/// Instructor shell tab: grading backlog and teaching shortcuts.
struct TeachHubView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var model = DashboardModel()

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        if shell.iaRedesignEnabled {
                            ShellHeaderBar { shell.showUniversalSearch = true }
                        }

                        LMSSectionHeader(title: L.text("mobile.ia.teach.title"), systemImage: "checkmark.circle")

                        if let error = model.errorMessage {
                            LMSErrorBanner(message: error)
                        }

                        if model.loading && model.staffBacklogs.isEmpty {
                            LMSSkeletonList(count: 3)
                        } else if model.staffBacklogs.isEmpty {
                            LMSEmptyState(
                                systemImage: "tray",
                                title: L.text("mobile.ia.teach.emptyTitle"),
                                message: L.text("mobile.ia.teach.emptyMessage")
                            )
                        } else {
                            ForEach(model.staffBacklogs) { backlog in
                                NavigationLink(value: backlog) {
                                    LMSCard {
                                        HStack {
                                            VStack(alignment: .leading, spacing: 4) {
                                                Text(backlog.course.displayTitle)
                                                    .font(LexturesTheme.displayFont(16))
                                                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                                                Text(L.plural("mobile.ia.teach.ungraded", count: backlog.total))
                                                    .font(.caption)
                                                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                            }
                                            Spacer(minLength: 0)
                                            Image(systemName: "chevron.right")
                                                .font(.caption.weight(.semibold))
                                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                                        }
                                    }
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .padding(16)
                }
                .refreshable {
                    await model.load(accessToken: session.accessToken, force: true)
                }
            }
            .navigationTitle(L.text("mobile.ia.tabs.teach"))
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .navigationDestination(for: StaffBacklog.self) { backlog in
                GradingBacklogView(course: backlog.course)
            }
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .task { await model.load(accessToken: session.accessToken) }
        }
    }
}