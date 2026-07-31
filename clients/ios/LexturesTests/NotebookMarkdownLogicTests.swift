import XCTest
@testable import Lextures

final class NotebookMarkdownLogicTests: XCTestCase {
    func testNormalizeTablesHealsBlankLines() {
        let input = """
        | A | B |

        | --- | --- |

        | 1 | 2 |

        After
        """
        let healed = MarkdownTableLogic.normalizeMarkdownTables(input)
        XCTAssertFalse(healed.contains("|\n\n|"))
        XCTAssertTrue(healed.contains("| A | B |"))
        XCTAssertTrue(healed.contains("| --- | --- |"))
        XCTAssertTrue(healed.contains("| 1 | 2 |"))
        XCTAssertTrue(healed.contains("After"))
    }

    func testNormalizeTablesLeavesPipeProseAlone() {
        let input = "Use | as OR in boolean logic."
        XCTAssertEqual(MarkdownTableLogic.normalizeMarkdownTables(input), input)
    }

    func testParseBasicTableAlignment() {
        let md = """
        | Left | Center | Right |
        | :--- | :---: | ---: |
        | a | b | c |
        """
        let blocks = NotebookMarkdown.parseBlocks(md)
        guard case .table(let align, let header, let rows) = blocks.first?.kind else {
            return XCTFail("expected table")
        }
        XCTAssertEqual(header, ["Left", "Center", "Right"])
        XCTAssertEqual(align, [.left, .center, .right])
        XCTAssertEqual(rows, [["a", "b", "c"]])
    }

    func testParseCodeLanguageAndLexTool() {
        let code = NotebookMarkdown.parseBlocks("```python\nprint(1)\n```")
        guard case .code(let language, let source) = code.first?.kind else {
            return XCTFail("expected code")
        }
        XCTAssertEqual(language, "python")
        XCTAssertEqual(source, "print(1)")

        let tool = NotebookMarkdown.parseBlocks("""
        ```lex-tool
        {"instanceId":"abc-123","toolId":"inline_questions","v":1}
        ```
        """)
        guard case .toolFence(let instanceId, let toolId, let version) = tool.first?.kind else {
            return XCTFail("expected toolFence")
        }
        XCTAssertEqual(instanceId, "abc-123")
        XCTAssertEqual(toolId, "inline_questions")
        XCTAssertEqual(version, 1)
        XCTAssertFalse(tool.contains { if case .code = $0.kind { return true }; return false })
    }

    func testParseMathAndTaskList() {
        let math = NotebookMarkdown.parseBlocks("$$\\frac{a}{b}$$")
        guard case .math(let latex, let display) = math.first?.kind else {
            return XCTFail("expected math")
        }
        XCTAssertEqual(latex, "\\frac{a}{b}")
        XCTAssertTrue(display)

        let tasks = NotebookMarkdown.parseBlocks("- [ ] Todo\n- [x] Done")
        XCTAssertEqual(tasks.count, 2)
        guard case .taskItem(false, "Todo", _) = tasks[0].kind else { return XCTFail("todo") }
        guard case .taskItem(true, "Done", _) = tasks[1].kind else { return XCTFail("done") }
    }

    func testPreviewHidesLexToolJSON() {
        let preview = NotebookMarkdown.previewText("""
        Hello

        ```lex-tool
        {"instanceId":"abc-123","toolId":"flashcards","v":1}
        ```
        """)
        XCTAssertFalse(preview.contains("abc-123"))
        XCTAssertFalse(preview.contains("instanceId"))
        XCTAssertTrue(preview.contains("flashcards") || preview.contains("Hello"))
    }

    func testSharedFixtureCorpusKindSequences() throws {
        let url = fixtureCorpusURL()
        let data = try Data(contentsOf: url)
        let root = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        let fixtures = root?["fixtures"] as? [[String: Any]] ?? []
        XCTAssertFalse(fixtures.isEmpty)
        for fixture in fixtures {
            let id = fixture["id"] as? String ?? "?"
            let markdown = fixture["markdown"] as? String ?? ""
            let expected = fixture["kinds"] as? [String] ?? []
            let actual = NotebookMarkdown.parseBlocks(markdown).map { NotebookMarkdown.fixtureKindName($0.kind) }
            XCTAssertEqual(actual, expected, "fixture \(id)")
        }
    }

    private func fixtureCorpusURL() -> URL {
        // Repo layout: clients/ios/LexturesTests/File.swift → ../../../mobile/fixtures/markdown/corpus.json
        let thisFile = URL(fileURLWithPath: #filePath)
        let corpus = thisFile
            .deletingLastPathComponent() // LexturesTests
            .deletingLastPathComponent() // ios
            .deletingLastPathComponent() // clients
            .appendingPathComponent("mobile/fixtures/markdown/corpus.json")
        if FileManager.default.fileExists(atPath: corpus.path) { return corpus }
        // Fallback: walk up from CWD (and from #filePath) for CI / DerivedData checkouts.
        var searchRoots = [
            URL(fileURLWithPath: FileManager.default.currentDirectoryPath),
            thisFile.deletingLastPathComponent(),
        ]
        for root in searchRoots {
            var dir = root
            for _ in 0 ..< 8 {
                for relative in [
                    "clients/mobile/fixtures/markdown/corpus.json",
                    "mobile/fixtures/markdown/corpus.json",
                ] {
                    let candidate = dir.appendingPathComponent(relative)
                    if FileManager.default.fileExists(atPath: candidate.path) { return candidate }
                }
                dir = dir.deletingLastPathComponent()
            }
        }
        return corpus
    }
}
