import SwiftUI

/// Watches intro-course progress and presents the completion celebration once (IC07).
struct IntroCelebrationPresenter: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline

    @Binding var isPresented: Bool
    @Binding var progress: IntroCourseProgress?

    var body: some View {
        Color.clear
            .frame(width: 0, height: 0)
            .task(id: session.accessToken) { await evaluate() }
    }

    private func evaluate() async {
        guard IntroCourseLogic.introCourseEnabled(shell.platformFeatures),
              let token = session.accessToken else { return }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.introCourseProgress(),
                accessToken: token
            ) {
                try await LMSAPI.fetchIntroCourseProgress(accessToken: token)
            }
            if IntroCourseLogic.shouldShowCelebration(result.value) {
                progress = result.value
                isPresented = true
            }
        } catch {
            // Celebration is optional; ignore fetch errors.
        }
    }
}