import XCTest

/// End-to-end: pair against a running Boop server (BOOP_TEST_SERVER, default http://localhost:8091),
/// then check the inbox and an event detail render. Skipped when the server is not reachable.
final class PairingFlowUITests: XCTestCase {
    let server = ProcessInfo.processInfo.environment["BOOP_TEST_SERVER"] ?? "http://localhost:8091"

    static func mintToken(_ server: String) async throws -> String? {
        var req = URLRequest(url: URL(string: server + "/api/v1/pairing")!)
        req.httpMethod = "POST"
        req.timeoutInterval = 5
        guard let (data, _) = try? await URLSession.shared.data(for: req) else { return nil }
        let obj = try JSONSerialization.jsonObject(with: data) as? [String: Any]
        return obj?["token"] as? String
    }

    @MainActor
    func testPairThenBrowseInbox() async throws {
        guard let token = try await Self.mintToken(server) else {
            throw XCTSkip("No Boop server at \(server)")
        }
        continueAfterFailure = false
        let app = XCUIApplication()
        app.launch()

        // If a previous run left the app paired, reset via Settings → Re-pair.
        if app.tabBars.buttons["Settings"].waitForExistence(timeout: 3) {
            app.tabBars.buttons["Settings"].tap()
            app.buttons["Re-pair"].firstMatch.tap()
        }

        XCTAssertTrue(app.staticTexts["Welcome to Boop"].waitForExistence(timeout: 5))
        app.buttons["Enter details manually"].firstMatch.tap()

        let serverField = app.textFields["https://boop.example.com"]
        XCTAssertTrue(serverField.waitForExistence(timeout: 3))
        serverField.tap()
        serverField.typeText(server)
        let tokenField = app.textFields["pair_…"]
        tokenField.tap()
        tokenField.typeText(token)
        app.buttons["Pair"].firstMatch.tap()

        XCTAssertTrue(app.staticTexts["Paired"].waitForExistence(timeout: 10))
        app.buttons["Continue"].tap()

        // Notification permission alert (springboard).
        let springboard = XCUIApplication(bundleIdentifier: "com.apple.springboard")
        let allow = springboard.buttons["Allow"]
        if allow.waitForExistence(timeout: 4) { allow.tap() }

        XCTAssertTrue(app.navigationBars["Inbox"].waitForExistence(timeout: 10))
        let row = app.staticTexts["KeyError"].firstMatch
        XCTAssertTrue(row.waitForExistence(timeout: 10), "seeded event should be listed")

        // Three seeded KeyErrors share a fingerprint: one grouped row that opens the occurrences.
        XCTAssertTrue(app.staticTexts["×3"].firstMatch.waitForExistence(timeout: 5), "grouped row should show the count")
        row.tap()
        XCTAssertTrue(app.staticTexts["Occurrences"].waitForExistence(timeout: 5), "tapping a grouped row shows its occurrences")
        XCTAssertTrue(app.staticTexts["uini-keyerror"].firstMatch.exists)
        let occurrence = app.buttons.containing(.staticText, identifier: "KeyError").firstMatch
        XCTAssertTrue(occurrence.waitForExistence(timeout: 5))
        occurrence.tap()

        XCTAssertTrue(app.staticTexts["Stacktrace"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["[REDACTED]"].firstMatch.exists)
        XCTAssertTrue(app.staticTexts["Actions"].firstMatch.exists, "actions card")
        XCTAssertTrue(app.buttons["Open in ErrorTracker"].firstMatch.exists || app.links["Open in ErrorTracker"].firstMatch.exists, "action button")

        // Share menu: Copy / Copy as Markdown / Share.
        app.buttons["event.share"].firstMatch.tap()
        XCTAssertTrue(app.buttons["Copy as Markdown"].waitForExistence(timeout: 3))
        XCTAssertTrue(app.buttons["Copy"].exists)
        XCTAssertTrue(app.buttons["Share"].exists)
        app.buttons["Copy as Markdown"].tap()
        XCTAssertTrue(app.staticTexts["Copied."].waitForExistence(timeout: 5), "copy confirmation")

        app.buttons["Close"].firstMatch.tap()
        XCTAssertTrue(app.staticTexts["Occurrences"].waitForExistence(timeout: 5))
        app.navigationBars.buttons.firstMatch.tap() // back
        XCTAssertTrue(app.navigationBars["Inbox"].waitForExistence(timeout: 5))
    }
}
