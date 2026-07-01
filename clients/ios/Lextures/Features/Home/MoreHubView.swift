import SwiftUI

struct MoreHubRoute: Hashable {}

/// Searchable grid of secondary destinations gated by role and platform flags.
struct MoreHubView: View {
    @Environment(AppShellModel.self) private var shell
    @Environment(\.colorScheme) private var colorScheme
    @State private var query = ""

    private var destinations: [MoreDestination] {
        MobileDestinations.moreDestinations(
            context: shell.activeRoleContext,
            platform: shell.platformFeatures
        )
    }

    private var filtered: [MoreDestination] {
        let q = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return destinations }
        return destinations.filter { $0.label.lowercased().contains(q) }
    }

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()

            if destinations.isEmpty {
                LMSEmptyState(
                    systemImage: "square.grid.2x2",
                    title: L.text("mobile.ia.more.emptyTitle"),
                    message: L.text("mobile.ia.more.emptyMessage")
                )
            } else {
                ScrollView {
                    LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                        ForEach(filtered) { destination in
                            MoreDestinationTile(destination: destination)
                        }
                    }
                    .padding(16)
                }
                .searchable(text: $query, prompt: L.text("mobile.ia.more.search"))
            }
        }
        .navigationTitle(L.text("mobile.ia.more.title"))
        .navigationBarTitleDisplayMode(.inline)
    }
}

private struct MoreDestinationTile: View {
    @Environment(\.colorScheme) private var colorScheme
    let destination: MoreDestination

    var body: some View {
        NavigationLink(value: destination) {
            LMSCard {
                VStack(alignment: .leading, spacing: 10) {
                    Image(systemName: destination.systemImage)
                        .font(.title3.weight(.semibold))
                        .foregroundStyle(LexturesTheme.accent(for: colorScheme))
                    Text(destination.label)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(LexturesTheme.textPrimary(for: colorScheme))
                        .multilineTextAlignment(.leading)
                }
                .frame(maxWidth: .infinity, minHeight: 88, alignment: .leading)
            }
        }
        .buttonStyle(.plain)
    }
}

/// Placeholder surface for More destinations until their stories land.
struct MoreDestinationPlaceholder: View {
    @Environment(\.colorScheme) private var colorScheme
    let destination: MoreDestination

    var body: some View {
        ZStack {
            LexturesTheme.sceneBackground(for: colorScheme).ignoresSafeArea()
            LMSEmptyState(
                systemImage: destination.systemImage,
                title: destination.label,
                message: L.text("mobile.ia.placeholder.message")
            )
        }
        .navigationTitle(destination.label)
        .navigationBarTitleDisplayMode(.inline)
    }
}