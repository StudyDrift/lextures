import XCTest
@testable import Lextures

final class ReaderLogicTests: XCTestCase {
    func testParseVttExtractsCueText() {
        let raw = """
        WEBVTT

        00:00:01.000 --> 00:00:03.000
        Hello <c>world</c>.

        00:00:04.000 --> 00:00:06.500
        Second line.
        """
        let cues = ReaderLogic.parseVtt(raw)
        XCTAssertEqual(cues.count, 2)
        XCTAssertEqual(cues[0].text, "Hello world.")
        XCTAssertEqual(cues[1].start, 4.0, accuracy: 0.01)
    }

    func testActiveCueSelectsCurrentWindow() {
        let cues = [
            ReaderLogic.VttCue(start: 0, end: 2, text: "A"),
            ReaderLogic.VttCue(start: 2, end: 5, text: "B"),
        ]
        XCTAssertEqual(ReaderLogic.activeCue(at: 1.5, in: cues)?.text, "A")
        XCTAssertEqual(ReaderLogic.activeCue(at: 3, in: cues)?.text, "B")
        XCTAssertNil(ReaderLogic.activeCue(at: 6, in: cues))
    }

    func testStorageObjectIdParsesApiFilePath() throws {
        let url = try XCTUnwrap(URL(string: "https://api.example.com/api/v1/files/550e8400-e29b-41d4-a716-446655440000/content"))
        XCTAssertEqual(ReaderLogic.storageObjectId(from: url), "550e8400-e29b-41d4-a716-446655440000")
    }

    func testReadyCaptionsFiltersNonReady() {
        let records = [
            CaptionRecord(id: "1", lang: "en", status: "ready"),
            CaptionRecord(id: "2", lang: "es", status: "processing"),
        ]
        XCTAssertEqual(ReaderLogic.readyCaptions(records).count, 1)
    }

    func testDyslexiaFontFaceMapping() {
        XCTAssertTrue(ReaderLogic.dyslexiaFromFontFace("open-dyslexic"))
        XCTAssertEqual(ReaderLogic.fontFaceFromDyslexia(true, current: "default"), "open-dyslexic")
        XCTAssertEqual(ReaderLogic.fontFaceFromDyslexia(false, current: "open-dyslexic"), "default")
    }
}