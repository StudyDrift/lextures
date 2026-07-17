import SwiftUI

struct SplashView: View {
    @Environment(\.colorScheme) private var colorScheme
    @Environment(\.lxReduceMotion) private var reduceMotion
    @State private var appeared = false

    var body: some View {
        ZStack {
            PublicAuthBackground()

            VStack(spacing: 18) {
                BrandLogoView(maxHeight: 120)
                    .frame(maxWidth: 240)
                    .lxEnter(appeared)

                Text("Lextures")
                    .font(LexturesTheme.displayFont(30))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .opacity(appeared ? 1 : 0)
                    .animation(
                        LexturesMotion.resolve(
                            LexturesMotion.standard.delay(reduceMotion ? 0 : 0.2),
                            reduceMotion: reduceMotion
                        ),
                        value: appeared
                    )
            }
        }
        .onAppear {
            appeared = true
        }
    }
}
