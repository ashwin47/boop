import XCTest

/// Walks the main screens of an already-paired app and attaches screenshots (for docs and design review).
/// Run with -only-testing:BoopUITests/ScreenshotUITests against a server that has events.
final class ScreenshotUITests: XCTestCase {
    @MainActor
    func testCaptureScreens() throws {
        let app = XCUIApplication()
        app.launch()
        guard app.navigationBars["Inbox"].waitForExistence(timeout: 8) else { throw XCTSkip("App is not paired") }
        XCTAssertTrue(app.staticTexts["KeyError"].firstMatch.waitForExistence(timeout: 10))
        sleep(1)
        attach(app, "inbox")
        app.staticTexts["KeyError"].firstMatch.tap()
        // A grouped row opens the occurrences first; a single event opens the sheet directly.
        if app.staticTexts["Occurrences"].waitForExistence(timeout: 3) {
            attach(app, "group")
            app.buttons.containing(.staticText, identifier: "KeyError").firstMatch.tap()
        }
        XCTAssertTrue(app.staticTexts["Stacktrace"].waitForExistence(timeout: 5))
        sleep(1)
        attach(app, "event-sheet")
        app.swipeUp()
        sleep(1)
        attach(app, "event-sheet-scrolled")
        app.buttons["Close"].firstMatch.tap()
        if app.staticTexts["Occurrences"].waitForExistence(timeout: 2) { app.navigationBars.buttons.firstMatch.tap() }
        XCTAssertTrue(app.navigationBars["Inbox"].waitForExistence(timeout: 5))
        app.navigationBars["Inbox"].buttons.firstMatch.tap()
        sleep(1)
        attach(app, "filter-menu")
        app.tap()
        app.tabBars.buttons["Settings"].tap()
        sleep(2)
        attach(app, "settings")
    }

    private func attach(_ app: XCUIApplication, _ name: String) {
        let a = XCTAttachment(screenshot: app.screenshot())
        a.name = name
        a.lifetime = .keepAlways
        add(a)
    }
}
