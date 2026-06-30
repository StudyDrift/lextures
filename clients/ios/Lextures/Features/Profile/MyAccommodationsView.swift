import SwiftUI

/// Navigation marker for the my-accommodations screen.
struct MyAccommodationsRoute: Hashable {}

/// A single plain-language support line within an accommodation card.
private struct AccommodationSupport: Identifiable {
    let id = UUID()
    let icon: String
    let text: String
}

/// Lists the student's currently active accommodations in plain language (FR-3).
/// Read-only — granting/approving lives on web/admin (Non-Goal §3). Data is the
/// student's own, so it is safe to show here (NFR privacy).
struct MyAccommodationsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    @State private var phase: Phase = .loading

    private enum Phase {
        case loading
        case failed
        case loaded([MyAccommodation])
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            switch phase {
            case .loading:
                ProgressView().controlSize(.large)
            case .failed:
                failed
            case let .loaded(items):
                if items.isEmpty {
                    emptyState
                } else {
                    list(items)
                }
            }
        }
        .navigationTitle(L.text("mobile.accommodations.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private func list(_ items: [MyAccommodation]) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text(L.text("mobile.accommodations.intro"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                ForEach(items) { item in
                    card(for: item)
                }
            }
            .padding(16)
        }
        .refreshable { await load() }
    }

    private func card(for item: MyAccommodation) -> some View {
        LMSCard {
            HStack(spacing: 10) {
                Image(systemName: "checkmark.seal.fill")
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                Text(scopeTitle(for: item))
                    .font(LexturesTheme.displayFont(17))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            }
            ForEach(supports(for: item)) { support in
                HStack(alignment: .top, spacing: 10) {
                    Image(systemName: support.icon)
                        .font(.footnote)
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                        .frame(width: 22)
                    Text(support.text)
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 0)
                }
            }
            if let window = effectiveWindow(for: item) {
                Divider()
                Text(window)
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            }
        }
    }

    private func scopeTitle(for item: MyAccommodation) -> String {
        if let code = item.courseCode, !code.isEmpty {
            return L.format("mobile.accommodations.scopeCourse", code)
        }
        return L.text("mobile.accommodations.scopeAll")
    }

    /// Maps the boolean flags to plain-language lines with "where it applies".
    private func supports(for item: MyAccommodation) -> [AccommodationSupport] {
        var out: [AccommodationSupport] = []
        if item.hasExtendedTime {
            out.append(.init(icon: "clock.fill", text: L.text("mobile.accommodations.extendedTime")))
        }
        if item.hasExtraAttempts {
            out.append(.init(icon: "arrow.clockwise", text: L.text("mobile.accommodations.extraAttempts")))
        }
        if item.hintsAlwaysAvailable {
            out.append(.init(icon: "lightbulb.fill", text: L.text("mobile.accommodations.hints")))
        }
        if item.reducedDistractionRecommended {
            out.append(.init(icon: "rectangle.dashed", text: L.text("mobile.accommodations.reducedDistraction")))
        }
        if item.speechToTextEnabled {
            out.append(.init(icon: "mic.fill", text: L.text("mobile.accommodations.speechToText")))
        }
        if item.ttsEnabled {
            out.append(.init(icon: "speaker.wave.2.fill", text: L.text("mobile.accommodations.readAloud")))
        }
        if item.dyslexiaDisplayEnabled {
            out.append(.init(icon: "textformat", text: L.text("mobile.accommodations.dyslexiaDisplay")))
        }
        if item.highContrastEnabled {
            out.append(.init(icon: "circle.lefthalf.filled", text: L.text("mobile.accommodations.highContrast")))
        }
        if item.reducedMotionEnabled {
            out.append(.init(icon: "wind", text: L.text("mobile.accommodations.reducedMotion")))
        }
        if item.separateSetting {
            out.append(.init(icon: "door.left.hand.open", text: L.text("mobile.accommodations.separateSetting")))
        }
        return out
    }

    private func effectiveWindow(for item: MyAccommodation) -> String? {
        switch (item.effectiveFrom, item.effectiveUntil) {
        case let (from?, until?):
            return L.format("mobile.accommodations.windowBetween", from, until)
        case let (from?, nil):
            return L.format("mobile.accommodations.windowFrom", from)
        case let (nil, until?):
            return L.format("mobile.accommodations.windowUntil", until)
        case (nil, nil):
            return nil
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "checkmark.circle")
                .font(.system(size: 44))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.text("mobile.accommodations.emptyTitle"))
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
            Text(L.text("mobile.accommodations.emptyBody"))
                .font(.subheadline)
                .multilineTextAlignment(.center)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
        }
        .padding(32)
    }

    private var failed: some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 40))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.text("mobile.accommodations.loadError"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.common.retry")) {
                Task { await load() }
            }
            .font(.subheadline.weight(.semibold))
        }
        .padding(32)
    }

    @MainActor
    private func load() async {
        guard let token = session.accessToken else {
            phase = .failed
            return
        }
        do {
            let items = try await LMSAPI.fetchMyAccommodations(accessToken: token)
                .filter { !$0.isEmpty }
            phase = .loaded(items)
        } catch {
            phase = .failed
        }
    }
}
