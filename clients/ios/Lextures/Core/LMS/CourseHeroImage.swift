import SwiftUI

/// Course banner image with auth for `/course-files/.../content` URLs (parity with web).
struct CourseHeroImage: View {
    @Environment(AuthSession.self) private var session
    let urlString: String?
    let fallbackKey: String
    var height: CGFloat = 84

    @State private var image: UIImage?

    private static let cache = NSCache<NSString, UIImage>()

    var body: some View {
        ZStack {
            if let image {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
            } else {
                LexturesTheme.coverGradient(for: fallbackKey)
            }
        }
        .frame(height: height)
        .frame(maxWidth: .infinity)
        .clipped()
        .accessibilityHidden(true)
        .task(id: urlString) { await load() }
    }

    private func load() async {
        guard let raw = urlString?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
            image = nil
            return
        }
        if let cached = Self.cache.object(forKey: raw as NSString) {
            image = cached
            return
        }
        guard let url = resolvedURL(raw) else { return }
        var request = URLRequest(url: url)
        if let token = session.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        guard
            let (data, response) = try? await URLSession.shared.data(for: request),
            (response as? HTTPURLResponse).map({ (200 ... 299).contains($0.statusCode) }) != false,
            let loaded = UIImage(data: data)
        else { return }
        Self.cache.setObject(loaded, forKey: raw as NSString)
        image = loaded
    }

    private func resolvedURL(_ raw: String) -> URL? {
        if raw.hasPrefix("/") {
            return AppConfiguration.apiURL(path: raw)
        }
        if let parsed = URL(string: raw), parsed.scheme == "https" || parsed.scheme == "http" {
            return parsed
        }
        return nil
    }
}
