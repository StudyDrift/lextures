import XCTest
@testable import Lextures

final class AiModelsAdminLogicTests: XCTestCase {
    func testEntryRequiresFlagAndRbacPermission() {
        let off = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: false)
        XCTAssertFalse(AiModelsAdminLogic.shouldShowEntry(
            features: off,
            permissions: [AiModelsAdminLogic.rbacManagePermission]
        ))
        let on = MobilePlatformFeatures(ffMobileAdminConsole: false, ffMobileAdminSettings: true)
        XCTAssertFalse(AiModelsAdminLogic.shouldShowEntry(features: on, permissions: []))
        XCTAssertTrue(AiModelsAdminLogic.shouldShowEntry(
            features: on,
            permissions: [AiModelsAdminLogic.rbacManagePermission]
        ))
    }

    func testBuildSaveRequestOmitsUnchangedPlaceholderKey() {
        let request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId: "img-1",
            courseSetupModelId: "text-1",
            notebookFlashcardsModelId: "text-2",
            vibeActivityModelId: "text-3",
            graderAgentModelId: "text-4",
            openRouterApiKey: AiModelsAdminLogic.platformSecretPlaceholder,
            openRouterApiKeyBaseline: AiModelsAdminLogic.platformSecretPlaceholder
        )
        XCTAssertNil(request.openRouterApiKey)
        XCTAssertNil(request.clearOpenRouterApiKey)
        XCTAssertEqual(request.imageModelId, "img-1")
        XCTAssertEqual(request.courseSetupModelId, "text-1")
    }

    func testBuildSaveRequestSendsNewKey() {
        let request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId: "img-1",
            courseSetupModelId: "text-1",
            notebookFlashcardsModelId: "text-2",
            vibeActivityModelId: "text-3",
            graderAgentModelId: "text-4",
            openRouterApiKey: "sk-new-secret",
            openRouterApiKeyBaseline: AiModelsAdminLogic.platformSecretPlaceholder
        )
        XCTAssertEqual(request.openRouterApiKey, "sk-new-secret")
        XCTAssertNil(request.clearOpenRouterApiKey)
        XCTAssertTrue(AiModelsAdminLogic.shouldSendOpenRouterKey("sk-new-secret"))
        XCTAssertFalse(AiModelsAdminLogic.shouldSendOpenRouterKey(AiModelsAdminLogic.platformSecretPlaceholder))
    }

    func testBuildSaveRequestClearsKeyWhenEmptiedFromPlaceholder() {
        let request = AiModelsAdminLogic.buildAiSettingsSaveRequest(
            imageModelId: "img-1",
            courseSetupModelId: "text-1",
            notebookFlashcardsModelId: "text-2",
            vibeActivityModelId: "text-3",
            graderAgentModelId: "text-4",
            openRouterApiKey: "",
            openRouterApiKeyBaseline: AiModelsAdminLogic.platformSecretPlaceholder
        )
        XCTAssertNil(request.openRouterApiKey)
        XCTAssertEqual(request.clearOpenRouterApiKey, true)
        XCTAssertTrue(AiModelsAdminLogic.shouldClearOpenRouterKey(
            draft: "",
            baseline: AiModelsAdminLogic.platformSecretPlaceholder
        ))
    }

    func testSaveDisabledWhenRequiredModelsMissing() {
        XCTAssertTrue(AiModelsAdminLogic.isSaveDisabled(
            saving: false,
            imageModelId: "",
            courseSetupModelId: "a",
            notebookFlashcardsModelId: "b",
            vibeActivityModelId: "c"
        ))
        XCTAssertFalse(AiModelsAdminLogic.isSaveDisabled(
            saving: false,
            imageModelId: "img",
            courseSetupModelId: "a",
            notebookFlashcardsModelId: "b",
            vibeActivityModelId: "c"
        ))
        XCTAssertTrue(AiModelsAdminLogic.isSaveDisabled(
            saving: true,
            imageModelId: "img",
            courseSetupModelId: "a",
            notebookFlashcardsModelId: "b",
            vibeActivityModelId: "c"
        ))
    }

    func testModelsWithSelectionInjectsMissingId() {
        let models = [AiModelOption(id: "a", name: "A")]
        let merged = AiModelsAdminLogic.modelsWithSelection(models, selectedId: "legacy")
        XCTAssertEqual(merged.first?.id, "legacy")
        XCTAssertEqual(merged.count, 2)
        XCTAssertEqual(
            AiModelsAdminLogic.modelsWithSelection(models, selectedId: "a").count,
            1
        )
    }

    func testReportRangeAndFormatting() {
        let now = Date(timeIntervalSince1970: 1_700_000_000)
        let range = AiModelsAdminLogic.utcRange(for: .hours24, now: now)
        XCTAssertFalse(range.from.isEmpty)
        XCTAssertFalse(range.to.isEmpty)
        XCTAssertEqual(AiModelsAdminLogic.formatUsd(0), "$0.00")
        XCTAssertEqual(AiModelsAdminLogic.formatUsd(0.0012), "$0.0012")
        XCTAssertEqual(AiModelsAdminLogic.formatUsd(1.5), "$1.50")
        XCTAssertEqual(AiModelsAdminLogic.featureLabel("ai_tutor"), "AI Tutor")
        XCTAssertEqual(AiModelsAdminLogic.featureLabel("custom_thing"), "custom thing")
        XCTAssertTrue(AiModelsAdminLogic.promptContentChanged(original: "a", draft: "b"))
        XCTAssertFalse(AiModelsAdminLogic.promptContentChanged(original: "a", draft: "a"))
    }

    func testDecodeAiSettingsAndReports() throws {
        let settingsData = Data(#"""
        {
          "imageModelId":"img","courseSetupModelId":"cs",
          "notebookFlashcardsModelId":"fc","vibeActivityModelId":"va",
          "graderAgentModelId":"ga","openRouterApiKey":"••••••••••••"
        }
        """#.utf8)
        let settings = try JSONDecoder().decode(AiSettingsResponse.self, from: settingsData)
        XCTAssertEqual(settings.imageModelId, "img")
        XCTAssertEqual(settings.openRouterApiKey, AiModelsAdminLogic.platformSecretPlaceholder)

        let reportsData = Data(#"""
        {
          "range":{"from":"2024-01-01T00:00:00Z","to":"2024-01-02T00:00:00Z"},
          "cost":{
            "summary":{"totalCostUsd":1.25,"totalCalls":3,"totalTokens":100},
            "byDay":[{"day":"2024-01-01","costUsd":1.25,"calls":3,"tokens":100}],
            "byFeature":[{"feature":"ai_tutor","costUsd":1.25,"calls":3,"tokens":100}]
          },
          "byUser":[{"userId":"u1","email":"a@b.c","displayName":"Ada","calls":1,"promptTokens":1,"completionTokens":1,"totalTokens":2,"costUsd":0.5}],
          "byCourse":[{"courseId":"c1","courseCode":"CS101","title":"Intro","calls":1,"totalTokens":2,"costUsd":0.5}]
        }
        """#.utf8)
        let report = try JSONDecoder().decode(AiReportsPayload.self, from: reportsData)
        XCTAssertEqual(report.cost.summary.totalCalls, 3)
        XCTAssertEqual(report.byUser.first?.displayName, "Ada")
        XCTAssertEqual(report.byCourse.first?.courseCode, "CS101")
    }
}
