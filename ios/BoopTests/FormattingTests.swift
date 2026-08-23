import XCTest
@testable import Boop

final class FormattingTests: XCTestCase {
    let now = Date(timeIntervalSince1970: 1_787_961_600) // 2026-08-29T00:00:00Z

    func testRelative() {
        XCTAssertEqual(Formatting.relative(now.addingTimeInterval(-10), now: now), "now")
        XCTAssertEqual(Formatting.relative(now.addingTimeInterval(-120), now: now), "2m")
        XCTAssertEqual(Formatting.relative(now.addingTimeInterval(-3 * 3600), now: now), "3h")
        XCTAssertEqual(Formatting.relative(now.addingTimeInterval(-2 * 86400), now: now), "2d")
        XCTAssertFalse(Formatting.relative(now.addingTimeInterval(-20 * 86400), now: now).hasSuffix("d"))
    }

    func testDayGroup() {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "UTC")!
        XCTAssertEqual(Formatting.dayGroup(now.addingTimeInterval(3600), now: now.addingTimeInterval(7200), calendar: cal), "Today")
        XCTAssertEqual(Formatting.dayGroup(now.addingTimeInterval(-3600), now: now.addingTimeInterval(7200), calendar: cal), "Yesterday")
        XCTAssertNotEqual(Formatting.dayGroup(now.addingTimeInterval(-10 * 86400), now: now, calendar: cal), "Today")
    }
}
