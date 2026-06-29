import SwiftUI

// MARK: - Shell state

enum AppTab: String, CaseIterable, Identifiable {
    case home, courses, notebooks, inbox, profile

    var id: String { rawValue }

    var label: String {
        switch self {
        case .home: return L.text("tabs.home")
        case .courses: return L.text("tabs.courses")
        case .notebooks: return L.text("tabs.notebooks")
        case .inbox: return L.text("tabs.inbox")
        case .profile: return L.text("tabs.profile")
        }
    }

    var systemImage: String {
        switch self {
        case .home: return "house.fill"
        case .courses: return "books.vertical.fill"
        case .notebooks: return "square.and.pencil"
        case .inbox: return "tray.fill"
        case .profile: return "person.fill"
        }
    }
}

/// Cross-tab state: selected tab, viewer profile, and unread counters.
/// Single source for the tab badge, Home stat card, and notification bell dot.
@MainActor
@Observable
final class AppShellModel {
    var selectedTab: AppTab = .home
    var profile: MeProfile?
    var unreadInbox = 0
    var unreadNotifications = 0
    var pendingDeepLink: DeepLinkDestination?

    func openDeepLink(_ destination: DeepLinkDestination) {
        pendingDeepLink = destination
        switch destination {
        case .home:
            selectedTab = .home
        case .inbox:
            selectedTab = .inbox
        case .course:
            selectedTab = .courses
        }
    }

    func consumePendingDeepLink() -> DeepLinkDestination? {
        defer { pendingDeepLink = nil }
        return pendingDeepLink
    }

    func refresh(accessToken: String?) async {
        guard let token = accessToken else { return }
        async let me = try? LMSAPI.fetchMe(accessToken: token)
        async let inbox = try? LMSAPI.fetchUnreadInboxCount(accessToken: token)
        async let notifications = try? LMSAPI.fetchNotifications(accessToken: token)
        if let me = await me { profile = me }
        if let inbox = await inbox { unreadInbox = inbox }
        if let page = await notifications { unreadNotifications = page.unreadCount }
    }
}

/// Post-auth shell: Home, Courses, Notebooks, Inbox, Profile behind a floating pill tab bar.
struct MainTabView: View {
    @Environment(AuthSession.self) private var session
    var initialDeepLink: DeepLinkDestination?
    @State private var shell = AppShellModel()

    var body: some View {
        VStack(spacing: 0) {
            tabContent
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            LexturesTabBar(shell: shell)
                .padding(.horizontal, 24)
                .padding(.top, 8)
                .padding(.bottom, 6)
        }
        .environment(shell)
        .task {
            await shell.refresh(accessToken: session.accessToken)
        }
        .onOpenURL { url in
            shell.openDeepLink(DeepLinkRouter.resolve(url.absoluteString))
        }
        .onAppear {
            PushManager.shared.configure(accessToken: { session.accessToken }) { destination in
                shell.openDeepLink(destination)
            }
            if let initialDeepLink {
                shell.openDeepLink(initialDeepLink)
            }
        }
        .onChange(of: session.accessToken) { _, token in
            if token != nil {
                Task { await PushManager.shared.syncTokenWithBackend() }
            }
        }
    }

    /// Keeps every tab's view (and its NavigationStack) alive so switching
    /// tabs never loses scroll position or navigation state.
    private var tabContent: some View {
        ZStack {
            pane(.home) { DashboardView() }
            pane(.courses) { CoursesListView() }
            pane(.notebooks) { NotebooksListView() }
            pane(.inbox) { InboxView() }
            pane(.profile) { ProfileView() }
        }
    }

    private func pane<Content: View>(_ tab: AppTab, @ViewBuilder content: () -> Content) -> some View {
        content()
            .opacity(shell.selectedTab == tab ? 1 : 0)
            .allowsHitTesting(shell.selectedTab == tab)
            .accessibilityHidden(shell.selectedTab != tab)
    }
}

// MARK: - Floating pill tab bar

/// Deep-teal floating capsule: selected tab gets a cream circular "puck".
struct LexturesTabBar: View {
    @Environment(\.colorScheme) private var colorScheme
    @Bindable var shell: AppShellModel

    var body: some View {
        HStack(spacing: 0) {
            ForEach(AppTab.allCases) { tab in
                Button {
                    withAnimation(.spring(duration: 0.3)) {
                        shell.selectedTab = tab
                    }
                } label: {
                    tabIcon(tab)
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.label)
                .accessibilityAddTraits(shell.selectedTab == tab ? [.isSelected] : [])
            }
        }
        .padding(.vertical, 9)
        .padding(.horizontal, 10)
        .background(
            Capsule(style: .continuous)
                .fill(colorScheme == .dark ? LexturesTheme.cardBackgroundDark : LexturesTheme.primaryDeep)
        )
        .overlay(
            Capsule(style: .continuous)
                .stroke(
                    colorScheme == .dark ? LexturesTheme.fieldBorderDark : .white.opacity(0.08),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.primaryDeep.opacity(colorScheme == .dark ? 0 : 0.35), radius: 16, y: 8)
    }

    @ViewBuilder
    private func tabIcon(_ tab: AppTab) -> some View {
        let selected = shell.selectedTab == tab
        ZStack(alignment: .topTrailing) {
            Image(systemName: tab.systemImage)
                .font(.system(size: 17, weight: .semibold))
                .foregroundStyle(
                    selected
                        ? LexturesTheme.primaryDeep
                        : (colorScheme == .dark ? LexturesTheme.textSecondaryDark : .white.opacity(0.72))
                )
                .frame(width: 44, height: 44)
                .background(
                    Circle().fill(selected ? AnyShapeStyle(LexturesTheme.brandCream) : AnyShapeStyle(.clear))
                )

            if tab == .inbox && shell.unreadInbox > 0 {
                Text(shell.unreadInbox > 99 ? "99+" : "\(shell.unreadInbox)")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 5)
                    .padding(.vertical, 2)
                    .background(LexturesTheme.coral)
                    .clipShape(Capsule())
                    .offset(x: 4, y: -2)
            }
        }
    }
}

// MARK: - Shared LMS UI helpers

/// Floating card: generous radius, soft warm shadow, hairline border in dark mode only.
struct LMSCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    var accent: Color? = nil
    @ViewBuilder var content: () -> Content

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(LexturesTheme.cardBackground(for: colorScheme))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(alignment: .leading) {
            if let accent {
                UnevenRoundedRectangle(
                    topLeadingRadius: 18,
                    bottomLeadingRadius: 18,
                    bottomTrailingRadius: 0,
                    topTrailingRadius: 0,
                    style: .continuous
                )
                .fill(accent)
                .frame(width: 4)
            }
        }
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(
                    LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.45),
                    lineWidth: 1
                )
        )
        .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 12, y: 5)
    }
}

/// Serif section header — editorial, like a textbook chapter heading.
struct LMSSectionHeader: View {
    @Environment(\.colorScheme) private var colorScheme
    let title: String
    var systemImage: String?

    var body: some View {
        HStack(spacing: 8) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.footnote.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 26, height: 26)
                    .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.18 : 0.16))
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
            Text(title)
                .font(LexturesTheme.displayFont(19))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .padding(.top, 6)
    }
}

struct LMSErrorBanner: View {
    let message: String

    var body: some View {
        Label(message, systemImage: "exclamationmark.triangle.fill")
            .font(.subheadline)
            .foregroundStyle(LexturesTheme.error)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(14)
            .background(LexturesTheme.error.opacity(0.09))
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }
}

struct LMSEmptyState: View {
    @Environment(\.colorScheme) private var colorScheme
    let systemImage: String
    let title: String
    let message: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 28))
                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                .frame(width: 72, height: 72)
                .background(LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.16 : 0.14))
                .clipShape(Circle())
            Text(title)
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(message)
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 36)
        .padding(.horizontal, 24)
    }
}

/// Rounded gradient tile used as a course "cover" thumbnail.
struct LMSCoverTile: View {
    let key: String
    let systemImage: String
    var size: CGFloat = 48

    var body: some View {
        RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
            .fill(LexturesTheme.coverGradient(for: key))
            .frame(width: size, height: size)
            .overlay(
                Image(systemName: systemImage)
                    .font(.system(size: size * 0.4, weight: .medium))
                    .foregroundStyle(.white)
            )
    }
}

/// Horizontally scrolling pill selector — solid deep teal when selected,
/// white card with hairline border otherwise (inspiration: segmented chips).
struct LMSSegmentedChips<Option: Hashable>: View {
    @Environment(\.colorScheme) private var colorScheme
    let options: [Option]
    @Binding var selection: Option
    let label: (Option) -> String

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(options, id: \.self) { option in
                    let selected = option == selection
                    Button {
                        withAnimation(.easeOut(duration: 0.15)) { selection = option }
                    } label: {
                        Text(label(option))
                            .font(.subheadline.weight(selected ? .semibold : .regular))
                            .padding(.horizontal, 15)
                            .padding(.vertical, 8)
                            .background(
                                selected
                                    ? AnyShapeStyle(LexturesTheme.accent(for: colorScheme))
                                    : AnyShapeStyle(LexturesTheme.cardBackground(for: colorScheme))
                            )
                            .foregroundStyle(
                                selected
                                    ? (colorScheme == .dark ? LexturesTheme.primaryDeep : .white)
                                    : LexturesTheme.textSecondary(for: colorScheme)
                            )
                            .clipShape(Capsule())
                            .overlay(
                                Capsule().stroke(
                                    selected ? .clear : LexturesTheme.fieldBorder(for: colorScheme),
                                    lineWidth: 1
                                )
                            )
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.vertical, 2)
        }
    }
}

/// Initials avatar chip; tapping jumps to the Profile tab.
struct LMSAvatarButton: View {
    @Environment(AppShellModel.self) private var shell
    var size: CGFloat = 34

    var body: some View {
        Button {
            shell.selectedTab = .profile
        } label: {
            Circle()
                .fill(LexturesTheme.heroGradient)
                .frame(width: size, height: size)
                .overlay(
                    Text(shell.profile?.initials ?? "··")
                        .font(.system(size: size * 0.36, weight: .bold))
                        .foregroundStyle(.white)
                )
        }
        .buttonStyle(.plain)
        .accessibilityLabel(L.text("tabs.profile"))
    }
}

/// Card-shaped redacted placeholder shown while a list loads.
struct LMSSkeletonCard: View {
    @Environment(\.colorScheme) private var colorScheme
    var lines: Int = 2

    var body: some View {
        LMSCard {
            HStack(spacing: 12) {
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.6))
                    .frame(width: 44, height: 44)
                VStack(alignment: .leading, spacing: 7) {
                    ForEach(0 ..< lines, id: \.self) { index in
                        RoundedRectangle(cornerRadius: 4)
                            .fill(LexturesTheme.fieldBorder(for: colorScheme).opacity(index == 0 ? 0.7 : 0.45))
                            .frame(width: index == 0 ? 180 : 120, height: 11)
                    }
                }
                Spacer(minLength: 0)
            }
        }
        .redacted(reason: .placeholder)
        .accessibilityHidden(true)
    }
}

/// Stack of skeleton cards for list screens.
struct LMSSkeletonList: View {
    var count: Int = 4

    var body: some View {
        VStack(spacing: 12) {
            ForEach(0 ..< count, id: \.self) { _ in
                LMSSkeletonCard()
            }
        }
    }
}

/// Small circular progress ring (course completion) — echoes the health-app ring.
struct LMSProgressRing: View {
    @Environment(\.colorScheme) private var colorScheme
    let progress: Double // 0...1
    var size: CGFloat = 38
    var tint: Color?

    var body: some View {
        ZStack {
            Circle()
                .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.8), lineWidth: 4)
            Circle()
                .trim(from: 0, to: max(0.02, min(1, progress)))
                .stroke(
                    tint ?? LexturesTheme.accent(for: colorScheme),
                    style: StrokeStyle(lineWidth: 4, lineCap: .round)
                )
                .rotationEffect(.degrees(-90))
            Text("\(Int((progress * 100).rounded()))")
                .font(.system(size: size * 0.3, weight: .bold, design: .rounded))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .frame(width: size, height: size)
        .accessibilityLabel(L.format("mobile.profile.percentComplete", Int((progress * 100).rounded())))
    }
}
