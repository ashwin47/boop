import XCTest
@testable import Boop

final class EventTests: XCTestCase {
    let rich = """
    {
      "id": "evt_1", "external_id": "4f9d", "project_id": "prj_1", "project_name": "Uini", "project_slug": "uini", "project_icon": "",
      "source": "error_tracker", "type": "error", "level": "error", "title": "KeyError", "body": "key not found", "fingerprint": "fp",
      "data": {
        "exception": {"type": "KeyError", "message": "boom"},
        "stacktrace": [{"file": "lib/a.ex", "line": 49, "function": "A.b/3", "in_app": true}, {"file": "lib/c.ex", "line": 5, "function": "C.d/1", "in_app": false}],
        "tags": {"environment": "production"},
        "context": {"user_id": "123", "request": {"path": "/x"}},
        "breadcrumbs": [{"timestamp": "12:51", "category": "nav", "message": "GET /"}],
        "custom": {"deep": [1, 2, {"k": "v"}]}
      },
      "occurred_at": "2026-08-28T12:51:44Z",
      "created_at": "2026-08-28T14:10:46.627043000Z"
    }
    """

    func testDecodesRichEventAndSections() throws {
        let e = try APIClient.decoder.decode(Event.self, from: Data(rich.utf8))
        XCTAssertEqual(e.level, .error)
        XCTAssertEqual(e.externalID, "4f9d")
        XCTAssertEqual(Calendar(identifier: .gregorian).component(.year, from: e.createdAt), 2026)
        // Nanosecond timestamps are accepted and keep millisecond precision.
        XCTAssertEqual(e.createdAt.timeIntervalSince1970, 1_787_926_246.627, accuracy: 0.001)

        let s = e.sections
        XCTAssertEqual(s.exception?["type"]?.display, "KeyError")
        XCTAssertEqual(s.frames?.count, 2)
        XCTAssertEqual(s.frames?[0].function, "A.b/3")
        XCTAssertEqual(s.frames?[0].line, "49")
        XCTAssertTrue(s.frames?[0].inApp ?? false)
        XCTAssertFalse(s.frames?[1].inApp ?? true)
        XCTAssertEqual(s.tags?.first?.0, "environment")
        XCTAssertEqual(s.context?.count, 2)
        XCTAssertEqual(s.breadcrumbs?.first?.message, "GET /")
        XCTAssertEqual(s.rest.map(\.0), ["custom"])
        XCTAssertFalse(s.isEmpty)
    }

    func testMinimalEventDefaults() throws {
        let json = #"{"id":"evt_2","project_id":"prj_1","title":"Hi","level":"bogus","occurred_at":"2026-08-28T12:51:44Z","created_at":"2026-08-28T12:51:44Z"}"#
        let e = try APIClient.decoder.decode(Event.self, from: Data(json.utf8))
        XCTAssertEqual(e.level, .info)
        XCTAssertEqual(e.body, "")
        XCTAssertEqual(e.data, .object([:]))
        XCTAssertTrue(e.sections.isEmpty)
    }

    func testJSONValueDisplay() {
        XCTAssertEqual(JSONValue.number(49).display, "49")
        XCTAssertEqual(JSONValue.number(1.5).display, "1.5")
        XCTAssertEqual(JSONValue.array([.null]).display, "[1]")
        XCTAssertEqual(JSONValue.object(["a": .bool(true)]).display, "{1}")
        XCTAssertTrue(JSONValue.object(["a": .bool(true)]).pretty.contains("\"a\" : true"))
    }
}
