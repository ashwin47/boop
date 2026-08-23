import XCTest

/// Verifies notification tap → event sheet. The push itself must be injected from outside while the test
/// waits, e.g. `xcrun simctl push booted com.example.Boop push.json`. Skipped unless BOOP_EXPECT_PUSH=1.
final class NotificationTapUITests: XCTestCase {
    @MainActor
    func testTapOpensEventSheet() throws {
        guard ProcessInfo.processInfo.environment["BOOP_EXPECT_PUSH"] == "1" else { throw XCTSkip("Set BOOP_EXPECT_PUSH=1 and inject a push") }
        let app = XCUIApplication()
        app.launch()
        XCTAssertTrue(app.navigationBars["Inbox"].waitForExistence(timeout: 8))
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let banner = springboard.otherElements["NotificationShortLookView"]
        XCTAssertTrue(banner.waitForExistence(timeout: 150), "no push banner arrived")
        banner.tap()
        XCTAssertTrue(app.staticTexts["Stacktrace"].waitForExistence(timeout: 10), "event sheet should open from the notification")
        XCTAssertTrue(app.staticTexts["KeyError"].firstMatch.exists)
    }
}
