import SwiftUI

/// People admin entry in profile settings (M14.3).
struct PeopleAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if PeopleAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: PeopleAdminRoute(),
                    systemImage: "person.3",
                    title: L.text("mobile.admin.people.title"),
                    subtitle: L.text("mobile.admin.people.entry.subtitle")
                )
            }
        }
    }
}
