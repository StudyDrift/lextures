import XCTest
@testable import Lextures

final class CourseSectionsLogicTests: XCTestCase {
    func testShouldShowEditorsWhenSectionsEnabled() {
        XCTAssertTrue(CourseSectionsLogic.shouldShowEditors(sectionsEnabled: true))
        XCTAssertFalse(CourseSectionsLogic.shouldShowEditors(sectionsEnabled: false))
    }

    func testActiveSectionsFiltersArchived() {
        let sections = [
            CourseSection(id: "1", sectionCode: "A", name: nil, status: "active", courseId: nil),
            CourseSection(id: "2", sectionCode: "B", name: nil, status: "archived", courseId: nil),
        ]
        XCTAssertEqual(CourseSectionsLogic.activeSections(sections).map(\.id), ["1"])
    }

    func testRosterCountStudentsOnly() {
        let enrollments = [
            enrollment(id: "e1", role: "student", sectionId: "sec-a"),
            enrollment(id: "e2", role: "teacher", sectionId: "sec-a"),
            enrollment(id: "e3", role: "student", sectionId: "sec-b"),
        ]
        XCTAssertEqual(CourseSectionsLogic.rosterCount(sectionId: "sec-a", enrollments: enrollments), 1)
    }

    func testAssignmentItemsFromStructure() {
        let items = [
            CourseStructureItem(
                id: "1", sortOrder: 0, kind: "assignment", title: "Essay", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
            CourseStructureItem(
                id: "2", sortOrder: 1, kind: "module", title: "Week 1", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
        ]
        XCTAssertEqual(CourseSectionsLogic.assignmentItems(from: items).map(\.id), ["1"])
    }

    func testBuildOverrideBodyRejectsInvalidDate() {
        XCTAssertNil(CourseSectionsLogic.buildOverrideBody(dueAtLocal: "not-a-date"))
    }

    func testBuildOverrideBodyClearsWhenEmpty() {
        let body = CourseSectionsLogic.buildOverrideBody(dueAtLocal: " ")
        XCTAssertNotNil(body)
        XCTAssertNil(body?.dueAt)
    }

    func testCrossListAddCandidates() {
        let sections = [
            CourseSection(id: "s1", sectionCode: "001", name: nil, status: "active", courseId: nil),
            CourseSection(id: "s2", sectionCode: "002", name: nil, status: "active", courseId: nil),
        ]
        let group = CrossListGroup(
            id: "g1",
            courseId: "c1",
            name: nil,
            createdAt: nil,
            primarySectionId: "s1",
            members: [CrossListMember(sectionId: "s1", isPrimary: true, sectionCode: "001", sectionName: nil)]
        )
        XCTAssertEqual(
            CourseSectionsLogic.crossListAddCandidates(activeSections: sections, group: group).map(\.id),
            ["s2"]
        )
    }

    func testCanManageCrossListingPermission() {
        XCTAssertFalse(CourseSectionsLogic.canManageCrossListing(permissions: []))
        XCTAssertTrue(
            CourseSectionsLogic.canManageCrossListing(
                permissions: [CourseSectionsLogic.orgUnitsAdminPermission]
            )
        )
    }

    private func enrollment(id: String, role: String, sectionId: String) -> CourseEnrollment {
        CourseEnrollment(
            id: id,
            userId: "u-\(id)",
            displayName: "User \(id)",
            avatarUrl: nil,
            role: role,
            roleDisplay: nil,
            lastCourseAccessAt: nil,
            sectionId: sectionId,
            sectionCode: nil,
            sectionName: nil,
            state: nil,
            invitationPending: nil
        )
    }
}
