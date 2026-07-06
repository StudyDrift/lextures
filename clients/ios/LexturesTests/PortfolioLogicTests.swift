import XCTest
@testable import Lextures

final class PortfolioLogicTests: XCTestCase {
    func testPortfolioEnabled() {
        var features = MobilePlatformFeatures()
        XCTAssertFalse(PortfolioLogic.portfolioEnabled(features))
        features.ffEportfolio = true
        XCTAssertTrue(PortfolioLogic.portfolioEnabled(features))
    }

    func testArtifactTypeLabel() {
        XCTAssertEqual(PortfolioLogic.artifactTypeLabel("upload"), L.text("mobile.portfolio.type.upload"))
        XCTAssertEqual(PortfolioLogic.artifactTypeLabel("heading"), L.text("mobile.portfolio.type.heading"))
    }

    func testOrderedArtifacts() {
        let first = PortfolioArtifact(
            id: "artifact-a", portfolioId: "portfolio-1", artifactType: "upload", title: "First", description: "",
            sourceSubmissionId: nil, sourceCourseId: nil, fileName: "", fileMime: "",
            textContent: "", externalUrl: "", outcomeIds: [], isPublic: false, sortOrder: 1,
            createdAt: "", updatedAt: ""
        )
        let second = PortfolioArtifact(
            id: "artifact-b", portfolioId: "portfolio-1", artifactType: "url", title: "Second", description: "",
            sourceSubmissionId: nil, sourceCourseId: nil, fileName: "", fileMime: "",
            textContent: "", externalUrl: "https://x", outcomeIds: [], isPublic: false, sortOrder: 0,
            createdAt: "", updatedAt: ""
        )
        let ordered = PortfolioLogic.orderedArtifacts([first, second], order: ["artifact-b", "artifact-a"])
        XCTAssertEqual(ordered.map(\.id), ["artifact-b", "artifact-a"])
    }

    func testParseOutcomeIds() {
        XCTAssertEqual(PortfolioLogic.parseOutcomeIds("a, b , c"), ["a", "b", "c"])
    }
}
