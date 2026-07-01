import SwiftUI

/// Profile depth entry points (M1.5) — hidden when nothing to show (FR-5).
struct ProfileDepthCards: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var personalDetailsVisible = false
    @State private var researchVisible = false

    var body: some View {
        Group {
            if shell.profileDepthEnabled {
                if personalDetailsVisible {
                    LMSCard {
                        SettingsNavigationRow(
                            route: ProfilePersonalDetailsRoute(),
                            systemImage: "list.bullet.rectangle",
                            title: L.text("mobile.profileDepth.personalDetails.title"),
                            subtitle: L.text("mobile.profileDepth.personalDetails.subtitle")
                        )
                    }
                }
                if researchVisible {
                    LMSCard {
                        SettingsNavigationRow(
                            route: ResearchStudiesRoute(),
                            systemImage: "shield.lefthalf.filled",
                            title: L.text("mobile.profileDepth.research.title"),
                            subtitle: L.text("mobile.profileDepth.research.subtitle")
                        )
                    }
                }
            }
        }
        .task(id: shell.profileDepthEnabled) {
            await refreshVisibility()
        }
    }

    @MainActor
    private func refreshVisibility() async {
        guard shell.profileDepthEnabled, let token = session.accessToken else {
            personalDetailsVisible = false
            researchVisible = false
            return
        }
        var fields = 0
        if shell.platformFeatures.customFieldsEnabled {
            fields = (try? await LMSAPI.fetchMyProfileFields(accessToken: token).fields.count) ?? 0
        }
        personalDetailsVisible = ProfileDepthLogic.shouldShowPersonalDetails(
            customFieldsEnabled: shell.platformFeatures.customFieldsEnabled,
            demographicsEnabled: shell.platformFeatures.ffDemographics,
            fieldCount: fields
        )
        if shell.platformFeatures.ffResearchConsent {
            let pending = (try? await LMSAPI.fetchPendingConsentStudies(accessToken: token).count) ?? 0
            let history = (try? await LMSAPI.fetchConsentHistory(accessToken: token).count) ?? 0
            researchVisible = ProfileDepthLogic.shouldShowResearchStudies(
                researchConsentEnabled: true,
                pendingCount: pending,
                historyCount: history
            )
        } else {
            researchVisible = false
        }
    }
}