import SwiftUI

/// In-course onboarding progress for the intro course modules tab (IC07).
struct IntroCourseProgressRail: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    let courseCode: String

    @State private var progress: IntroCourseProgress?
    @State private var loading = true
    @State private var loadFailed = false

    var body: some View {
        Group {
            if IntroCourseLogic.introCourseEnabled(shell.platformFeatures),
               IntroCourseLogic.isIntroCourse(courseCode),
               !loading,
               !loadFailed,
               let progress,
               progress.enrolled {
                rail(progress)
            }
        }
        .task(id: session.accessToken) { await load() }
    }

    private func rail(_ progress: IntroCourseProgress) -> some View {
        LMSCard(accent: LexturesTheme.primary) {
            Text(L.text("mobile.introCourse.rail.title"))
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            IntroCourseProgressBar(
                percent: progress.percent,
                modulesComplete: progress.modulesComplete,
                modulesTotal: progress.modulesTotal
            )

            if let nextTitle = progress.nextItem?.title, progress.completedAt == nil {
                Text(L.format("mobile.introCourse.rail.nextUp", nextTitle))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            if let modules = progress.modules, !modules.isEmpty {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(modules) { module in
                        HStack(spacing: 8) {
                            moduleIcon(module.status)
                            Text(module.title)
                                .font(.caption)
                                .foregroundStyle(module.status == "current"
                                    ? LexturesTheme.textPrimary(for: colorScheme)
                                    : LexturesTheme.textSecondary(for: colorScheme))
                                .fontWeight(module.status == "current" ? .semibold : .regular)
                            Spacer(minLength: 0)
                        }
                        .accessibilityLabel(moduleAccessibilityLabel(module))
                    }
                }
                .padding(.top, 4)
                .accessibilityElement(children: .contain)
                .accessibilityLabel(L.text("mobile.introCourse.rail.modulesAria"))
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.introCourse.rail.ariaLabel"))
    }

    @ViewBuilder
    private func moduleIcon(_ status: String) -> some View {
        switch status {
        case "done":
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(LexturesTheme.primary)
        case "current":
            Image(systemName: "circle.inset.filled")
                .foregroundStyle(LexturesTheme.primary)
        default:
            Image(systemName: "circle")
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.5))
        }
    }

    private func moduleAccessibilityLabel(_ module: IntroCourseModuleProgress) -> String {
        let statusKey: String.LocalizationValue = switch module.status {
        case "done": "mobile.introCourse.rail.statusDone"
        case "current": "mobile.introCourse.rail.statusCurrent"
        default: "mobile.introCourse.rail.statusUpcoming"
        }
        return "\(module.title), \(L.text(statusKey))"
    }

    private func load() async {
        guard IntroCourseLogic.introCourseEnabled(shell.platformFeatures),
              IntroCourseLogic.isIntroCourse(courseCode),
              let token = session.accessToken else {
            loading = false
            return
        }
        loading = true
        loadFailed = false
        defer { loading = false }
        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.introCourseProgress(),
                accessToken: token
            ) {
                try await LMSAPI.fetchIntroCourseProgress(accessToken: token)
            }
            progress = result.value
        } catch {
            loadFailed = true
        }
    }
}