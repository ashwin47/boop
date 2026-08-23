import XCTest
@testable import Boop

/// Intercepts URLSession traffic so the client can be tested without a server.
final class StubProtocol: URLProtocol {
    nonisolated(unsafe) static var handler: ((URLRequest) -> (Int, Data))?

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func startLoading() {
        guard let handler = Self.handler else { return }
        let (status, data) = handler(request)
        let resp = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: nil, headerFields: ["Content-Type": "application/json"])!
        client?.urlProtocol(self, didReceive: resp, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
    override func stopLoading() {}
}

final class APIClientTests: XCTestCase {
    var client: APIClient!

    override func setUp() {
        let cfg = URLSessionConfiguration.ephemeral
        cfg.protocolClasses = [StubProtocol.self]
        client = APIClient(baseURL: URL(string: "https://boop.test")!, credential: "boop_dev_secret", session: URLSession(configuration: cfg))
    }

    func testEventsSendsBearerAndQuery() async throws {
        nonisolated(unsafe) var captured: URLRequest?
        StubProtocol.handler = { req in
            captured = req
            return (200, Data(#"{"events":[],"next_cursor":"evt_9"}"#.utf8))
        }
        let page = try await client.events(project: "prj_1", level: .error, before: "evt_5", limit: 20)
        XCTAssertEqual(page.nextCursor, "evt_9")
        XCTAssertEqual(captured?.value(forHTTPHeaderField: "Authorization"), "Bearer boop_dev_secret")
        let q = URLComponents(url: captured!.url!, resolvingAgainstBaseURL: false)!.queryItems!
        XCTAssertEqual(q.first { $0.name == "project" }?.value, "prj_1")
        XCTAssertEqual(q.first { $0.name == "level" }?.value, "error")
        XCTAssertEqual(q.first { $0.name == "before" }?.value, "evt_5")
        XCTAssertEqual(q.first { $0.name == "limit" }?.value, "20")
        XCTAssertEqual(captured?.url?.path(), "/api/v1/events")
    }

    func testPairingExchangeIsUnauthenticatedAndPostsJSON() async throws {
        nonisolated(unsafe) var captured: URLRequest?
        StubProtocol.handler = { req in
            captured = req
            return (201, Data(#"{"device":{"id":"dev_1","name":"Phone","push_registered":false,"platform":"ios","app_bundle_id":"","last_seen_at":null,"created_at":"2026-08-28T12:00:00Z"},"credential":"boop_dev_new"}"#.utf8))
        }
        let r = try await client.exchangePairing(token: "pair_x", name: "Phone")
        XCTAssertEqual(r.credential, "boop_dev_new")
        XCTAssertEqual(r.device.id, "dev_1")
        XCTAssertNil(captured?.value(forHTTPHeaderField: "Authorization"))
        XCTAssertEqual(captured?.httpMethod, "POST")
        let body = try XCTUnwrap(captured?.httpBody ?? captured?.httpBodyStream.map { s in
            s.open(); defer { s.close() }
            var d = Data(); var buf = [UInt8](repeating: 0, count: 1024)
            while s.hasBytesAvailable { let n = s.read(&buf, maxLength: 1024); if n > 0 { d.append(buf, count: n) } else { break } }
            return d
        })
        let dict = try JSONSerialization.jsonObject(with: body) as? [String: String]
        XCTAssertEqual(dict?["token"], "pair_x")
        XCTAssertEqual(dict?["platform"], "ios")
    }

    func testErrorsSurfaceServerMessage() async {
        StubProtocol.handler = { _ in (401, Data(#"{"error":"unauthorized","message":"invalid device credential"}"#.utf8)) }
        do {
            _ = try await client.event(id: "evt_1")
            XCTFail("expected error")
        } catch let e as APIError {
            XCTAssertTrue(e.isUnauthorized)
            XCTAssertEqual(e.message, "invalid device credential")
        } catch {
            XCTFail("wrong error type \(error)")
        }
    }
}
