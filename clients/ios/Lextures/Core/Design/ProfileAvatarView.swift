import SwiftUI

/// Circular profile image with initials fallback (http(s) or data:image URLs).
struct ProfileAvatarView: View {
    @Environment(\.colorScheme) private var colorScheme

    let avatarUrl: String?
    let initials: String
    var size: CGFloat = 84
    var initialsBackground: Color = LexturesTheme.brandTeal.opacity(0.16)
    var initialsForeground: Color? = nil

    var body: some View {
        Group {
            if let url = resolvedURL {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                    case .failure:
                        initialsCircle
                    case .empty:
                        ProgressView()
                            .controlSize(.small)
                    @unknown default:
                        initialsCircle
                    }
                }
            } else {
                initialsCircle
            }
        }
        .frame(width: size, height: size)
        .clipShape(Circle())
    }

    private var resolvedURL: URL? {
        guard let raw = avatarUrl?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            return nil
        }
        return URL(string: raw)
    }

    private var initialsCircle: some View {
        Circle()
            .fill(initialsBackground)
            .overlay(
                Text(initials)
                    .font(.system(size: size * 0.33, weight: .bold))
                    .foregroundStyle(initialsForeground ?? LexturesTheme.accent(for: colorScheme))
            )
    }
}
