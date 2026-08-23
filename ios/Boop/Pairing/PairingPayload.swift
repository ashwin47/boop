import Foundation

/// The JSON encoded in the pairing QR code shown by the web UI.
struct PairingPayload: Codable, Equatable, Sendable {
    let version: Int
    let server: String
    let token: String

    enum ParseError: Error, LocalizedError, Equatable {
        case notJSON
        case unsupportedVersion(Int)
        case badServer
        case badToken

        var errorDescription: String? {
            switch self {
            case .notJSON: "That code is not a Boop pairing code."
            case .unsupportedVersion(let v): "Pairing code version \(v) is not supported by this app."
            case .badServer: "The pairing code has an invalid server address."
            case .badToken: "The pairing code has an invalid token."
            }
        }
    }

    static func parse(_ text: String) throws -> PairingPayload {
        guard let data = text.data(using: .utf8), let p = try? JSONDecoder().decode(PairingPayload.self, from: data) else {
            throw ParseError.notJSON
        }
        try p.validate()
        return p
    }

    /// Manual entry: server address plus token typed by hand.
    static func manual(server: String, token: String) throws -> PairingPayload {
        var s = server.trimmingCharacters(in: .whitespacesAndNewlines)
        if !s.contains("://") { s = "https://" + s }
        let p = PairingPayload(version: 1, server: s, token: token.trimmingCharacters(in: .whitespacesAndNewlines))
        try p.validate()
        return p
    }

    func validate() throws {
        guard version == 1 else { throw ParseError.unsupportedVersion(version) }
        guard serverURL != nil else { throw ParseError.badServer }
        guard token.hasPrefix("pair_"), token.count > 10 else { throw ParseError.badToken }
    }

    var serverURL: URL? {
        guard let url = URL(string: server), let scheme = url.scheme?.lowercased(), url.host() != nil else { return nil }
        guard scheme == "https" || scheme == "http" else { return nil }
        var s = server
        while s.hasSuffix("/") { s.removeLast() }
        return URL(string: s)
    }
}
