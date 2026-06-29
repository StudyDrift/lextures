import SwiftUI

/// Non-blocking staleness indicator for cached read screens.
struct StalenessChip: View {
    @Environment(\.colorScheme) private var colorScheme
    let label: String

    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: "clock.arrow.circlepath")
                .font(.caption2.weight(.semibold))
            Text(label)
                .font(.caption2.weight(.medium))
        }
        .foregroundStyle(LexturesTheme.amber)
        .padding(.horizontal, 10)
        .padding(.vertical, 5)
        .background(LexturesTheme.amber.opacity(colorScheme == .dark ? 0.16 : 0.12))
        .clipShape(Capsule())
    }
}

/// Per-item outbox sync state chip.
struct OutboxStatusChip: View {
    @Environment(\.colorScheme) private var colorScheme
    let status: OutboxStatus

    private var tint: Color {
        switch status {
        case .queued: return LexturesTheme.amber
        case .syncing: return LexturesTheme.accent(for: colorScheme)
        case .synced: return LexturesTheme.brandTeal
        case .failed: return LexturesTheme.error
        case .conflict: return LexturesTheme.coral
        }
    }

    var body: some View {
        Text(status.userLabel)
            .font(.caption2.weight(.semibold))
            .foregroundStyle(tint)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(tint.opacity(0.12))
            .clipShape(Capsule())
    }
}

/// Global pending-changes affordance for the tab shell.
struct PendingSyncBadge: View {
    let count: Int

    var body: some View {
        if count > 0 {
            Text(count > 99 ? "99+" : "\(count)")
                .font(.caption2.weight(.bold))
                .foregroundStyle(.white)
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(LexturesTheme.amber)
                .clipShape(Capsule())
                .accessibilityLabel("\(count) pending changes")
        }
    }
}

/// Offline banner shown when the device has no connectivity.
struct OfflineBanner: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: "wifi.slash")
                .font(.caption.weight(.semibold))
            Text("You're offline — showing saved data")
                .font(.caption.weight(.medium))
        }
        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(LexturesTheme.amber.opacity(colorScheme == .dark ? 0.18 : 0.14))
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}
