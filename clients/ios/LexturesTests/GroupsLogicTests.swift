import XCTest
@testable import Lextures

final class GroupsLogicTests: XCTestCase {
    func testCourseCollabDocsFiltersGroupScoped() {
        let docs = [
            CollabDoc(
                id: "1",
                courseId: "c1",
                groupId: nil,
                title: "Course doc",
                docType: .richText,
                createdBy: "u1",
                createdAt: "",
                updatedAt: ""
            ),
            CollabDoc(
                id: "2",
                courseId: "c1",
                groupId: "g1",
                title: "Group doc",
                docType: .richText,
                createdBy: "u1",
                createdAt: "",
                updatedAt: ""
            ),
        ]
        XCTAssertEqual(GroupsLogic.courseCollabDocs(docs).map(\.id), ["1"])
        XCTAssertEqual(GroupsLogic.groupCollabDocs(docs, groupId: "g1").map(\.id), ["2"])
    }

    func testMemberRowsUsesRosterForAuthors() {
        let roster = [
            FeedRosterPerson(userId: "u1", email: "a@school.edu", displayName: "Alex"),
            FeedRosterPerson(userId: "u2", email: "b@school.edu", displayName: "Blair"),
        ]
        let rows = GroupsLogic.memberRows(roster: roster, messageAuthorIds: ["u2"])
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(rows.first?.displayName, "Blair")
    }

    func testDisplayInitials() {
        XCTAssertEqual(GroupsLogic.displayInitials("Alex Kim"), "AK")
        XCTAssertEqual(GroupsLogic.displayInitials("solo"), "SO")
    }
}