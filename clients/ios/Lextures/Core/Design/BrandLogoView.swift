import SwiftUI

/// Official Lextures mark (`logo-trimmed.svg`). Uses original rendering so system tint does not flatten it to a solid block.
struct BrandLogoView: View {
    var maxHeight: CGFloat = 56

    var body: some View {
        Image("Logo")
            .resizable()
            .renderingMode(.original)
            .scaledToFit()
            .frame(maxHeight: maxHeight)
            .accessibilityLabel("Lextures")
    }
}
