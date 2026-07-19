import XCTest
@testable import Lextures
import CoreGraphics

final class WhiteboardLogicTests: XCTestCase {
    override func setUp() {
        super.setUp()
        WhiteboardObservability.resetForTests()
    }

    func testCanEditRequiresFlagAndStaff() {
        var features = MobilePlatformFeatures()
        features.ffMobileWhiteboardEdit = false
        XCTAssertFalse(WhiteboardLogic.canEdit(viewerIsStaff: true, features: features))
        features.ffMobileWhiteboardEdit = true
        XCTAssertTrue(WhiteboardLogic.canEdit(viewerIsStaff: true, features: features))
        XCTAssertFalse(WhiteboardLogic.canEdit(viewerIsStaff: false, features: features))
    }

    func testTitleValidation() {
        XCTAssertFalse(WhiteboardLogic.isValidTitle("  "))
        XCTAssertTrue(WhiteboardLogic.isValidTitle(" Board "))
        XCTAssertEqual(WhiteboardLogic.normalizeTitle(" Board "), "Board")
        XCTAssertEqual(WhiteboardLogic.defaultTitle(existingCount: 2), "Whiteboard 3")
    }

    func testSerializeStrokeRoundTrip() {
        let stroke = WhiteboardLogic.stroke(
            color: "#ef4444",
            width: 4,
            pts: [CGPoint(x: 1, y: 2), CGPoint(x: 3, y: 4)]
        )
        let json = WhiteboardLogic.canvasDataJSONString([stroke])
        let parsed = WhiteboardLogic.parseElements(fromJSON: json)
        XCTAssertEqual(parsed.count, 1)
        XCTAssertEqual(parsed[0].type, "stroke")
        XCTAssertEqual(parsed[0].color, "#ef4444")
        XCTAssertEqual(parsed[0].pts?.count, 2)
        XCTAssertEqual(parsed[0].pts?[0][0], 1, accuracy: 0.001)
    }

    func testSerializeShapeBuildersMatchWebKeys() {
        let rect = WhiteboardLogic.rect(
            color: "#22c55e",
            width: 2,
            from: CGPoint(x: 10, y: 20),
            to: CGPoint(x: 40, y: 50)
        )
        let dict = WhiteboardLogic.serializeElement(rect)!
        XCTAssertEqual(dict["type"] as? String, "rect")
        XCTAssertEqual(dict["x"] as? Double, 10)
        XCTAssertEqual(dict["y"] as? Double, 20)
        XCTAssertEqual(dict["w"] as? Double, 30)
        XCTAssertEqual(dict["h"] as? Double, 30)
    }

    func testUndoRedoStack() {
        var history = WhiteboardLogic.History()
        let before = [WhiteboardLogic.stroke(color: "#000000", width: 2, pts: [CGPoint(x: 0, y: 0), CGPoint(x: 1, y: 1)])]
        let after = before + [WhiteboardLogic.line(color: "#111111", width: 2, from: .zero, to: CGPoint(x: 5, y: 5))]
        history.push(before)
        XCTAssertTrue(history.canUndo)
        let undone = history.undo(current: after)
        XCTAssertEqual(undone?.count, 1)
        let redone = history.redo(current: undone!)
        XCTAssertEqual(redone?.count, 2)
    }

    func testEraseRemovesHitStroke() {
        let stroke = WhiteboardLogic.stroke(
            color: "#000000",
            width: 2,
            pts: [CGPoint(x: 0, y: 0), CGPoint(x: 10, y: 0)]
        )
        let kept = WhiteboardLogic.line(color: "#000000", width: 2, from: CGPoint(x: 100, y: 100), to: CGPoint(x: 120, y: 120))
        let next = WhiteboardLogic.erase(from: [stroke, kept], at: CGPoint(x: 5, y: 0), radius: 8)
        XCTAssertEqual(next.count, 1)
        XCTAssertEqual(next[0].type, "line")
    }

    func testTranslateAndPick() {
        let rect = WhiteboardLogic.rect(color: "#000", width: 2, from: CGPoint(x: 0, y: 0), to: CGPoint(x: 20, y: 20))
        let moved = WhiteboardLogic.translate(rect, dx: 5, dy: 10)
        XCTAssertEqual(moved.rectX, 5, accuracy: 0.001)
        XCTAssertEqual(moved.rectY, 10, accuracy: 0.001)
        XCTAssertEqual(WhiteboardLogic.pickElement([moved], at: CGPoint(x: 5, y: 10)), 0)
        XCTAssertNil(WhiteboardLogic.pickElement([moved], at: CGPoint(x: 200, y: 200)))
    }

    func testStylusExclusiveDrawing() {
        XCTAssertTrue(WhiteboardLogic.shouldAcceptTouch(isStylus: false, stylusExclusiveDrawing: false))
        XCTAssertFalse(WhiteboardLogic.shouldAcceptTouch(isStylus: false, stylusExclusiveDrawing: true))
        XCTAssertTrue(WhiteboardLogic.shouldAcceptTouch(isStylus: true, stylusExclusiveDrawing: true))
    }

    func testObservabilityCounters() {
        WhiteboardObservability.record("whiteboard_edited", attributes: ["tool": "pen"])
        WhiteboardObservability.record("whiteboard_undo")
        XCTAssertEqual(WhiteboardObservability.count(for: "whiteboard_edited"), 1)
        XCTAssertEqual(WhiteboardObservability.count(for: "whiteboard_undo"), 1)
    }
}
