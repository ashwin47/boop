import XCTest
@testable import Boop

final class NotificationActionsTests: XCTestCase {
    let info: [AnyHashable: Any] = [
        "event_id": "evt_1",
        "actions": [
            ["label": " Open deploy ", "url": "https://github.com/x/y/actions/runs/1"],
            ["label": "", "url": "https://dropped"],
            ["label": "No url"],
            ["label": "App", "url": "myapp://orders/42"],
        ],
    ]

    func testParsesValidActionsOnly() {
        let a = NotificationActions.parse(info)
        XCTAssertEqual(a.map(\.label), ["Open deploy", "App"])
        XCTAssertEqual(a[1].destination?.scheme, "myapp")
        XCTAssertEqual(NotificationActions.parse([:]), [])
        XCTAssertEqual(NotificationActions.parse(["actions": "nope"]), [])
    }

    func testCapsAtThree() {
        let many: [AnyHashable: Any] = ["actions": (0..<5).map { ["label": "a\($0)", "url": "https://a/\($0)"] }]
        XCTAssertEqual(NotificationActions.parse(many).count, 3)
    }

    func testIdentifiersRoundTrip() {
        XCTAssertEqual(NotificationActions.categoryID(eventID: "evt_1"), "boop.event.actions.evt_1")
        XCTAssertEqual(NotificationActions.actionID(index: 2), "boop.action.2")
        XCTAssertEqual(NotificationActions.actionIndex("boop.action.2"), 2)
        XCTAssertNil(NotificationActions.actionIndex("com.apple.UNNotificationDefaultActionIdentifier"))
        XCTAssertNil(NotificationActions.actionIndex("boop.action.x"))
    }

    func testURLForTappedAction() {
        XCTAssertEqual(NotificationActions.url(for: "boop.action.0", userInfo: info)?.absoluteString, "https://github.com/x/y/actions/runs/1")
        XCTAssertEqual(NotificationActions.url(for: "boop.action.1", userInfo: info)?.absoluteString, "myapp://orders/42")
        XCTAssertNil(NotificationActions.url(for: "boop.action.2", userInfo: info))
        XCTAssertNil(NotificationActions.url(for: "com.apple.UNNotificationDefaultActionIdentifier", userInfo: info))
    }
}
