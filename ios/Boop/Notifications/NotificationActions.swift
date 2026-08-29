import Foundation

/// The action buttons carried in a push payload. Shared by the app (which
/// opens the URL when a button is tapped) and the notification service
/// extension (which registers the buttons before the alert is shown).
enum NotificationActions {
    /// The `aps.category` the server sets on pushes that carry actions.
    static let serverCategory = "boop.event.actions"
    /// Prefix of the per-event category the extension registers.
    static let categoryPrefix = "boop.event.actions."
    /// Prefix of action identifiers; the suffix is the index into `actions`.
    static let actionPrefix = "boop.action."
    static let maxActions = 3

    /// Actions from a push `userInfo` (`actions: [{label, url}]`). Malformed entries are dropped.
    static func parse(_ userInfo: [AnyHashable: Any]) -> [EventAction] {
        guard let raw = userInfo["actions"] as? [Any] else { return [] }
        var out: [EventAction] = []
        for item in raw {
            if out.count == maxActions { break }
            guard let dict = item as? [String: Any],
                  let label = (dict["label"] as? String)?.trimmingCharacters(in: .whitespaces), !label.isEmpty,
                  let url = (dict["url"] as? String)?.trimmingCharacters(in: .whitespaces), URL(string: url) != nil
            else { continue }
            out.append(EventAction(label: label, url: url))
        }
        return out
    }

    /// Category identifier for one event's buttons.
    static func categoryID(eventID: String) -> String { categoryPrefix + eventID }

    static func actionID(index: Int) -> String { actionPrefix + String(index) }

    /// The index encoded in an action identifier, if it is one of ours.
    static func actionIndex(_ identifier: String) -> Int? {
        guard identifier.hasPrefix(actionPrefix) else { return nil }
        return Int(identifier.dropFirst(actionPrefix.count))
    }

    /// The URL a tapped action should open, given the push's userInfo.
    static func url(for identifier: String, userInfo: [AnyHashable: Any]) -> URL? {
        guard let i = actionIndex(identifier) else { return nil }
        let actions = parse(userInfo)
        guard actions.indices.contains(i) else { return nil }
        return actions[i].destination
    }
}
