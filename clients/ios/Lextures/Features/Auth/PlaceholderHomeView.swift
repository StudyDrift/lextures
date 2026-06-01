import SwiftUI

/// Temporary post-auth shell until LMS features ship on iOS.
struct PlaceholderHomeView: View {
    @Environment(AuthSession.self) private var session
    @Environment(\.colorScheme) private var colorScheme

    var body: some View {
        NavigationStack {
            ZStack {
                PublicAuthBackground()

                VStack(spacing: 20) {
                    BrandLogoView(maxHeight: 72)

                    Text("You're signed in")
                        .font(.title2.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))

                    if let email = session.userEmail {
                        Text(email)
                            .font(.subheadline)
                            .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                    }

                    Text("Course and dashboard features are coming soon to the iOS app.")
                        .font(.subheadline)
                        .multilineTextAlignment(.center)
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                        .padding(.horizontal)

                    Button("Sign out") {
                        session.signOut()
                    }
                    .buttonStyle(AuthPrimaryButtonStyle())
                    .padding(.horizontal, 40)
                    .padding(.top, 8)
                }
                .padding()
            }
            .navigationTitle("Lextures")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}
