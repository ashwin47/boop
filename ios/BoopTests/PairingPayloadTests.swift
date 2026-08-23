import XCTest
@testable import Boop

final class PairingPayloadTests: XCTestCase {
    func testParsesQRPayload() throws {
        let p = try PairingPayload.parse(#"{"version":1,"server":"https://boop.example.com/","token":"pair_abcdefghijklmnop"}"#)
        XCTAssertEqual(p.serverURL?.absoluteString, "https://boop.example.com")
        XCTAssertEqual(p.token, "pair_abcdefghijklmnop")
    }

    func testRejectsGarbage() {
        XCTAssertThrowsError(try PairingPayload.parse("hello")) { XCTAssertEqual($0 as? PairingPayload.ParseError, .notJSON) }
        XCTAssertThrowsError(try PairingPayload.parse(#"{"version":2,"server":"https://x.y","token":"pair_abcdefghijk"}"#)) {
            XCTAssertEqual($0 as? PairingPayload.ParseError, .unsupportedVersion(2))
        }
        XCTAssertThrowsError(try PairingPayload.parse(#"{"version":1,"server":"ftp://x.y","token":"pair_abcdefghijk"}"#)) {
            XCTAssertEqual($0 as? PairingPayload.ParseError, .badServer)
        }
        XCTAssertThrowsError(try PairingPayload.parse(#"{"version":1,"server":"https://x.y","token":"nope"}"#)) {
            XCTAssertEqual($0 as? PairingPayload.ParseError, .badToken)
        }
    }

    func testManualEntryAddsScheme() throws {
        let p = try PairingPayload.manual(server: " boop.example.com ", token: " pair_abcdefghijklmnop\n")
        XCTAssertEqual(p.serverURL?.absoluteString, "https://boop.example.com")
        XCTAssertEqual(p.token, "pair_abcdefghijklmnop")
        let local = try PairingPayload.manual(server: "http://192.168.1.10:8080", token: "pair_abcdefghijklmnop")
        XCTAssertEqual(local.serverURL?.absoluteString, "http://192.168.1.10:8080")
    }
}
