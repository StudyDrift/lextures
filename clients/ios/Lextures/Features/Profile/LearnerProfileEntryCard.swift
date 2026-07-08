import SwiftUI

/// Learner Profile entry in account settings (LP10).
struct LearnerProfileEntryCard: View {
    @Environment(AppShellModel.self) private var shell

    var body: some View {
        if LearnerProfileLogic.learnerProfileEnabled(shell.platformFeatures) {
            LMSCard {
                SettingsNavigationRow(
                    route: LearnerProfileRoute(),
                    systemImage: "person.text.rectangle",
                    title: L.text("mobile.learnerProfile.title"),
                    subtitle: L.text("mobile.learnerProfile.entry.subtitle")
                )
            }
        }
    }
}