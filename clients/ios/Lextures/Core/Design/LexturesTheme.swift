import SwiftUI

/// Lextures by StudyDrift brand system.
///
/// Derived from the logo: a rocket lifting off an open book. Warm cream paper,
/// deep-teal ink, coral energy, amber highlights. Serif display type for an
/// editorial, scholarly feel; system sans for body copy.
enum LexturesTheme {
    // MARK: Brand anchors (from logo)
    static let brandTeal = Color(hex: 0x6EC0B1)   // logo teal — glows, tints
    static let brandCoral = Color(hex: 0xF6684B)  // logo coral — energy accent
    static let brandAmber = Color(hex: 0xF69945)  // logo amber — highlights
    static let brandCream = Color(hex: 0xF4E4C0)  // logo cream — paper tint

    // MARK: Action colors
    static let primary = Color(hex: 0x12756A)      // deep teal, contrast-safe
    static let primaryDeep = Color(hex: 0x0C4F47)  // gradient anchor
    static let primaryMuted = Color(hex: 0x135854) // links
    static let coral = brandCoral
    static let amber = brandAmber
    static let error = Color(hex: 0xDF3250)

    // MARK: Light surfaces
    static let sceneBackground = Color(hex: 0xFAF5EA)   // warm paper
    static let cardBackground = Color.white
    static let fieldBorder = Color(hex: 0xEAE0CC)
    static let textPrimary = Color(hex: 0x1F2D2A)       // teal-ink
    static let textSecondary = Color(hex: 0x64746F)

    // MARK: Dark surfaces (teal-tinted, never pure gray)
    static let sceneBackgroundDark = Color(hex: 0x111B19)
    static let cardBackgroundDark = Color(hex: 0x1B2725)
    static let fieldBorderDark = Color(hex: 0x32423E)
    static let textPrimaryDark = Color(hex: 0xF2EFE6)
    static let textSecondaryDark = Color(hex: 0x9CAEA8)

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

    /// Brighter primary for dark backgrounds.
    static func accent(for scheme: ColorScheme) -> Color {
        scheme == .dark ? brandTeal : primary
    }

    // MARK: Display typography (serif, editorial)
    static func displayFont(_ size: CGFloat, weight: Font.Weight = .semibold) -> Font {
        .system(size: size, weight: weight, design: .serif)
    }

    // MARK: Hero gradient (deep teal, dashboard greeting / course banners)
    static let heroGradient = LinearGradient(
        colors: [primaryDeep, Color(hex: 0x17897B)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Deterministic course-cover gradient: every course gets a stable brand color.
    static func coverGradient(for key: String) -> LinearGradient {
        let palettes: [(Color, Color)] = [
            (Color(hex: 0x17897B), Color(hex: 0x6EC0B1)), // teal
            (Color(hex: 0xE2553A), Color(hex: 0xF69945)), // coral → amber
            (Color(hex: 0x0C4F47), Color(hex: 0x2BA391)), // deep teal
            (Color(hex: 0xD9822B), Color(hex: 0xF6B95A)), // amber
            (Color(hex: 0xC65441), Color(hex: 0xF6684B)), // coral
        ]
        let index = abs(key.unicodeScalars.reduce(0) { ($0 &* 31) &+ Int($1.value) }) % palettes.count
        let pair = palettes[index]
        return LinearGradient(colors: [pair.0, pair.1], startPoint: .topLeading, endPoint: .bottomTrailing)
    }

    /// Soft elevation shadow for floating cards.
    static func cardShadow(for scheme: ColorScheme) -> Color {
        scheme == .dark ? .clear : Color(hex: 0x3A2E18).opacity(0.08)
    }
}

extension Color {
    init(hex: UInt32) {
        self.init(
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255
        )
    }
}

/// Warm paper backdrop with teal + coral glows (auth, splash).
struct PublicAuthBackground: View {
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme)
                .ignoresSafeArea()

            RadialGradient(
                colors: [
                    LexturesTheme.brandTeal.opacity(colorScheme == .dark ? 0.12 : 0.18),
                    .clear,
                ],
                center: .topTrailing,
                startRadius: 8,
                endRadius: 460
            )
            .ignoresSafeArea()

            RadialGradient(
                colors: [
                    LexturesTheme.brandCoral.opacity(colorScheme == .dark ? 0.07 : 0.10),
                    .clear,
                ],
                center: .bottomLeading,
                startRadius: 8,
                endRadius: 420
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
            .clipShape(RoundedRectangle(cornerRadius: 22, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(colorScheme == .dark ? 0.9 : 0.5), lineWidth: 1)
            )
            .shadow(color: LexturesTheme.cardShadow(for: colorScheme), radius: 18, y: 8)
    }
}

struct AuthPrimaryButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.subheadline.weight(.semibold))
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .foregroundStyle(.white)
            .background(
                LinearGradient(
                    colors: [LexturesTheme.primary, Color(hex: 0x17897B)],
                    startPoint: .leading,
                    endPoint: .trailing
                )
                .opacity(isEnabled ? 1 : 0.55)
            )
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .shadow(
                color: LexturesTheme.primary.opacity(isEnabled ? 0.35 : 0),
                radius: configuration.isPressed ? 4 : 10,
                y: configuration.isPressed ? 2 : 5
            )
            .scaleEffect(configuration.isPressed ? 0.98 : 1)
            .animation(.easeOut(duration: 0.15), value: configuration.isPressed)
    }
}

struct AuthTextField: View {
    let title: String
    @Binding var text: String
    var placeholder: String = ""
    var isSecure = false
    var keyboard: UIKeyboardType = .default
    var textContentType: UITextContentType?
    var autocapitalization: UITextAutocapitalizationType = .none

    @Environment(\.colorScheme) private var colorScheme
    @FocusState private var focused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

            AuthFieldRepresentable(
                text: $text,
                placeholder: placeholder,
                isSecure: isSecure,
                keyboard: keyboard,
                textContentType: textContentType,
                autocapitalization: isSecure ? .none : autocapitalization
            )
            .focused($focused)
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
            .background(colorScheme == .dark ? Color(hex: 0x141F1D) : LexturesTheme.sceneBackground.opacity(0.6))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(
                        focused ? LexturesTheme.primary : LexturesTheme.fieldBorder(for: colorScheme),
                        lineWidth: focused ? 2 : 1
                    )
            )
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            .animation(.easeOut(duration: 0.15), value: focused)
        }
    }
}
