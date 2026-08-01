import SwiftUI

/// Persistent native chrome for the in-app browser (FR-11–FR-15).
struct BrowserChromeHeader: View {
    let host: String
    let title: String
    let isSecure: Bool
    let progress: Double
    let canGoBack: Bool
    var onClose: () -> Void
    var onBack: () -> Void
    var onCopy: () -> Void
    var onOverflow: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                if canGoBack {
                    Button(action: onBack) {
                        Image(systemName: "chevron.backward")
                            .font(.body.weight(.semibold))
                    }
                    .accessibilityLabel(L.text("mobile.browser.back"))
                }

                Button(action: onClose) {
                    Image(systemName: "xmark")
                        .font(.body.weight(.semibold))
                        .frame(minWidth: 28, minHeight: 28)
                }
                .accessibilityLabel(L.text("mobile.browser.close"))

                Button(action: onCopy) {
                    HStack(spacing: 4) {
                        Image(systemName: isSecure ? "lock.fill" : "exclamationmark.triangle.fill")
                            .font(.caption2)
                            .foregroundStyle(isSecure ? .secondary : Color.orange)
                        Text(isSecure ? host : L.text("mobile.browser.notSecure"))
                            .font(.subheadline.weight(.semibold))
                            .lineLimit(1)
                    }
                    .padding(.horizontal, 10)
                    .padding(.vertical, 6)
                    .background(Color.primary.opacity(0.06), in: Capsule())
                }
                .accessibilityLabel(L.text("mobile.browser.copyLink") + ", " + host)
                .accessibilityHint(L.text("mobile.browser.linkCopied"))

                Spacer(minLength: 0)

                Button(action: onOverflow) {
                    Image(systemName: "ellipsis")
                        .font(.body.weight(.semibold))
                        .frame(minWidth: 28, minHeight: 28)
                }
                .accessibilityLabel(L.text("mobile.browser.overflowTitle"))
            }
            .padding(.horizontal, 12)
            .padding(.top, 10)
            .padding(.bottom, 4)

            if !title.isEmpty {
                Text(title)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal, 16)
                    .padding(.bottom, 6)
            }

            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Rectangle().fill(Color.clear)
                    if progress > 0 && progress < 1 {
                        Rectangle()
                            .fill(Color.accentColor)
                            .frame(width: geo.size.width * progress)
                            .accessibilityValue("\(Int(progress * 100))%")
                    }
                }
            }
            .frame(height: 2)
        }
        .background(.bar)
    }
}
