import XCTest
@testable import Lextures

final class CourseGradingAgentsLogicTests: XCTestCase {
    func testGradableOptionsExcludesExistingAgents() {
        let structure = [
            CourseStructureItem(
                id: "a1", sortOrder: 0, kind: "assignment", title: "Essay", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
            CourseStructureItem(
                id: "q1", sortOrder: 1, kind: "quiz", title: "Quiz 1", parentId: nil,
                published: true, dueAt: nil, pointsWorth: nil, pointsPossible: nil, archived: nil, updatedAt: nil
            ),
        ]
        let options = CourseGradingAgentsLogic.gradableOptions(from: structure, excluding: ["a1"])
        XCTAssertEqual(options.count, 1)
        XCTAssertEqual(options.first?.id, "q1")
    }

    func testIsDirtyDetectsPromptChange() {
        let baseline = CourseGradingAgentsLogic.draft(from: nil as GraderAgentConfig?)
        var current = baseline
        XCTAssertFalse(CourseGradingAgentsLogic.isDirty(current: current, baseline: baseline))
        current.prompt = "Grade for clarity"
        XCTAssertTrue(CourseGradingAgentsLogic.isDirty(current: current, baseline: baseline))
    }

    func testValidateDraftRequiresPrompt() {
        XCTAssertEqual(CourseGradingAgentsLogic.validateDraft(.init(
            prompt: "   ",
            includeAssignmentContent: false,
            includeRubric: false,
            status: "draft",
            autoGradeNew: false,
            postPolicy: "draft",
            confidenceFloor: nil,
            workflowGraph: nil
        )), .promptRequired)
        XCTAssertNil(CourseGradingAgentsLogic.validateDraft(.init(
            prompt: "Use the rubric",
            includeAssignmentContent: true,
            includeRubric: true,
            status: "draft",
            autoGradeNew: false,
            postPolicy: "draft",
            confidenceFloor: nil,
            workflowGraph: nil
        )))
    }

    func testBuildPutBodyTrimsPrompt() {
        let draft = CourseGradingAgentsLogic.AgentDraft(
            prompt: "  Grade carefully  ",
            includeAssignmentContent: true,
            includeRubric: false,
            status: "accepted",
            autoGradeNew: true,
            postPolicy: "draft",
            confidenceFloor: nil,
            workflowGraph: nil
        )
        let body = CourseGradingAgentsLogic.buildPutBody(current: draft, itemKind: "assignment")
        XCTAssertEqual(body.prompt, "Grade carefully")
        XCTAssertEqual(body.status, "accepted")
        XCTAssertTrue(body.autoGradeNew)
    }

    func testDefaultWorkflowGraphIncludesOutputNode() {
        let graph = CourseGradingAgentsLogic.defaultWorkflowGraph(itemKind: "assignment")
        guard case .object(let object) = graph,
              case .array(let nodes) = object["nodes"] else {
            return XCTFail("expected object graph")
        }
        XCTAssertEqual(nodes.count, 1)
        guard case .object(let node) = nodes[0],
              case .string(let type)? = node["type"] else {
            return XCTFail("expected output node")
        }
        XCTAssertEqual(type, "output")
    }

    func testGraderAgentPathUsesQuizCollection() {
        XCTAssertTrue(
            CourseGradingAgentsLogic.graderAgentPath(courseCode: "C-1", itemId: "item", itemKind: "quiz")
                .contains("/quizzes/")
        )
        XCTAssertTrue(
            CourseGradingAgentsLogic.graderAgentPath(courseCode: "C-1", itemId: "item", itemKind: "assignment")
                .contains("/assignments/")
        )
    }
}
