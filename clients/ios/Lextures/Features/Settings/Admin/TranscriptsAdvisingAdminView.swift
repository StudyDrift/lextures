import SwiftUI

struct TranscriptsAdvisingAdminRoute: Hashable {}
struct TranscriptsSettingsRoute: Hashable {}
struct AdvisingSettingsRoute: Hashable {}

/// Hub for transcripts and advising configuration admin (M14.9).
struct TranscriptsAdvisingAdminView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    private var canView: Bool {
        TranscriptsAdvisingAdminLogic.canView(
            features: shell.platformFeatures,
            permissions: shell.permissions
        )
    }

    private var sections: [TranscriptsAdvisingAdminLogic.Section] {
        TranscriptsAdvisingAdminLogic.visibleSections(features: shell.platformFeatures)
    }

    var body: some View {
        Group {
            if canView {
                content
            } else {
                accessDenied
            }
        }
        .navigationTitle(L.text("mobile.admin.transcriptsAdvising.hub.title"))
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(for: TranscriptsSettingsRoute.self) { _ in
            TranscriptsSettingsView()
        }
        .navigationDestination(for: AdvisingSettingsRoute.self) { _ in
            AdvisingSettingsView()
        }
    }

    private var accessDenied: some View {
        LMSEmptyState(
            systemImage: "lock.fill",
            title: L.text("mobile.admin.transcriptsAdvising.accessDenied.title"),
            message: L.text("mobile.admin.transcriptsAdvising.accessDenied.message")
        )
        .padding(16)
    }

    private var content: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    Text(L.text("mobile.admin.transcriptsAdvising.hub.description"))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if sections.isEmpty {
                        LMSEmptyState(
                            systemImage: "doc.text",
                            title: L.text("mobile.admin.transcriptsAdvising.emptyTitle"),
                            message: L.text("mobile.admin.transcriptsAdvising.emptyMessage")
                        )
                    } else {
                        LMSCard {
                            VStack(spacing: 0) {
                                ForEach(Array(sections.enumerated()), id: \.element.id) { index, section in
                                    if index > 0 {
                                        Divider().padding(.leading, 44)
                                    }
                                    sectionRow(section)
                                }
                            }
                        }
                    }
                }
                .padding(16)
            }
        }
    }

    @ViewBuilder
    private func sectionRow(_ section: TranscriptsAdvisingAdminLogic.Section) -> some View {
        switch section {
        case .transcripts:
            SettingsNavigationRow(
                route: TranscriptsSettingsRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        case .advising:
            SettingsNavigationRow(
                route: AdvisingSettingsRoute(),
                systemImage: section.systemImage,
                title: L.text(section.titleKey),
                subtitle: L.text(section.subtitleKey)
            )
        }
    }
}
