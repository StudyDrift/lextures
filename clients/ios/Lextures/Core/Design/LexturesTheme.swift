import SwiftUI

/// Visual tokens aligned with the web auth shell (`lex-auth-scene`, teal primary).
enum LexturesTheme {
    static let primary = Color(red: 0.059, green: 0.463, blue: 0.431) // teal-700
    static let primaryMuted = Color(red: 0.075, green: 0.345, blue: 0.314) // teal-900 links
    static let sceneBackground = Color(red: 0.980, green: 0.980, blue: 0.976) // stone-50
    static let cardBackground = Color.white
    static let fieldBorder = Color(red: 0.898, green: 0.886, blue: 0.871) // stone-200
    static let textPrimary = Color(red: 0.110, green: 0.098, blue: 0.090) // stone-900
    static let textSecondary = Color(red: 0.420, green: 0.392, blue: 0.365) // stone-600
    static let error = Color(red: 0.878, green: 0.196, blue: 0.314) // rose-600

    static let sceneBackgroundDark = Color(red: 0.090, green: 0.090, blue: 0.090)
    static let cardBackgroundDark = Color(red: 0.090, green: 0.090, blue: 0.090)
    static let fieldBorderDark = Color(red: 0.310, green: 0.310, blue: 0.310)
    static let textPrimaryDark = Color(red: 0.980, green: 0.980, blue: 0.980)
    static let textSecondaryDark = Color(red: 0.639, green: 0.639, blue: 0.639)

    static func sceneBackground(for scheme: ColorScheme) -> Color {
        scheme == .dark ? sceneBackgroundDark : sceneBackground
    }

    static func cardBackground(for scheme: ColorScheme) -> Color {
        scheme == .dark ? cardBackgroundDark : cardBackground
    }

    static func fieldBorder(for scheme: ColorScheme) -> Color {
        scheme == .dark ? fieldBorderDark : fieldBorder
    }

    static func textPrimary(for scheme: ColorScheme) -> Color {
        scheme == .dark ? textPrimaryDark : textPrimary
    }

    static func textSecondary(for scheme: ColorScheme) -> Color {
        scheme == .dark ? textSecondaryDark : textSecondary
    }
}

struct PublicAuthBackground: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme)
                .ignoresSafeArea()

            RadialGradient(
                colors: [
                    LexturesTheme.primary.opacity(colorScheme == .dark ? 0.07 : 0.055),
                    .clear,
                ],
                center: .topTrailing,
                startRadius: 8,
                endRadius: 420
            )
            .ignoresSafeArea()

            RadialGradient(
                colors: [
                    Color.gray.opacity(colorScheme == .dark ? 0.04 : 0.05),
                    .clear,
                ],
                center: .bottomLeading,
                startRadius: 8,
                endRadius: 380
            )
            .ignoresSafeArea()
        }
    }
}

struct AuthCard<Content: View>: View {
    @Environment(\.colorScheme) private var colorScheme
    @ViewBuilder var content: () -> Content

    var body: some View {
        content()
            .padding(28)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.9), lineWidth: 1)
            )
            .shadow(color: .black.opacity(colorScheme == .dark ? 0 : 0.04), radius: 2, y: 1)
    }
}

struct AuthPrimaryButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.subheadline.weight(.semibold))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 11)
            .foregroundStyle(.white)
            .background(LexturesTheme.primary.opacity(isEnabled ? 1 : 0.6))
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .opacity(configuration.isPressed ? 0.88 : 1)
    }
}

struct AuthTextField: View {
    let title: String
    @Binding var text: String
    var placeholder: String = ""
    var isSecure = false
    var keyboard: UIKeyboardType = .default
    var textContentType: UITextContentType?
    var autocapitalization: TextInputAutocapitalization = .never

    @Environment(\.colorScheme) private var colorScheme
    @FocusState private var focused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            Group {
                if isSecure {
                    SecureField(placeholder, text: $text)
                } else {
                    TextField(placeholder, text: $text)
                        .keyboardType(keyboard)
                        .textInputAutocapitalization(autocapitalization)
                }
            }
            .textContentType(textContentType)
            .focused($focused)
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background(colorScheme == .dark ? Color(white: 0.05) : .white)
            .overlay(
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .stroke(
                        focused ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: focused ? 2 : 1
                    )
            )
            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
    }
}
