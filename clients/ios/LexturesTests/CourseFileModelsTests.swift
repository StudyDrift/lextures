import XCTest
@testable import Lextures

final class CourseFileModelsTests: XCTestCase {
    func testPreviewKindDetectsPdfAndVideo() {
        XCTAssertEqual(CourseFileLogic.previewKind(mimeType: "application/pdf", fileName: "notes.pdf"), .pdf)
        XCTAssertEqual(CourseFileLogic.previewKind(mimeType: "video/mp4", fileName: "lecture.mp4"), .video)
        XCTAssertEqual(CourseFileLogic.previewKind(mimeType: "image/png", fileName: "slide.png"), .image)
        XCTAssertEqual(CourseFileLogic.previewKind(mimeType: nil, fileName: "data.zip"), .downloadOnly)
    }

    func testContentPathForFileManagerAndCourseFile() {
        let managerPath = CourseFileLogic.contentPath(
            courseCode: "CS101",
            source: .fileManager(itemId: "item-1")
        )
        XCTAssertTrue(managerPath.contains("/files/items/item-1/content"))

        let legacyPath = CourseFileLogic.contentPath(
            courseCode: "CS101",
            source: .courseFile(fileId: "file-1")
        )
        XCTAssertTrue(legacyPath.contains("/course-files/file-1/content"))
    }

    func testDownloadKeyStable() {
        let target = FilePreviewTarget(
            courseCode: "CS101",
            displayName: "Reading.pdf",
            mimeType: "application/pdf",
            byteSize: 1024,
            source: .fileManager(itemId: "abc")
        )
        XCTAssertEqual(CourseFileLogic.downloadKey(courseCode: "CS101", target: target), "download:CS101:fm:abc")
    }

    func testFilePreviewTargetFromModuleItem() {
        let item = CourseStructureItem(
            id: "f1",
            sortOrder: 0,
            kind: "file",
            title: "Week 1.pdf",
            parentId: "m1",
            published: true,
            dueAt: nil,
            pointsWorth: nil,
            pointsPossible: nil
        )
        let target = FilePreviewTarget.from(moduleItem: item, courseCode: "CS101")
        XCTAssertEqual(target.displayName, "Week 1.pdf")
        XCTAssertEqual(target.mimeType, "application/pdf")
    }
}
