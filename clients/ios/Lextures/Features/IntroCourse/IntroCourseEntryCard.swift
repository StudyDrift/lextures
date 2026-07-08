import SwiftUI

/// Prominent intro-course onboarding card on the mobile dashboard (IC07).
struct IntroCourseEntryCard: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme
    @State private var progress: IntroCourseProgress?
    @State private var loading = true
    @State private var loadFailed = false
    @State private var cacheLabel: String?
    @State private var recordedView = false

    private var state: IntroCourseCardState {
        IntroCourseLogic.cardState(progress: progress, loading: loading, error: loadFailed)
    }

    var body: some View {
        Group {
            switch state {
            case .hidden:
                EmptyView()
            case .loading:
                skeleton
            case .error:
                fallbackCard
            case .notStarted, .inProgress:
                if let progress { activeCard(progress) }
            case .completed:
                if let progress { completedCard(progress) }
            }
        }
        .task(id: session.accessToken) { await load() }
        .onChange(of: state) { _, newState in
            guard !recordedView, newState != .hidden, newState != .loading else { return }
            recordedView = true
            IntroCourseObservability.recordCardView()
        }
    }

    private var skeleton: some View {
        LMSCard {
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.12))
                .frame(width: 88, height: 12)
            RoundedRectangle(cornerRadius: 6, style: .continuous)
                .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.12))
                .frame(height: 20)
                .padding(.top, 4)
            RoundedRectangle(cornerRadius: 8, style: .continuous)
                .fill(LexturesTheme.textSecondary(for: colorScheme).opacity(0.1))
                .frame(height: 8)
                .padding(.top, 8)
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(LexturesTheme.primary.opacity(0.15))
                .frame(width: 140, height: 36)
                .padding(.top, 8)
        }
        .accessibilityLabel(L.text("common.loading"))
    }

    private func activeCard(_ progress: IntroCourseProgress) -> some View {
        let isNotStarted = state == .notStarted
        return LMSCard(accent: LexturesTheme.primary) {
            if let cacheLabel {
                StalenessChip(label: cacheLabel)
            }
            Label {
                Text(isNotStarted
                     ? L.text("mobile.introCourse.card.startHere")
                     : L.text("mobile.introCourse.card.continueOnboarding"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.primary)
            } icon: {
                Image(systemName: "sparkles")
                    .foregroundStyle(LexturesTheme.primary)
            }

            Text(L.text("mobile.introCourse.card.title"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            if let nextTitle = progress.nextItem?.title, !isNotStarted {
                Text(L.format("mobile.introCourse.card.nextUp", nextTitle))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } else {
                Text(L.text("mobile.introCourse.card.subtitle"))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }

            IntroCourseProgressBar(
                percent: progress.percent,
                modulesComplete: progress.modulesComplete,
                modulesTotal: progress.modulesTotal
            )
            .padding(.top, 4)

            Button {
                IntroCourseObservability.recordCtaClick()
                shell.openDeepLink(DeepLinkRouter.resolve(IntroCourseLogic.ctaRoute(for: progress)))
            } label: {
                Text(isNotStarted
                     ? L.text("mobile.introCourse.card.ctaStart")
                     : L.text("mobile.introCourse.card.ctaContinue"))
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(AuthPrimaryButtonStyle())
            .accessibilityHint(L.text("mobile.introCourse.card.ariaLabel"))
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel(L.text("mobile.introCourse.card.ariaLabel"))
    }

    private func completedCard(_ progress: IntroCourseProgress) -> some View {
        LMSCard {
            HStack(spacing: 10) {
                Image(systemName: "checkmark.circle.fill")
                    .foregroundStyle(LexturesTheme.primary)
                Text(L.text("mobile.introCourse.card.completedLabel"))
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                Spacer(minLength: 0)
                Button(L.text("mobile.introCourse.card.revisit")) {
                    IntroCourseObservability.recordCtaClick()
                    shell.openDeepLink(DeepLinkRouter.resolve(
                        IntroCourseLogic.fallbackRoute(courseCode: progress.courseCode ?? IntroCourseConstants.courseCode)
                    ))
                }
                .font(.caption.weight(.semibold))
                .foregroundStyle(LexturesTheme.primary)
            }
        }
        .accessibilityLabel(L.text("mobile.introCourse.card.completedAria"))
    }

    private var fallbackCard: some View {
        LMSCard {
            Label {
                Text(L.text("mobile.introCourse.card.fallbackLabel"))
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            } icon: {
                Image(systemName: "book.closed")
            }
            Button(L.text("mobile.introCourse.card.fallbackLink")) {
                IntroCourseObservability.recordCtaClick()
                shell.openDeepLink(DeepLinkRouter.resolve(IntroCourseLogic.fallbackRoute()))
            }
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(LexturesTheme.primary)
        }
    }

    private func load() async {
        guard IntroCourseLogic.introCourseEnabled(shell.platformFeatures),
              let token = session.accessToken else {
            progress = nil
            loading = false
            loadFailed = false
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
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            loadFailed = true
        }
    }
}

struct IntroCourseProgressBar: View {
    @Environment(\.colorScheme) private var colorScheme
    let percent: Int
    let modulesComplete: Int
    let modulesTotal: Int

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(L.format("mobile.introCourse.progress.modules", "\(max(modulesComplete, 1))", "\(modulesTotal)"))
                .font(.caption2.weight(.semibold))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule()
                        .fill(LexturesTheme.primary.opacity(0.14))
                    Capsule()
                        .fill(LexturesTheme.primary)
                        .frame(width: geo.size.width * CGFloat(min(max(percent, 0), 100)) / 100)
                }
            }
            .frame(height: 8)
            .accessibilityLabel(
                L.format(
                    "mobile.introCourse.progress.ariaLabel",
                    "\(modulesComplete)",
                    "\(modulesTotal)",
                    "\(percent)"
                )
            )
        }
    }
}