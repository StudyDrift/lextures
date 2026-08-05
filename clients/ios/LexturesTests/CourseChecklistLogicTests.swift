import XCTest
@testable import Lextures

final class CourseChecklistLogicTests: XCTestCase {
    private func fixtureURL() -> URL {
        let thisFile = URL(fileURLWithPath: #filePath)
        let direct = thisFile
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("mobile/fixtures/checklist/logic-parity.json")
        if FileManager.default.fileExists(atPath: direct.path) { return direct }
        var dir = URL(fileURLWithPath: FileManager.default.currentDirectoryPath)
        for _ in 0 ..< 10 {
            let candidate = dir.appendingPathComponent("clients/mobile/fixtures/checklist/logic-parity.json")
            if FileManager.default.fileExists(atPath: candidate.path) { return candidate }
            dir = dir.deletingLastPathComponent()
        }
        return direct
    }

    private func fixtureRoot() throws -> [String: Any] {
        let data = try Data(contentsOf: fixtureURL())
        return try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
    }

    private func table() -> [String: String] {
        CourseChecklistTargetTable.load()
    }

    func testBadgeMatchesFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["badge"] as? [[String: Any]])
        for item in cases {
            let count = try XCTUnwrap(item["outstandingEssential"] as? Int)
            let badge = CourseChecklistLogic.badgePresentation(outstandingEssential: count)
            XCTAssertEqual(badge.visible, item["visible"] as? Bool)
            XCTAssertEqual(badge.text, item["text"] as? String)
            let contains = item["accessibilityContains"] as? String ?? ""
            if !contains.isEmpty {
                XCTAssertEqual(badge.accessibilityLabel, contains)
            }
        }
    }

    func testStatusMatchesFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["status"] as? [[String: Any]])
        for item in cases {
            let raw = try XCTUnwrap(item["raw"] as? String)
            XCTAssertEqual(CourseChecklistLogic.normalizeStatus(raw).rawValue, item["normalized"] as? String)
            XCTAssertEqual(CourseChecklistLogic.isOutstanding(raw), item["outstanding"] as? Bool)
            XCTAssertEqual(CourseChecklistLogic.accessibilityStatusValue(raw), item["accessibilityValue"] as? String)
        }
    }

    func testProgressMatchesFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["progress"] as? [[String: Any]])
        for item in cases {
            let done = try XCTUnwrap(item["done"] as? Int)
            let total = try XCTUnwrap(item["total"] as? Int)
            let fraction = try XCTUnwrap(item["fraction"] as? Double)
            XCTAssertEqual(CourseChecklistLogic.progressFraction(done: done, total: total), fraction, accuracy: 0.0001)
            XCTAssertEqual(CourseChecklistLogic.progressLabel(done: done, total: total), item["label"] as? String)
        }
    }

    func testTargetResolutionMatchesFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["targets"] as? [[String: Any]])
        let table = table()
        XCTAssertFalse(table.isEmpty, "target table should load from packages or bundle")
        for item in cases {
            let name = item["name"] as? String ?? "?"
            let courseCode = try XCTUnwrap(item["courseCode"] as? String)
            let targetObj = item["target"] as? [String: Any]
            let target: ChecklistNavTarget?
            if let targetObj {
                target = ChecklistNavTarget(
                    route: try XCTUnwrap(targetObj["route"] as? String),
                    anchor: targetObj["anchor"] as? String,
                    entityKey: targetObj["entityKey"] as? String
                )
            } else {
                target = nil
            }
            let resolved = CourseChecklistLogic.resolveTarget(target, courseCode: courseCode, table: table)
            XCTAssertEqual(resolved.kind.rawValue, item["expectedKind"] as? String, name)
            if let section = item["expectedSection"] as? String {
                XCTAssertEqual(resolved.workspaceSection?.rawValue, section, name)
            }
            if let contains = item["webPathContains"] as? String {
                XCTAssertTrue(resolved.webPath?.contains(contains) == true, "\(name): \(resolved.webPath ?? "nil")")
            }
        }
    }

    func testVisibilityMatchesFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["visibility"] as? [[String: Any]])
        for item in cases {
            let staff = try XCTUnwrap(item["viewerIsStaff"] as? Bool)
            let roleRaw = try XCTUnwrap(item["roleContext"] as? String)
            let role = MobileRoleContext(rawValue: roleRaw)!
            let expected = try XCTUnwrap(item["show"] as? Bool)
            XCTAssertEqual(
                CourseChecklistLogic.shouldShowWorkspaceSection(viewerIsStaff: staff, roleContext: role),
                expected
            )
        }
    }

    func testNotesMatchFixture() throws {
        let cases = try XCTUnwrap(try fixtureRoot()["notes"] as? [[String: Any]])
        for item in cases {
            if let repeatCount = item["repeat"] as? Int {
                let unit = item["input"] as? String ?? "x"
                let input = String(repeating: unit, count: repeatCount)
                let out = CourseChecklistLogic.clampedNote(input)
                XCTAssertEqual(out.count, item["expectedLength"] as? Int)
            } else if item["input"] is NSNull || item["input"] == nil {
                XCTAssertEqual(CourseChecklistLogic.clampedNote(nil), item["expected"] as? String)
            } else {
                XCTAssertEqual(CourseChecklistLogic.clampedNote(item["input"] as? String), item["expected"] as? String)
            }
        }
    }

    func testPresentationParity() throws {
        let root = try XCTUnwrap(try fixtureRoot()["presentation"] as? [String: Any])
        let checklistJSON = try JSONSerialization.data(withJSONObject: try XCTUnwrap(root["checklist"]))
        let checklist = try JSONDecoder().decode(CourseChecklist.self, from: checklistJSON)
        let presented = CourseChecklistLogic.presentChecklist(checklist, table: table())
        let expected = try XCTUnwrap(root["expected"] as? [[String: Any]])
        XCTAssertEqual(presented.count, expected.count)
        for (idx, item) in expected.enumerated() {
            let presentedItem = presented[idx]
            XCTAssertEqual(presentedItem.id, item["id"] as? String)
            XCTAssertEqual(presentedItem.title, item["title"] as? String)
            XCTAssertEqual(presentedItem.status, item["status"] as? String)
            XCTAssertEqual(presentedItem.accessibilityValue, item["accessibilityValue"] as? String)
            XCTAssertEqual(presentedItem.isDone, item["isDone"] as? Bool)
            XCTAssertEqual(presentedItem.isOutstanding, item["isOutstanding"] as? Bool)
            XCTAssertEqual(presentedItem.targetKind, item["targetKind"] as? String)
        }
    }

    func testSummaryStoreNeverUsesDiskPrefix() throws {
        let storage = try XCTUnwrap(try fixtureRoot()["storage"] as? [String: Any])
        let prefix = try XCTUnwrap(storage["forbiddenKeyPrefix"] as? String)
        XCTAssertEqual(CourseChecklistLogic.offlineCacheKeyPrefix, prefix)
        // OfflineCacheKey helpers must not include the checklist prefix.
        let coursesKey = OfflineCacheKey.courses()
        XCTAssertFalse(coursesKey.hasPrefix(prefix))
        CourseChecklistSummaryStore.shared.clearAll()
        CourseChecklistSummaryStore.shared.put(
            courseCode: "C-test",
            summary: CourseChecklistSummary(
                outstandingEssential: 2,
                outstandingTotal: 3,
                done: 1,
                total: 4,
                dismissed: 0,
                computedAt: "now",
                stale: false
            )
        )
        XCTAssertEqual(CourseChecklistSummaryStore.shared.outstandingEssential(courseCode: "C-test"), 2)
        CourseChecklistSummaryStore.shared.clearAll()
    }

    func testWorkspaceInsertsChecklistAfterOverviewForStaffTeaching() {
        let course = CourseSummary(
            id: "1",
            courseCode: "demo",
            title: "Demo",
            description: "",
            viewerEnrollmentRoles: ["teacher"]
        )
        let sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course, roleContext: .teaching)
        )
        XCTAssertEqual(sections.first, .overview)
        XCTAssertEqual(sections.dropFirst().first, .checklist)
        let learning = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course: course, roleContext: .learning)
        )
        XCTAssertFalse(learning.contains(.checklist))
    }
}
