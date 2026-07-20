import SwiftUI

/// Profile tab: identity hero, notifications, app info, and sign-out.
struct ProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.accessibilityPreferences) private var accessibilityPreferences
    @State private var confirmingSignOut = false
    @State private var confirmingClearCache = false
    @State private var confirmingClearSearchHistory = false
    @State private var navigatedMoreDestination: MoreDestination?
    @State private var billingNav: BillingRoute?
    @State private var purchasesNav: MyPurchasesRoute?
    @State private var notificationPrefsNav = false
    @State private var editProfileNav = false
    @State private var learnerProfileNav = false
    @State private var settingsAdminHubNav = false
    @State private var showShareFeedback = false
    @State private var feedbackSuccessMessage: String?
    @State private var localeError: String?

    var body: some View {
        NavigationStack {
            ZStack {
                LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        ProfileIdentityHero()
                        if shell.iaRedesignEnabled {
                            ProfileIaContextCard()
                            ProfileMoreHubCard()
                        }
                        if offline.pendingCount > 0 {
                            ProfileOfflineSyncCard()
                        }
                        ProfilePersonalCard()
                        IntegrationsEntryCard()
                        SettingsAdminHubEntryCard()
                        ArchivedCoursesAdminEntryCard()
                        RolesPermissionsAdminEntryCard()
                        PeopleAdminEntryCard()
                        OrgStructureAdminEntryCard()
                        OrgBrandingAdminEntryCard()
                        AiAdminEntryCard()
                        IntegrationsAdminEntryCard()
                        TranscriptsAdvisingAdminEntryCard()
                        PlatformSettingsAdminEntryCard()
                        ProfileDepthCards()
                        LearnerProfileEntryCard()
                        ShareFeedbackEntryCard {
                            showShareFeedback = true
                        }
                        if let feedbackSuccessMessage {
                            Text(feedbackSuccessMessage)
                                .font(.footnote.weight(.medium))
                                .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .accessibilityLabel(feedbackSuccessMessage)
                        }
                        ProfileOfflineStorageCard(
                            confirmingClearCache: $confirmingClearCache,
                            confirmingClearSearchHistory: $confirmingClearSearchHistory
                        )
                        ProfileAppearanceCard()
                        ProfileLocaleCard(localeError: $localeError)
                        ProfileUIModeCard()
                        accessibilityCard
                        ProfileSecurityCard()
                        ProfileAccountCard(billingNav: $billingNav, purchasesNav: $purchasesNav)
                        ProfileNotificationsCard()
                        ProfileLegalCard()
                        ProfileAboutCard()
                        ProfileSignOutButton(confirmingSignOut: $confirmingSignOut)
                    }
                    .padding(16)
                }
                .refreshable {
                    await shell.refresh(accessToken: session.accessToken)
                }
            }
            .navigationTitle(L.text("mobile.profile.title"))
            .navigationBarTitleDisplayMode(.inline)
            .globalDrawerToolbar()
            .navigationDestination(for: NotificationsRoute.self) { _ in
                NotificationsView()
            }
            .navigationDestination(for: DeviceSessionsRoute.self) { _ in
                DeviceSessionsView()
            }
            .navigationDestination(for: EditProfileRoute.self) { _ in
                EditProfileView()
            }
            .navigationDestination(for: MyAccommodationsRoute.self) { _ in
                MyAccommodationsView()
            }
            .navigationDestination(for: ProfilePersonalDetailsRoute.self) { _ in
                ProfilePersonalDetailsView()
            }
            .navigationDestination(for: ResearchStudiesRoute.self) { _ in
                ResearchStudiesView()
            }
            .navigationDestination(for: LearnerProfileRoute.self) { _ in
                LearnerProfileView()
            }
            .navigationDestination(for: IntegrationsRoute.self) { _ in
                IntegrationsView()
            }
            .navigationDestination(for: ArchivedCoursesAdminRoute.self) { _ in
                ArchivedCoursesAdminView()
            }
            .navigationDestination(for: RolesPermissionsAdminRoute.self) { _ in
                RolesPermissionsAdminView()
            }
            .navigationDestination(for: PeopleAdminRoute.self) { _ in
                PeopleAdminView()
            }
            .navigationDestination(for: OrgStructureAdminRoute.self) { _ in
                OrgStructureAdminView()
            }
            .navigationDestination(for: OrgBrandingAdminRoute.self) { _ in
                OrgBrandingAdminView()
            }
            .navigationDestination(for: AiAdminHubRoute.self) { _ in
                AiAdminHubView()
            }
            .navigationDestination(for: IntegrationsAdminRoute.self) { _ in
                IntegrationsAdminView()
            }
            .navigationDestination(for: TranscriptsAdvisingAdminRoute.self) { _ in
                TranscriptsAdvisingAdminView()
            }
            .navigationDestination(for: PlatformSettingsAdminRoute.self) { _ in
                PlatformSettingsView()
            }
            .navigationDestination(for: SettingsAdminHubRoute.self) { _ in
                SettingsAdminHubView()
            }
            .navigationDestination(for: AuditLogAdminRoute.self) { _ in
                AuditLogAdminView()
            }
            .navigationDestination(isPresented: $settingsAdminHubNav) {
                SettingsAdminHubView()
            }
            .navigationDestination(for: MoreHubRoute.self) { _ in
                MoreHubView()
            }
            .navigationDestination(item: $billingNav) { _ in
                BillingView()
            }
            .navigationDestination(item: $purchasesNav) { _ in
                MyPurchasesView()
            }
            .navigationDestination(for: MoreDestination.self) { destination in
                ProfileMoreDestinationScreen(destination: destination)
            }
            .navigationDestination(item: $navigatedMoreDestination) { destination in
                ProfileMoreDestinationScreen(destination: destination)
            }
            .confirmationDialog(
                L.text("mobile.profile.clearCacheConfirm"),
                isPresented: $confirmingClearCache,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.profile.clearCache"), role: .destructive) {
                    Task { await offline.clearStorage() }
                }
            } message: {
                Text(L.text("mobile.profile.clearCacheMessage"))
            }
            .confirmationDialog(
                L.text("mobile.search.clearHistoryConfirm"),
                isPresented: $confirmingClearSearchHistory,
                titleVisibility: .visible
            ) {
                Button(L.text("mobile.search.clearHistory"), role: .destructive) {
                    SearchRecentsStore.clearAll()
                }
            } message: {
                Text(L.text("mobile.search.clearHistoryMessage"))
            }
            .task { await offline.refreshState() }
            .navigationDestination(isPresented: $notificationPrefsNav) {
                NotificationPreferencesView()
            }
            .navigationDestination(isPresented: $editProfileNav) {
                EditProfileView()
            }
            .navigationDestination(isPresented: $learnerProfileNav) {
                LearnerProfileView()
            }
            .sheet(isPresented: $showShareFeedback) {
                ShareFeedbackView(isPresented: $showShareFeedback) {
                    feedbackSuccessMessage = L.text("mobile.feedback.success")
                }
            }
            .onAppear {
                openPendingMoreDestinationIfNeeded()
                openPendingProfileSettingsIfNeeded()
                if shell.consumePendingBilling() {
                    billingNav = BillingRoute()
                }
            }
            // The profile pane is mounted permanently (toggled by opacity), so
            // `onAppear` only fires once at launch. Deep links routed here in-session
            // (e.g. the drawer's "System settings" → admin hub) set the pending route
            // after that, so consume it reactively too.
            .onChange(of: shell.pendingProfileSettingsRoute) { _, route in
                if route != nil { openPendingProfileSettingsIfNeeded() }
            }
        }
    }

    private func openPendingMoreDestinationIfNeeded() {
        guard let destination = shell.consumePendingMoreDestination() else { return }
        navigatedMoreDestination = destination
    }

    private var accessibilityCard: some View {
        LMSCard {
            Text(L.text("mobile.profile.accessibility"))
                .font(LexturesTheme.displayFont(17))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Toggle(isOn: Binding(
                get: { accessibilityPreferences.dyslexiaDisplayEnabled },
                set: { accessibilityPreferences.dyslexiaDisplayEnabled = $0 }
            )) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(L.text("mobile.profile.dyslexiaFriendly"))
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Text(L.text("mobile.profile.dyslexiaFriendlyDescription"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }
            }
        }
    }

    private func openPendingProfileSettingsIfNeeded() {
        guard let route = shell.consumePendingProfileSettingsRoute() else { return }
        switch route {
        case .account:
            editProfileNav = true
        case .notifications:
            notificationPrefsNav = true
        case .learnerProfile:
            if LearnerProfileLogic.learnerProfileEnabled(shell.platformFeatures) {
                learnerProfileNav = true
            }
        case .adminHub, .auditLog:
            if SettingsMenuLogic.shouldShowHubEntry(
                features: shell.platformFeatures,
                permissions: shell.permissions
            ) {
                settingsAdminHubNav = true
            }
        }
    }
}
