import XCTest
@testable import Lextures

final class CourseImportExportLogicTests: XCTestCase {
    func testExportFileName() {
        XCTAssertEqual(
            CourseImportExportLogic.exportFileName(courseCode: "C-ABC123"),
            "C-ABC123-course-export.json"
        )
    }

    func testWebImportExportPath() {
        XCTAssertEqual(
            CourseImportExportLogic.webImportExportPath(courseCode: "C-1"),
            "/courses/C-1/settings/import-export"
        )
    }

    func testParseValidImportFile() throws {
        let data = Data("""
        {"formatVersion":1,"courseCode":"C-1","course":{"title":"Intro"}}
        """.utf8)
        let parsed = try CourseImportExportLogic.parseImportFileData(data)
        XCTAssertEqual(parsed["courseCode"], .string("C-1"))
    }

    func testParseInvalidJson() {
        let data = Data("not json".utf8)
        XCTAssertThrowsError(try CourseImportExportLogic.parseImportFileData(data)) { error in
            XCTAssertEqual(error as? CourseImportExportLogic.ImportExportError, .invalidJSON)
        }
    }

    func testParseEmptyObjectRejected() {
        let data = Data("{}".utf8)
        XCTAssertThrowsError(try CourseImportExportLogic.parseImportFileData(data)) { error in
            XCTAssertEqual(error as? CourseImportExportLogic.ImportExportError, .invalidObject)
        }
    }

    func testFileTooLargeRejected() {
        let oversized = Data(count: CourseImportExportLogic.maxImportBytes + 1)
        XCTAssertThrowsError(try CourseImportExportLogic.parseImportFileData(oversized)) { error in
            XCTAssertEqual(error as? CourseImportExportLogic.ImportExportError, .fileTooLarge)
        }
    }

    func testImportConfirmMessageKeys() {
        XCTAssertEqual(
            CourseImportExportLogic.importConfirmMessageKey(.erase),
            "mobile.courseSettings.importExport.confirmMessage.erase"
        )
    }
}
