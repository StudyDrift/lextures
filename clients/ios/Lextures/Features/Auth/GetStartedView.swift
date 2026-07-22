import SwiftUI

/// First auth screen: choose homeschool vs school (parity with lextures.com/get-started).
struct GetStartedView: View {
    var onComplete: () -> Void

    @Environment(\.colorScheme) private var colorScheme
    @State private var step: Step = .choose
    @State private var schoolCode = ""

    private enum Step: Equatable {
        case choose
        case schoolCode
    }

    var body: some View {
        ScrollView {
            VStack(spacing: 0) {
                switch step {
                case .choose:
                    chooseStep
                        .transition(.move(edge: .leading).combined(with: .opacity))
                case .schoolCode:
                    schoolCodeStep
                        .transition(.move(edge: .trailing).combined(with: .opacity))
                }
            }
            .padding(.horizontal, 20)
            .padding(.vertical, 24)
            .frame(maxWidth: 520)
            .frame(maxWidth: .infinity)
        }
        .scrollDismissesKeyboard(.automatic)
        .animation(.easeInOut(duration: 0.2), value: step)
    }

    // MARK: - Choose

    private var chooseStep: some View {
        VStack(spacing: 0) {
            BrandLogoView(maxHeight: 56)
                .accessibilityHidden(true)
                .padding(.bottom, 28)

            VStack(spacing: 8) {
                Text(L.text("auth.getStarted.title"))
                    .font(.system(.title, design: .serif).weight(.semibold))
                    .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    .multilineTextAlignment(.center)
                    .accessibilityAddTraits(.isHeader)

                Text(L.text("auth.getStarted.subtitle"))
                    .font(.subheadline)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    .multilineTextAlignment(.center)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .padding(.bottom, 28)

            VStack(spacing: 12) {
                pathCard(
                    systemImage: "house.fill",
                    title: L.text("auth.getStarted.homeschoolTitle"),
                    description: L.text("auth.getStarted.homeschoolDescription")
                ) {
                    EnvironmentStore.shared.selectHomeschool()
                    onComplete()
                }

                pathCard(
                    systemImage: "graduationcap.fill",
                    title: L.text("auth.getStarted.schoolTitle"),
                    description: L.text("auth.getStarted.schoolDescription")
                ) {
                    withAnimation(.easeInOut(duration: 0.2)) {
                        step = .schoolCode
                    }
                }
            }
        }
    }

    private func pathCard(
        systemImage: String,
        title: String,
        description: String,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            HStack(alignment: .top, spacing: 14) {
                Image(systemName: systemImage)
                    .font(.title3.weight(.semibold))
                    .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    .frame(width: 44, height: 44)
                    .background(
                        LexturesTheme.accent(for: colorScheme).opacity(colorScheme == .dark ? 0.18 : 0.12)
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                    Text(description)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                        .fixedSize(horizontal: false, vertical: true)
                }
                Spacer(minLength: 0)
            }
            .padding(18)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LexturesTheme.cardBackground(for: colorScheme))
            .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 18, style: .continuous)
                    .stroke(LexturesTheme.fieldBorder(for: colorScheme).opacity(0.7), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .accessibilityElement(children: .combine)
    }

    // MARK: - School code

    private var schoolCodeStep: some View {
        let errorKey = schoolCode.isEmpty ? nil : SchoolCodeLogic.errorKey(for: schoolCode)
        let preview = SchoolCodeLogic.previewHost(schoolCode: schoolCode)

        return VStack(alignment: .leading, spacing: 0) {
            Button {
                withAnimation(.easeInOut(duration: 0.2)) {
                    step = .choose
                    schoolCode = ""
                }
            } label: {
                Label(L.text("auth.getStarted.back"), systemImage: "chevron.left")
                    .font(.subheadline.weight(.medium))
            }
            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
            .padding(.bottom, 24)

            Text(L.text("auth.getStarted.schoolCodeTitle"))
                .font(.system(.title, design: .serif).weight(.semibold))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                .accessibilityAddTraits(.isHeader)
                .padding(.bottom, 8)

            Text(L.text("auth.getStarted.schoolCodeSubtitle"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                .fixedSize(horizontal: false, vertical: true)
                .padding(.bottom, 24)

            AuthCard {
                VStack(alignment: .leading, spacing: 16) {
                    AuthTextField(
                        title: L.text("auth.getStarted.schoolCodeLabel"),
                        text: $schoolCode,
                        placeholder: L.text("auth.getStarted.schoolCodePlaceholder"),
                        keyboard: .asciiCapable,
                        autocapitalization: .none
                    )

                    Text(L.text("auth.getStarted.schoolCodeHelp"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))

                    if let errorKey {
                        Text(L.dynamicText(errorKey))
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.error)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .accessibilityLabel(L.dynamicText(errorKey))
                    }

                    Text(L.format("auth.getStarted.schoolCodePreview", preview))
                        .font(.subheadline)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(
                            LexturesTheme.sceneBackground(for: colorScheme).opacity(0.7)
                        )
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))

                    Button(L.text("auth.getStarted.continue")) {
                        guard SchoolCodeLogic.isValid(schoolCode) else { return }
                        EnvironmentStore.shared.selectSchool(code: schoolCode)
                        onComplete()
                    }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    .disabled(!SchoolCodeLogic.isValid(schoolCode))
                }
            }
        }
    }
}
