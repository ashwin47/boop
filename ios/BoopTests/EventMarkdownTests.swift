import XCTest
@testable import Boop

final class EventMarkdownTests: XCTestCase {
    func decode(_ json: String) throws -> Event {
        try APIClient.decoder.decode(Event.self, from: Data(json.utf8))
    }

    func testRichErrorRendersEverySection() throws {
        let e = try decode("""
        {
          "id": "evt_1", "project_id": "prj_1", "project_name": "Uini", "source": "error_tracker", "level": "error",
          "title": "KeyError", "body": "key :can_palette? not found", "fingerprint": "uini-keyerror",
          "data": {
            "exception": {"type": "KeyError", "message": "key :can_palette? not found", "module": "Elixir.KeyError"},
            "stacktrace": [
              {"file": "lib/uini_web/live/widget_settings_live.ex", "line": 49, "function": "handle_event/3", "in_app": true},
              {"file": "lib/phoenix_live_view.ex", "line": 12, "function": "Phoenix.LiveView.mount/2", "in_app": false}
            ],
            "tags": {"environment": "production", "runtime": "elixir 1.19"},
            "context": {"user_id": "123", "password": "[REDACTED]", "request": {"path": "/x", "method": "POST"}},
            "breadcrumbs": [{"timestamp": "12:51:40", "category": "navigation", "message": "GET /sites/1"}],
            "custom": {"deep": [1, 2]}
          },
          "actions": [{"label": "Open in ErrorTracker", "url": "https://uini.app/errors/1"}],
          "occurred_at": "2026-08-28T12:51:44Z", "created_at": "2026-08-28T12:51:45Z"
        }
        """)
        let md = e.agentMarkdown
        let expectedOrder = ["# KeyError", "key :can_palette? not found", "## Event", "- Project: Uini", "- Level: error", "- Source: error_tracker",
                             "- Occurred: 2026-08-28T12:51:44Z", "- Fingerprint: uini-keyerror", "- Event id: evt_1",
                             "## Exception", "**KeyError**: key :can_palette? not found", "- module: Elixir.KeyError",
                             "## Environment", "production", "## Tags", "- runtime: elixir 1.19",
                             "## Stack trace", "```", "* handle_event/3  (lib/uini_web/live/widget_settings_live.ex:49)", "  Phoenix.LiveView.mount/2  (lib/phoenix_live_view.ex:12)", "```",
                             "## Context", "- password: [REDACTED]", "- request: {\"method\":\"POST\",\"path\":\"/x\"}", "- user_id: 123",
                             "## Breadcrumbs", "- 12:51:40 [navigation] GET /sites/1",
                             "## Data", "```json", "\"custom\"", "```",
                             "## Links", "- [Open in ErrorTracker](https://uini.app/errors/1)"]
        var cursor = md.startIndex
        for needle in expectedOrder {
            guard let r = md.range(of: needle, range: cursor..<md.endIndex) else {
                return XCTFail("missing or out of order: \(needle)\n\n\(md)")
            }
            cursor = r.upperBound
        }
        XCTAssertTrue(md.hasSuffix("\n"))
        // The environment tag is not repeated under Tags.
        XCTAssertNil(md.range(of: "- environment: production"))
    }

    func testMinimalEventKeepsOnlyWhatExists() throws {
        let e = try decode(#"{"id":"evt_2","project_id":"prj_1","project_name":"Infra","title":"Backup complete","level":"success","occurred_at":"2026-08-28T12:51:44Z","created_at":"2026-08-28T12:51:44Z"}"#)
        let md = e.agentMarkdown
        XCTAssertEqual(md, """
        # Backup complete

        ## Event

        - Project: Infra
        - Level: success
        - Occurred: 2026-08-28T12:51:44Z
        - Event id: evt_2

        """)
        for absent in ["## Exception", "## Stack trace", "## Context", "## Breadcrumbs", "## Data", "## Links", "## Environment"] {
            XCTAssertNil(md.range(of: absent), absent)
        }
    }

    func testGroupInfoAndPlainText() throws {
        let e = try decode(#"{"id":"evt_3","project_id":"prj_1","project_name":"Uini","title":"Timeout","body":"upstream 504","level":"warning","source":"cron","fingerprint":"t","group":{"count":47,"first_seen":"2026-08-28T09:31:00Z","last_seen":"2026-08-28T10:42:00Z"},"actions":[{"label":"Open","url":"https://x"}],"occurred_at":"2026-08-28T10:42:00Z","created_at":"2026-08-28T10:42:00Z"}"#)
        XCTAssertTrue(e.agentMarkdown.contains("- Occurrences: 47 (first 2026-08-28T09:31:00Z, last 2026-08-28T10:42:00Z)"))
        let plain = e.plainText
        XCTAssertTrue(plain.hasPrefix("Timeout\nupstream 504\n\nUini · Warning · cron · "))
        XCTAssertTrue(plain.hasSuffix("\nOpen: https://x"))
    }

    func testInlineJSON() {
        XCTAssertEqual(JSONValue.string("a/b").inline, "a/b")
        XCTAssertEqual(JSONValue.object(["b": .number(1), "a": .array([.string("x/y")])]).inline, #"{"a":["x/y"],"b":1}"#)
    }
}
