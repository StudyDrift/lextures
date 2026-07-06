import Foundation

/// ePortfolio helpers (M12.1).
enum PortfolioLogic {
    static func portfolioEnabled(_ features: MobilePlatformFeatures) -> Bool {
        features.ffEportfolio
    }

    static func cacheKeyList() -> String { "portfolio:list" }

    static func cacheKeyDetail(portfolioId: String) -> String { "portfolio:\(portfolioId)" }

    static func artifactTypeLabel(_ type: String) -> String {
        switch type {
        case "submission": return L.text("mobile.portfolio.type.submission")
        case "upload": return L.text("mobile.portfolio.type.upload")
        case "text_page": return L.text("mobile.portfolio.type.textPage")
        case "url": return L.text("mobile.portfolio.type.url")
        case "heading": return L.text("mobile.portfolio.type.heading")
        default: return type.replacingOccurrences(of: "_", with: " ")
        }
    }

    static func isHeading(_ artifact: PortfolioArtifact) -> Bool {
        artifact.artifactType == "heading"
    }

    static func isContentPage(_ artifact: PortfolioArtifact) -> Bool {
        artifact.artifactType == "text_page"
    }

    static func hasFile(_ artifact: PortfolioArtifact) -> Bool {
        !artifact.fileName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || artifact.artifactType == "upload"
            || artifact.artifactType == "submission"
    }

    static func publicPortfolioURL(slug: String) -> URL {
        AppConfiguration.webURL(path: "/p/\(slug)")
    }

    static func shareText(title: String, url: URL) -> String {
        L.format("mobile.portfolio.shareText", title, url.absoluteString)
    }

    static func parseOutcomeIds(_ raw: String) -> [String] {
        raw.split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }

    static func orderedArtifacts(_ artifacts: [PortfolioArtifact], order: [String]) -> [PortfolioArtifact] {
        guard !order.isEmpty else {
            return artifacts.sorted { $0.sortOrder < $1.sortOrder }
        }
        let byId = Dictionary(uniqueKeysWithValues: artifacts.map { ($0.id, $0) })
        var out: [PortfolioArtifact] = []
        var seen = Set<String>()
        for id in order {
            if let art = byId[id] {
                out.append(art)
                seen.insert(id)
            }
        }
        for art in artifacts where !seen.contains(art.id) {
            out.append(art)
        }
        return out
    }
}