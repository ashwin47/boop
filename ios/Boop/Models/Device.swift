import Foundation

struct Device: Codable, Identifiable, Sendable, Equatable {
    let id: String
    var name: String
    var pushRegistered: Bool
    var platform: String
    var appBundleID: String
    var lastSeenAt: Date?
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id, name, platform
        case pushRegistered = "push_registered"
        case appBundleID = "app_bundle_id"
        case lastSeenAt = "last_seen_at"
        case createdAt = "created_at"
    }
}

struct PairingResult: Codable, Sendable {
    let device: Device
    let credential: String
}

struct Project: Codable, Identifiable, Sendable, Hashable {
    let id: String
    let name: String
    let slug: String
    let icon: String
}
