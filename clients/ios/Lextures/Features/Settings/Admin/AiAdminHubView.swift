import SwiftUI

struct AiAdminHubRoute: Hashable {}
struct AiModelsSettingsRoute: Hashable {}
struct SystemPromptsRoute: Hashable {}
struct AiReportsRoute: Hashable {}

/// Hub for AI models, system prompts, and usage reports (M14.7).
struct AiAdminHubView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    private var canView: Bool {
        AiModelsAdminLogic.canView(features: shell.platformFeatures, permissions: shell.permissions)
    }

    var body: some View {
        Group {
            if canView {
                content
            } else {
                accessDenied
            }
        }
        .navigationTitle(L.text("mobile.admin.ai.hub.title"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: AiModelsSettingsRoute.self) { _ in
            AiModelsSettingsView()
        }
        .navigationDestination(for: SystemPromptsRoute.self) { _ in
            SystemPromptsView()
        }
        .navigationDestination(for: AiReportsRoute.self) { _ in
            AiReportsView()
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.ai.accessDenied.title"),
            message: L.text("mobile.admin.ai.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.ai.hub.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    LMSCard {
                        VStack(spacing: 0) {
                            SettingsNavigationRow(
                                route: AiModelsSettingsRoute(),
                                systemImage: "cpu",
                                title: L.text("mobile.admin.ai.models.title"),
                                subtitle: L.text("mobile.admin.ai.models.entry.subtitle")
                            )
                            Divider().padding(.leading, 44)
                            SettingsNavigationRow(
                                route: SystemPromptsRoute(),
                                systemImage: "text.alignleft",
                                title: L.text("mobile.admin.ai.prompts.title"),
                                subtitle: L.text("mobile.admin.ai.prompts.entry.subtitle")
                            )
                            Divider().padding(.leading, 44)
                            SettingsNavigationRow(
                                route: AiReportsRoute(),
                                systemImage: "chart.bar",
                                title: L.text("mobile.admin.ai.reports.title"),
                                subtitle: L.text("mobile.admin.ai.reports.entry.subtitle")
                            )
                        }
                    }
                }
                .padding(16)
            }
        }
    }
}
