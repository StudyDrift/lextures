import Foundation

// MARK: - ePortfolio (M12.1)

struct PortfolioSummary: Codable, Identifiable, Hashable {
    var id: String
    var title: String
    var introText: String
    var isPublic: Bool
    var publicSlug: String?
    var order: [String]
    var createdAt: String
    var updatedAt: String
}

struct PortfolioArtifact: Codable, Identifiable, Hashable {
    var id: String
    var portfolioId: String
    var artifactType: String
    var title: String
    var description: String
    var sourceSubmissionId: String?
    var sourceCourseId: String?
    var fileName: String
    var fileMime: String
    var textContent: String
    var externalUrl: String
    var outcomeIds: [String]
    var isPublic: Bool
    var sortOrder: Int
    var createdAt: String
    var updatedAt: String
}

struct PortfolioDetailResponse: Codable {
    var portfolio: PortfolioSummary
    var artifacts: [PortfolioArtifact]
}

struct PortfoliosListResponse: Codable {
    var portfolios: [PortfolioSummary]?
}

struct CreatePortfolioRequest: Encodable {
    var title: String
    var introText: String
}

struct PatchPortfolioRequest: Encodable {
    var title: String?
    var introText: String?
    var isPublic: Bool?
    var order: [String]?
}

struct CreateArtifactRequest: Encodable {
    var artifactType: String
    var title: String
    var description: String?
    var sourceSubmissionId: String?
    var textContent: String?
    var externalUrl: String?
    var outcomeIds: [String]?
    var isPublic: Bool?
}

struct PatchArtifactRequest: Encodable {
    var title: String?
    var description: String?
    var textContent: String?
    var externalUrl: String?
    var outcomeIds: [String]?
    var isPublic: Bool?
}