import SwiftUI

struct SplashView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var logoScale: CGFloat = 0.96
    @State private var titleOpacity: Double = 0

    var body: some View {
        ZStack {
            PublicAuthBackground()

            VStack(spacing: 16) {
                BrandLogoView(maxHeight: 120)
                    .frame(maxWidth: 240)
                    .scaleEffect(logoScale)

                Text("Lextures")
                    .font(.system(.title2, design: .serif).weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .opacity(titleOpacity)
            }
        }
        .onAppear {
            withAnimation(.easeOut(duration: 0.45)) {
                logoScale = 1
                titleOpacity = 1
            }
        }
    }
}
