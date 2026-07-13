import SwiftUI

/// Org branding & AI admin entry in profile settings (M14.5).
struct OrgBrandingAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if OrgBrandingAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: OrgBrandingAdminRoute(),
                    systemImage: "paintpalette.fill",
                    title: L.text("mobile.admin.orgBranding.title"),
                    subtitle: L.text("mobile.admin.orgBranding.entry.subtitle")
                )
            }
        }
    }
}
