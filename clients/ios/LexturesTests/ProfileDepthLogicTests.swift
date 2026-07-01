import XCTest
@testable import Lextures

final class ProfileDepthLogicTests: XCTestCase {
    func testValidateCustomFieldsFlagsRequiredBlank() {
        let defs = [
            ProfileFieldDefinition(id: "1", key: "student_id", label: "Student ID", fieldType: "text", selectOptions: nil, isRequired: true),
        ]
        let errors = ProfileDepthLogic.validateCustomFields(definitions: defs, draft: [:])
        XCTAssertEqual(errors["student_id"], L.text("mobile.profileDepth.requiredField"))
    }

    func testValidateCustomFieldsAcceptsValidNumber() {
        let defs = [
            ProfileFieldDefinition(id: "1", key: "gpa", label: "GPA", fieldType: "number", selectOptions: nil, isRequired: false),
        ]
        let errors = ProfileDepthLogic.validateCustomFields(definitions: defs, draft: ["gpa": "3.5"])
        XCTAssertTrue(errors.isEmpty)
    }

    func testValidateCustomFieldsRejectsInvalidSelect() {
        let defs = [
            ProfileFieldDefinition(
                id: "1",
                key: "dept",
                label: "Department",
                fieldType: "select",
                selectOptions: ["Math", "Science"],
                isRequired: true
            ),
        ]
        let errors = ProfileDepthLogic.validateCustomFields(definitions: defs, draft: ["dept": "History"])
        XCTAssertEqual(errors["dept"], L.text("mobile.profileDepth.invalidSelect"))
    }

    func testEncodeCustomFieldValues() {
        let defs = [
            ProfileFieldDefinition(id: "1", key: "active", label: "Active", fieldType: "boolean", selectOptions: nil, isRequired: false),
            ProfileFieldDefinition(id: "2", key: "note", label: "Note", fieldType: "text", selectOptions: nil, isRequired: false),
        ]
        let encoded = ProfileDepthLogic.encodeCustomFieldValues(
            definitions: defs,
            draft: ["active": "true", "note": "hello"]
        )
        XCTAssertEqual(encoded["active"], .bool(true))
        XCTAssertEqual(encoded["note"], .string("hello"))
    }

    func testLatestConsentByStudyKeepsNewestPerStudy() {
        let history = [
            ConsentHistoryEntry(id: "a", studyId: "s1", studyTitle: "A", decision: .granted, createdAt: "2026-01-02"),
            ConsentHistoryEntry(id: "b", studyId: "s1", studyTitle: "A", decision: .withdrawn, createdAt: "2026-01-01"),
            ConsentHistoryEntry(id: "c", studyId: "s2", studyTitle: "B", decision: .declined, createdAt: "2026-01-03"),
        ]
        let latest = ProfileDepthLogic.latestConsentByStudy(history)
        XCTAssertEqual(latest.count, 2)
        XCTAssertEqual(latest[0].decision, .granted)
        XCTAssertEqual(latest[1].studyId, "s2")
    }

    func testShouldShowPersonalDetails() {
        XCTAssertTrue(ProfileDepthLogic.shouldShowPersonalDetails(customFieldsEnabled: true, demographicsEnabled: false, fieldCount: 1))
        XCTAssertTrue(ProfileDepthLogic.shouldShowPersonalDetails(customFieldsEnabled: false, demographicsEnabled: true, fieldCount: 0))
        XCTAssertFalse(ProfileDepthLogic.shouldShowPersonalDetails(customFieldsEnabled: true, demographicsEnabled: false, fieldCount: 0))
    }

    func testShouldShowResearchStudies() {
        XCTAssertTrue(ProfileDepthLogic.shouldShowResearchStudies(researchConsentEnabled: true, pendingCount: 1, historyCount: 0))
        XCTAssertTrue(ProfileDepthLogic.shouldShowResearchStudies(researchConsentEnabled: true, pendingCount: 0, historyCount: 2))
        XCTAssertFalse(ProfileDepthLogic.shouldShowResearchStudies(researchConsentEnabled: false, pendingCount: 1, historyCount: 1))
        XCTAssertFalse(ProfileDepthLogic.shouldShowResearchStudies(researchConsentEnabled: true, pendingCount: 0, historyCount: 0))
    }

    func testDecodesProfileFieldsResponse() throws {
        let json = """
        {"fields":[{"id":"f1","key":"student_id","label":"Student ID","fieldType":"text","isRequired":true}],\
        "values":{"student_id":"123"}}
        """
        let response = try JSONDecoder().decode(ProfileFieldsResponse.self, from: Data(json.utf8))
        XCTAssertEqual(response.fields.count, 1)
        XCTAssertEqual(response.values["student_id"], .string("123"))
    }
}