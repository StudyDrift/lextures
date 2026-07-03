import SwiftUI

/// Earned completion certificates and open badges (M9.3).
struct CredentialsView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.colorScheme) private var colorScheme

    @State private var credentials: [IssuedCredentialSummary] = []
    @State private var loading = true
    @State private var errorMessage: String?
    @State private var cacheLabel: String?
    @State private var selectedCredential: IssuedCredentialSummary?

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if loading {
                LMSSkeletonList(count: 3)
            } else if let errorMessage, credentials.isEmpty {
                LMSEmptyState(
                    systemImage: "rosette",
                    title: L.text("mobile.credentials.errorTitle"),
                    message: errorMessage
                )
            } else if credentials.isEmpty {
                LMSEmptyState(
                    systemImage: "rosette",
                    title: L.text("mobile.credentials.emptyTitle"),
                    message: L.text("mobile.credentials.emptyMessage")
                )
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: 12) {
                        if let cacheLabel {
                            StalenessChip(label: cacheLabel)
                        }
                        ForEach(credentials) { credential in
                            Button {
                                selectedCredential = credential
                            } label: {
                                credentialRow(credential)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(16)
                }
            }
        }
        .navigationTitle(L.text("mobile.credentials.title"))
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
        .navigationDestination(item: $selectedCredential) { credential in
            CredentialDetailView(credential: credential)
        }
    }

    @ViewBuilder
    private func credentialRow(_ credential: IssuedCredentialSummary) -> some View {
        LMSCard {
            VStack(alignment: .leading, spacing: 6) {
                HStack {
                    Image(systemName: "rosette")
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    Text(credential.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                    Spacer(minLength: 0)
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme).opacity(0.6))
                }
                Text(CredentialsLogic.sourceTypeLabel(credential.sourceType))
                    .font(.caption)
                    .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                Text(L.format(
                    "mobile.credentials.issued",
                    CredentialsLogic.issuedDateLabel(iso: credential.issuedAt)
                ))
                .font(.caption)
                .foregroundStyle(LexturesTheme.textSecondary(for: colorScheme))
                if credential.revoked {
                    Text(L.text("mobile.credentials.revoked"))
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(LexturesTheme.coral)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .accessibilityElement(children: .combine)
            .accessibilityLabel(L.format(
                "mobile.credentials.rowAccessibility",
                credential.title,
                CredentialsLogic.issuedDateLabel(iso: credential.issuedAt)
            ))
        }
    }

    private func load() async {
        guard let token = session.accessToken else { return }
        loading = credentials.isEmpty
        errorMessage = nil
        defer { loading = false }

        do {
            let result = try await offline.cachedFetch(
                key: OfflineCacheKey.credentialsList(),
                accessToken: token
            ) {
                try await LMSAPI.fetchMyCredentials(accessToken: token)
            }
            credentials = result.value
            if let cached = result.cached, cached.isStale(isOnline: NetworkMonitor.shared.isOnline) {
                cacheLabel = cached.lastUpdatedLabel
            } else {
                cacheLabel = nil
            }
        } catch {
            errorMessage = L.text("mobile.credentials.loadError")
        }
    }
}