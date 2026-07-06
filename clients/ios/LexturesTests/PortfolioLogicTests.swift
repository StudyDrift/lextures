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
        let a = PortfolioArtifact(
            id: "a", portfolioId: "p", artifactType: "upload", title: "A", description: "",
            sourceSubmissionId: nil, sourceCourseId: nil, fileName: "", fileMime: "",
            textContent: "", externalUrl: "", outcomeIds: [], isPublic: false, sortOrder: 1,
            createdAt: "", updatedAt: ""
        )
        let b = PortfolioArtifact(
            id: "b", portfolioId: "p", artifactType: "url", title: "B", description: "",
            sourceSubmissionId: nil, sourceCourseId: nil, fileName: "", fileMime: "",
            textContent: "", externalUrl: "https://x", outcomeIds: [], isPublic: false, sortOrder: 0,
            createdAt: "", updatedAt: ""
        )
        let ordered = PortfolioLogic.orderedArtifacts([a, b], order: ["b", "a"])
        XCTAssertEqual(ordered.map(\.id), ["b", "a"])
    }

    func testParseOutcomeIds() {
        XCTAssertEqual(PortfolioLogic.parseOutcomeIds("a, b , c"), ["a", "b", "c"])
    }
}