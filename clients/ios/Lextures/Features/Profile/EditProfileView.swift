import SwiftUI

/// Navigation marker for the edit-profile screen.
struct EditProfileRoute: Hashable {}

/// Edits the student's server-backed profile: name, avatar URL, and phone.
/// Persists through PATCH `/api/v1/settings/account` (FR-1).
struct EditProfileView: View {
    @Environment(AuthSession.self) private var session
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme

    @State private var firstName = ""
    @State private var lastName = ""
    @State private var avatarUrl = ""
    @State private var phoneNumber = ""
    @State private var email = ""

    @State private var phase: Phase = .loading
    @State private var saving = false
    @State private var errorMessage: String?
    @State private var saved = false

    private enum Phase: Equatable {
        case loading
        case ready
        case failed
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            switch phase {
            case .loading:
                ProgressView()
                    .controlSize(.large)
            case .failed:
                loadFailed
            case .ready:
                form
            }
        }
        .navigationTitle(L.text("mobile.editProfile.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
    }

    private var loadFailed: some View {
        VStack(spacing: 12) {
            Image(systemName: "person.crop.circle.badge.exclamationmark")
                .font(.system(size: 40))
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Text(L.text("mobile.editProfile.loadError"))
                .font(.subheadline)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
            Button(L.text("mobile.common.retry")) {
                Task { await load() }
            }
            .font(.subheadline.weight(.semibold))
        }
        .padding(32)
    }

    private var form: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                avatarPreview

                LMSCard {
                    AuthTextField(
                        title: L.text("mobile.editProfile.firstName"),
                        text: $firstName,
                        autocapitalization: .words
                    )
                    AuthTextField(
                        title: L.text("mobile.editProfile.lastName"),
                        text: $lastName,
                        autocapitalization: .words
                    )
                }

                LMSCard {
                    AuthTextField(
                        title: L.text("mobile.editProfile.avatarUrl"),
                        text: $avatarUrl,
                        placeholder: "https://…",
                        keyboard: .URL
                    )
                    Text(L.text("mobile.editProfile.avatarHint"))
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                }

                LMSCard {
                    AuthTextField(
                        title: L.text("mobile.editProfile.phone"),
                        text: $phoneNumber,
                        keyboard: .phonePad,
                        textContentType: .telephoneNumber
                    )
                    Divider()
                    HStack(spacing: 12) {
                        Image(systemName: "envelope")
                            .font(.footnote)
                            .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                            .frame(width: 24)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(L.text("mobile.profile.email"))
                                .font(.caption)
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                            Text(email.isEmpty ? L.text("mobile.emDash") : email)
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        }
                        Spacer(minLength: 0)
                    }
                }

                if let errorMessage {
                    Text(errorMessage)
                        .font(.caption)
                        .foregroundStyle(LexturesTheme.error)
                }

                saveButton
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(16)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private var avatarPreview: some View {
        VStack(spacing: 10) {
            ProfileAvatarView(
                avatarUrl: avatarUrl,
                initials: initials,
                size: 84
            )
            Text(previewName)
                .font(LexturesTheme.displayFont(18))
                .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
        }
        .frame(maxWidth: .infinity)
        .padding(.top, 8)
    }

    private var saveButton: some View {
        Button {
            Task { await save() }
        } label: {
            Group {
                if saving {
                    ProgressView().tint(.white)
                } else {
                    Text(saved ? L.text("mobile.editProfile.saved") : L.text("mobile.common.save"))
                }
            }
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(.white)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(saved ? LexturesTheme.brandTeal : LexturesTheme.primaryDeep)
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        }
        .buttonStyle(.plain)
        .disabled(saving)
    }

    private var previewName: String {
        let name = "\(firstName) \(lastName)".trimmingCharacters(in: .whitespaces)
        return name.isEmpty ? (email.isEmpty ? L.text("mobile.profile.welcome") : email) : name
    }

    private var initials: String {
        let letters = "\(firstName) \(lastName)"
            .split(separator: " ")
            .compactMap(\.first)
        if letters.count >= 2 { return String([letters[0], letters[1]]).uppercased() }
        if let first = letters.first { return String(first).uppercased() }
        return String(email.prefix(1)).uppercased()
    }

    @MainActor
    private func load() async {
        phase = .loading
        guard let token = session.accessToken else {
            phase = .failed
            return
        }
        do {
            let profile = try await LMSAPI.fetchAccountProfile(accessToken: token)
            let names = profile.resolvedNameFields
            firstName = names.firstName
            lastName = names.lastName
            avatarUrl = profile.avatarUrl ?? ""
            phoneNumber = profile.phoneNumber ?? ""
            email = profile.email
            phase = .ready
        } catch {
            phase = .failed
        }
    }

    @MainActor
    private func save() async {
        guard let token = session.accessToken else { return }
        saving = true
        saved = false
        errorMessage = nil
        defer { saving = false }
        let patch = AccountProfilePatch(
            firstName: firstName.trimmingCharacters(in: .whitespacesAndNewlines),
            lastName: lastName.trimmingCharacters(in: .whitespacesAndNewlines),
            avatarUrl: avatarUrl.trimmingCharacters(in: .whitespacesAndNewlines),
            phoneNumber: phoneNumber.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        do {
            let updated = try await LMSAPI.updateAccountProfile(patch, accessToken: token)
            // Keep the rest of the app in sync with the new display name.
            await shell.refresh(accessToken: token)
            let names = updated.resolvedNameFields
            firstName = names.firstName
            lastName = names.lastName
            avatarUrl = updated.avatarUrl ?? ""
            phoneNumber = updated.phoneNumber ?? ""
            saved = true
        } catch let APIError.httpStatus(_, message) {
            errorMessage = message ?? L.text("mobile.editProfile.saveError")
        } catch {
            errorMessage = L.text("mobile.editProfile.saveError")
        }
    }
}
