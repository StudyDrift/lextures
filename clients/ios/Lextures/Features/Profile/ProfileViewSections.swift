import SwiftUI

struct ProfileIaContextCard: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        Group {
            if shell.roleSnapshot.availableContexts.count > 1 {
                LMSCard {
                    Text(L.text("mobile.ia.context.title"))
                        .font(LexturesTheme.displayFont(17))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Picker(L.text("mobile.ia.context.title"), selection: Binding(
                        get: { shell.activeRoleContext },
                        set: { shell.setRoleContext($0) }
                    )) {
                        ForEach(shell.roleSnapshot.availableContexts, id: \.self) { context in
                            Text(context.label).tag(context)
                        }
                    }
                    .pickerStyle(.segmented)
                }
            }
        }
    }
}

struct ProfileMoreHubCard: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        Group {
            if !MobileDestinations.moreDestinations(
                context: shell.activeRoleContext,
                platform: shell.platformFeatures
            ).isEmpty {
                LMSCard {
                    NavigationLink(value: MoreHubRoute()) {
                        HStack(spacing: 12) {
                            Image(systemName: "square.grid.2x2.fill")
                                .font(.footnote.weight(.semibold))
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            Text(L.text("mobile.ia.more.title"))
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                            Spacer(minLength: 0)
                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }
}

struct ProfileOfflineSyncCard: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            Text(L.text("mobile.profile.pendingSync"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.plural("mobile.pendingSync.waiting", count: offline.pendingCount))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            ForEach(offline.outboxItems.filter {
                $0.status == .queued || $0.status == .failed || $0.status == .conflict
            }) { item in
                Divider()
                VStack(alignment: .leading, spacing: 6) {
                    Text(item.label)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    OutboxStatusChip(status: item.status)
                    if item.status == .failed || item.status == .conflict {
                        Button(L.text("mobile.profile.retry")) {
                            Task { await offline.retryOutboxItem(id: item.id, accessToken: session.accessToken) }
                        }
                        .font(.caption.weight(.semibold))
                    }
                }
            }
        }
    }
}

struct ProfileNotificationsCard: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        LMSCard {
            NavigationLink(value: NotificationsRoute()) {
                HStack(spacing: 12) {
                    Image(systemName: "bell.fill")
                        .font(.footnote.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        .frame(width: 32, height: 32)
                        .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.14))
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                    VStack(alignment: .leading, spacing: 2) {
                        Text(L.text("mobile.profile.notifications"))
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        Text(
                            shell.unreadNotifications > 0
                                ? L.format("mobile.profile.unread", shell.unreadNotifications)
                                : L.text("mobile.dashboard.caughtUp")
                        )
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }
                    Spacer(minLength: 0)
                    if shell.unreadNotifications > 0 {
                        Text("\(shell.unreadNotifications)")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(LexturesTheme.coral)
                            .clipShape(Capsule())
                    }
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                        .flipsForRightToLeftLayoutDirection(true)
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .simultaneousGesture(TapGesture().onEnded {
                Task { await PushManager.shared.requestPermissionIfNeeded() }
            })
        }
    }
}

struct ProfileMoreDestinationScreen: View {
    @Environment(AppShellModel.self) private var shell
    let destination: MoreDestination

    var body: some View {
        if destination == .library && shell.platformFeatures.libraryBrowseEnabled {
            LibraryBrowseView()
        } else if destination == .askAi {
            TutorChatView(mode: .askAi)
        } else if destination == .peerReviews {
            PeerReviewListView()
        } else if destination == .reportCards {
            ReportCardListView()
        } else if destination == .catalog {
            if shell.platformFeatures.ffPublicCatalog {
                CatalogView()
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .paths {
            if shell.platformFeatures.ffLearningPaths {
                MyPathsView()
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .insights {
            if shell.platformFeatures.selfReflectionEnabled {
                InsightsView(
                    onOpenCourse: { course in
                        shell.activeCourse = course
                        shell.activeCourseRoot = .profile
                        shell.activeCourseSection = .modules
                        shell.select(.courses)
                    },
                    onOpenReview: { shell.select(.review) }
                )
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .reading {
            if shell.platformFeatures.ffLibrary {
                ReadingDashboardView { course in
                    shell.activeCourse = course
                    shell.activeCourseRoot = .profile
                    shell.activeCourseSection = .groups
                    shell.select(.courses)
                }
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .credentials {
            if CredentialsLogic.credentialsEnabled(shell.platformFeatures) {
                CredentialsView()
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .gamification {
            if GamificationLogic.gamificationEnabled(shell.platformFeatures) {
                GamificationView()
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else if destination == .advising {
            if AdvisingLogic.advisingEnabled(shell.platformFeatures) {
                AdvisingView()
            } else {
                MoreDestinationPlaceholder(destination: destination)
            }
        } else {
            MoreDestinationPlaceholder(destination: destination)
        }
    }
}
