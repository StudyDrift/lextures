import XCTest
@testable import Lextures

final class CoursePlagiarismLogicTests: XCTestCase {
    func testDraftUsesDefaultsWhenSettingsMissing() {
        let draft = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
        XCTAssertTrue(draft.checksEnabled)
        XCTAssertEqual(draft.provider, "")
        XCTAssertEqual(draft.thresholdPct, "40")
    }

    func testDraftMapsSettings() {
        let settings = CoursePlagiarismSettings(
            plagiarismChecksEnabled: false,
            plagiarismProvider: "turnitin",
            plagiarismAlertThresholdPct: 55
        )
        let draft = CoursePlagiarismLogic.draft(from: settings)
        XCTAssertFalse(draft.checksEnabled)
        XCTAssertEqual(draft.provider, "turnitin")
        XCTAssertEqual(draft.thresholdPct, "55")
    }

    func testIsDirtyDetectsProviderChange() {
        let baseline = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
        var current = baseline
        XCTAssertFalse(CoursePlagiarismLogic.isDirty(current: current, baseline: baseline))
        current.provider = "copyleaks"
        XCTAssertTrue(CoursePlagiarismLogic.isDirty(current: current, baseline: baseline))
    }

    func testValidateDraftRejectsInvalidThreshold() {
        var draft = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
        draft.thresholdPct = "150"
        XCTAssertEqual(CoursePlagiarismLogic.validateDraft(draft), .thresholdInvalid)
        draft.thresholdPct = "40"
        XCTAssertNil(CoursePlagiarismLogic.validateDraft(draft))
    }

    func testBuildPatchBodyUsesNullProviderForDefault() {
        var draft = CoursePlagiarismLogic.draft(from: nil as CoursePlagiarismSettings?)
        draft.provider = ""
        draft.thresholdPct = "25"
        let body = CoursePlagiarismLogic.buildPatchBody(current: draft)
        XCTAssertNil(body.plagiarismProvider)
        XCTAssertEqual(body.plagiarismAlertThresholdPct, 25)
    }

    func testNormalizedProviderRejectsUnknown() {
        XCTAssertEqual(CoursePlagiarismLogic.normalizedProvider("proprietary"), "")
        XCTAssertEqual(CoursePlagiarismLogic.normalizedProvider("Turnitin"), "turnitin")
    }
}
