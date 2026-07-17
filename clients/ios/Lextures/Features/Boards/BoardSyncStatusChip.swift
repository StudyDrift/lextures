import SwiftUI

/// Subtle Live / Reconnecting / Offline chip for the board header (VC.M4 FR-8).
struct BoardSyncStatusChip: View {
    let state: BoardSyncState
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        HStack(spacing: 6) {
            if state == .live {
                Circle()
                    .fill(Color.green.opacity(0.85))
                    .frame(width: 6, height: 6)
                    .accessibilityHidden(true)
            }
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(foreground)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel(label)
    }

    private var label: String {
        switch state {
        case .connecting: return L.text("mobile.boards.sync.connecting")
        case .live: return L.text("mobile.boards.sync.live")
        case .reconnecting: return L.text("mobile.boards.sync.reconnecting")
        case .offline: return L.text("mobile.boards.sync.offline")
        }
    }

    private var foreground: Color {
        switch state {
        case .live:
            return Color.green.opacity(colorScheme == .dark ? 0.9 : 0.8)
        case .reconnecting:
            return Color.orange.opacity(0.9)
        case .connecting, .offline:
            return LexturesTheme.textSecondary(for: colorScheme)
        }
    }
}
