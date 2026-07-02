import SwiftUI

struct SplashView: View {
    @Environment(\.colorScheme) private var colorScheme
    @State private var logoScale: CGFloat = 0.94
    @State private var logoOffset: CGFloat = 14
    @State private var titleOpacity: Double = 0

    var body: some View {
        ZStack {
            PublicAuthBackground()

            VStack(spacing: 18) {
                BrandLogoView(maxHeight: 120)
                    .frame(maxWidth: 240)
                    .scaleEffect(logoScale)
                    .offset(y: logoOffset)

                Text("Lextures")
                    .font(LexturesTheme.displayFont(30))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .opacity(titleOpacity)
            }
        }
        .onAppear {
            // Logo drifts upward on launch — a nod to the rocket in the mark.
            withAnimation(.easeOut(duration: 0.6)) {
                logoScale = 1
                logoOffset = 0
            }
            withAnimation(.easeOut(duration: 0.5).delay(0.2)) {
                titleOpacity = 1
            }
        }
    }
}
