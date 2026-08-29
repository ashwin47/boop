import Foundation

/// A button attached to an event: on the notification and in the detail view.
struct EventAction: Codable, Hashable, Sendable {
    let label: String
    let url: String

    var destination: URL? { URL(string: url) }
}
