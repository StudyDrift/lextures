import SwiftUI

struct BrowserErrorState: View {
    enum Kind { case error, offline, blocked }

    let kind: Kind
    let message: String
    var onRetry: (() -> Void)?
    var onOpenExternal: (() -> Void)?

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundStyle(.secondary)
            Text(title)
                .font(.title3.weight(.semibold))
                .multilineTextAlignment(.center)
            Text(message)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)
            if let onRetry {
                Button(L.text("mobile.browser.retry"), action: onRetry)
                    .buttonStyle(.borderedProminent)
            }
            if let onOpenExternal {
                Button(L.text("mobile.browser.openInBrowser"), action: onOpenExternal)
                    .buttonStyle(.bordered)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding()
    }

    private var icon: String {
        switch kind {
        case .error: return "exclamationmark.triangle"
        case .offline: return "wifi.slash"
        case .blocked: return "hand.raised"
        }
    }

    private var title: String {
        switch kind {
        case .error: return L.text("mobile.browser.errorTitle")
        case .offline: return L.text("mobile.browser.offlineTitle")
        case .blocked: return L.text("mobile.browser.blockedByPolicy")
        }
    }
}
