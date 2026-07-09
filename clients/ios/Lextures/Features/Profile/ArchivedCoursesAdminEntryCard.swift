import SwiftUI

/// Archived courses admin entry in profile settings (M14.10).
struct ArchivedCoursesAdminEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if ArchivedCoursesAdminLogic.shouldShowEntry(
            features: shell.platformFeatures,
            permissions: shell.permissions
        ) {
            LMSCard {
                SettingsNavigationRow(
                    route: ArchivedCoursesAdminRoute(),
                    systemImage: "archivebox",
                    title: L.text("mobile.admin.archivedCourses.title"),
                    subtitle: L.text("mobile.admin.archivedCourses.entry.subtitle")
                )
            }
        }
    }
}